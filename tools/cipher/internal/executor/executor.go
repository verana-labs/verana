package executor

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/verana-labs/verana/tools/cipher/internal/config"
	gh "github.com/verana-labs/verana/tools/cipher/internal/github"
	"github.com/verana-labs/verana/tools/cipher/internal/prompt"
	"github.com/verana-labs/verana/tools/cipher/internal/state"
)

type Reporter func(string)

type Executor struct {
	cfg    *config.Config
	gh     *gh.Client
	state  *state.State
	mu     sync.Mutex
	cancel context.CancelFunc
}

func New(cfg *config.Config, ghc *gh.Client, st *state.State) *Executor {
	return &Executor{cfg: cfg, gh: ghc, state: st}
}

func (e *Executor) Cancel() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancel != nil {
		e.cancel()
	}
}

func (e *Executor) start(fn func(context.Context, Reporter) error, report Reporter) bool {
	if !e.mu.TryLock() {
		return false
	}
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	go func() {
		defer e.mu.Unlock()
		defer cancel()
		if err := fn(ctx, report); err != nil && ctx.Err() == nil {
			report(fmt.Sprintf("❌ %v", err))
		}
	}()
	return true
}

func (e *Executor) CmdImplementIssue(n int, r Reporter) bool {
	return e.start(func(ctx context.Context, r Reporter) error {
		issue, err := e.gh.GetIssue(n)
		if err != nil {
			return err
		}
		r(fmt.Sprintf("🔍 Issue #%d: **%s**", issue.Number, issue.Title))
		return e.implement(ctx, issue, r)
	}, r)
}

func (e *Executor) CmdImplementFreeform(desc string, r Reporter) bool {
	return e.start(func(ctx context.Context, r Reporter) error {
		r("📝 Creating GitHub issue…")
		issue, err := e.gh.CreateIssue(truncate(desc, 120), "Auto-created by Cipher.\n\n"+desc, []string{e.cfg.BotLabel})
		if err != nil {
			return err
		}
		r(fmt.Sprintf("✅ Created issue #%d", issue.Number))
		return e.implement(ctx, issue, r)
	}, r)
}

func (e *Executor) CmdFixCI(n int, r Reporter) bool {
	return e.start(func(ctx context.Context, r Reporter) error {
		pr, err := e.gh.GetPR(n)
		if err != nil {
			return err
		}
		summary, err := e.gh.GetCIFailureSummary(pr.HeadSHA)
		if err != nil {
			return err
		}
		if summary == "" {
			r("✅ No CI failures found.")
			return nil
		}
		return e.fixCI(ctx, pr, summary, r)
	}, r)
}

func (e *Executor) CmdAddressReviews(n int, r Reporter) bool {
	return e.start(func(ctx context.Context, r Reporter) error {
		pr, err := e.gh.GetPR(n)
		if err != nil {
			return err
		}
		feedback, err := e.gh.FormatReviewFeedback(n)
		if err != nil {
			return err
		}
		if feedback == "" {
			r("✅ No pending reviewer feedback.")
			return nil
		}
		return e.addressReviews(ctx, pr, feedback, r)
	}, r)
}

func (e *Executor) CmdRebase(n int, r Reporter) bool {
	return e.start(func(ctx context.Context, r Reporter) error {
		pr, err := e.gh.GetPR(n)
		if err != nil {
			return err
		}
		r(fmt.Sprintf("📐 Rebasing PR #%d onto %s…", n, e.cfg.BaseBranch))
		return e.rebase(ctx, pr, r)
	}, r)
}

func (e *Executor) CmdRun(task string, r Reporter) bool {
	return e.start(func(ctx context.Context, r Reporter) error {
		branch := fmt.Sprintf("cipher-task-%s-%d", prompt.Slugify(task, 30), time.Now().Unix()%10000)
		r(fmt.Sprintf("🌿 Branch: %s", branch))
		wt := e.createWorktree(branch)
		commits, _ := e.gh.GetRecentCommits(e.cfg.BaseBranch, 20)
		p := prompt.Freeform(e.cfg, task, commits)
		r("🤖 Running Cipher (Claude Code)…")
		if _, err := e.runClaude(ctx, wt, p, branch); err != nil {
			return err
		}
		if e.gitPush(wt, branch, false) {
			pr, err := e.gh.CreatePR(truncate("feat: "+task, 80), "🔐 Free-form task by **Cipher**.\n\n"+task, branch, e.cfg.BaseBranch)
			if err != nil {
				return err
			}
			r(fmt.Sprintf("✅ PR #%d: %s", pr.Number, pr.URL))
			e.state.RecordTask(state.TaskRecord{Type: "run", Branch: branch, PR: pr.Number, OK: true})
		} else {
			r("⚠️ No changes to push.")
			e.removeWorktree(branch)
		}
		return nil
	}, r)
}

func (e *Executor) CmdStatus(r Reporter) {
	prs, err := e.gh.GetBotPRs()
	if err != nil {
		r(fmt.Sprintf("❌ %v", err))
		return
	}
	if len(prs) == 0 {
		r("No open Cipher PRs.")
		return
	}
	lines := []string{fmt.Sprintf("**%d open Cipher PRs:**\n", len(prs))}
	for _, pr := range prs {
		ci := "✅"
		if e.gh.HasCIFailure(pr.HeadSHA) {
			ci = "❌"
		}
		crs, _ := e.gh.GetChangeRequests(pr.Number)
		crStr := ""
		if len(crs) > 0 {
			crStr = fmt.Sprintf(" | 🔴 %d CR(s)", len(crs))
		}
		att := e.state.GetCIAttempts(pr.Branch)
		attStr := ""
		if att > 0 {
			attStr = fmt.Sprintf(" | CI: %d/%d", att, e.cfg.MaxCIAttempts)
		}
		lines = append(lines, fmt.Sprintf("• PR #%d `%s` %s%s%s\n  %s", pr.Number, pr.Branch, ci, crStr, attStr, pr.URL))
	}
	r(strings.Join(lines, "\n"))
}

func (e *Executor) CmdReviewPR(n int, r Reporter) bool {
	return e.start(func(ctx context.Context, r Reporter) error {
		pr, err := e.gh.GetPR(n)
		if err != nil {
			return err
		}
		diff, err := e.gh.GetPRDiff(n)
		if err != nil {
			return err
		}
		if diff == "" {
			r("No diff found for PR.")
			return nil
		}
		r(fmt.Sprintf("🔍 Reviewing PR #%d: **%s**…", n, pr.Title))
		p := prompt.ReviewPR(e.cfg, n, pr.Title, diff)
		out, err := e.runClaudeCapture(ctx, p)
		if err != nil {
			return err
		}
		r(fmt.Sprintf("**Review of PR #%d — %s:**\n\n%s", n, pr.Title, out))
		return nil
	}, r)
}

func (e *Executor) CmdCheckSpec(n int, r Reporter) bool {
	return e.start(func(ctx context.Context, r Reporter) error {
		pr, err := e.gh.GetPR(n)
		if err != nil {
			return err
		}
		diff, err := e.gh.GetPRDiff(n)
		if err != nil {
			return err
		}
		if diff == "" {
			r("No diff found for PR.")
			return nil
		}
		r(fmt.Sprintf("📋 Checking spec compliance for PR #%d: **%s**…", n, pr.Title))
		p := prompt.CheckSpec(e.cfg, n, pr.Title, diff)
		out, err := e.runClaudeCapture(ctx, p)
		if err != nil {
			return err
		}
		r(fmt.Sprintf("**Spec Check — PR #%d — %s:**\n\n%s", n, pr.Title, out))
		return nil
	}, r)
}

func (e *Executor) CmdDiff(n int, r Reporter) {
	pr, err := e.gh.GetPR(n)
	if err != nil {
		r(fmt.Sprintf("❌ %v", err))
		return
	}
	out, _ := e.git(e.cfg.RepoPath, "diff", fmt.Sprintf("origin/%s...origin/%s", e.cfg.BaseBranch, pr.Branch), "--stat")
	if len(out) > 1800 {
		out = out[:1800]
	}
	r(fmt.Sprintf("**PR #%d diff:**\n```\n%s\n```", n, out))
}

func (e *Executor) GetLogTail(branch string, lines int) string {
	matches, _ := filepath.Glob(filepath.Join(e.cfg.LogDir, strings.ReplaceAll(branch, "/", "_")+"_*.log"))
	if len(matches) == 0 {
		return "No logs found."
	}
	b, _ := os.ReadFile(matches[len(matches)-1])
	all := strings.Split(string(b), "\n")
	if len(all) > lines {
		all = all[len(all)-lines:]
	}
	tail := strings.Join(all, "\n")
	if len(tail) > 1800 {
		tail = tail[len(tail)-1800:]
	}
	return fmt.Sprintf("**Log:** `%s`\n```\n%s\n```", filepath.Base(matches[len(matches)-1]), tail)
}

// Watcher-triggered (non-blocking, ignore if already running)
func (e *Executor) AutoFixCI(pr *gh.PR, report string, r Reporter) {
	e.start(func(ctx context.Context, r Reporter) error { return e.fixCI(ctx, pr, report, r) }, r)
}
func (e *Executor) AutoAddressReviews(pr *gh.PR, fb string, r Reporter) {
	e.start(func(ctx context.Context, r Reporter) error { return e.addressReviews(ctx, pr, fb, r) }, r)
}
func (e *Executor) AutoRebase(pr *gh.PR, r Reporter) {
	e.start(func(ctx context.Context, r Reporter) error { return e.rebase(ctx, pr, r) }, r)
}

// Core implementations

func (e *Executor) implement(ctx context.Context, issue *gh.Issue, r Reporter) error {
	branch := fmt.Sprintf("cipher-issue-%d", issue.Number)
	existing, _ := e.gh.GetBotPRs()
	for _, pr := range existing {
		if pr.Branch == branch {
			r(fmt.Sprintf("⚠️ Already have PR for issue #%d.", issue.Number))
			return nil
		}
	}
	if len(existing) >= e.cfg.MaxOpenPRs {
		r(fmt.Sprintf("⚠️ At PR limit (%d). Merge existing PRs first.", e.cfg.MaxOpenPRs))
		return nil
	}
	r(fmt.Sprintf("🌿 Branch: %s", branch))
	wt := e.createWorktree(branch)
	_ = e.gh.AddLabel(issue.Number, "cipher-wip")
	_ = e.gh.PostIssueComment(issue.Number, fmt.Sprintf("🔐 **Cipher** implementing on `%s`.", branch))
	commits, _ := e.gh.GetRecentCommits(e.cfg.BaseBranch, 20)
	p := prompt.ImplementIssue(e.cfg, issue.Number, issue.Title, issue.Body, commits, branch)
	r(fmt.Sprintf("🤖 Running Claude Code on issue #%d…", issue.Number))
	if _, err := e.runClaude(ctx, wt, p, branch); err != nil {
		_ = e.gh.RemoveLabel(issue.Number, "cipher-wip")
		_ = e.gh.PostIssueComment(issue.Number, "🔐 **Cipher** error. Manual help needed.")
		e.removeWorktree(branch)
		return err
	}
	if e.gitPush(wt, branch, false) {
		pr, err := e.gh.CreatePR(fmt.Sprintf("feat: %s (#%d)", issue.Title, issue.Number),
			fmt.Sprintf("## Summary\nCloses #%d\n\n🔐 Implemented by **Cipher**. *Review carefully before merging.*", issue.Number),
			branch, e.cfg.BaseBranch)
		if err != nil {
			return err
		}
		_ = e.gh.RemoveLabel(issue.Number, "cipher-wip")
		_ = e.gh.AddLabel(issue.Number, "cipher-done")
		_ = e.gh.PostIssueComment(issue.Number, fmt.Sprintf("🔐 **Cipher** opened PR #%d: %s", pr.Number, pr.URL))
		r(fmt.Sprintf("✅ PR #%d: %s", pr.Number, pr.URL))
		e.state.RecordTask(state.TaskRecord{Type: "implement", Issue: issue.Number, PR: pr.Number, Branch: branch, OK: true})
	} else {
		r("⚠️ No code changes — issue needs manual triage.")
		_ = e.gh.RemoveLabel(issue.Number, "cipher-wip")
		e.removeWorktree(branch)
	}
	return nil
}

func (e *Executor) fixCI(ctx context.Context, pr *gh.PR, failReport string, r Reporter) error {
	attempts := e.state.GetCIAttempts(pr.Branch)
	if attempts >= e.cfg.MaxCIAttempts {
		_ = e.gh.PostPRComment(pr.Number, fmt.Sprintf("🔐 **Cipher** exhausted %d CI fix attempts. Manual help needed. 🆘", e.cfg.MaxCIAttempts))
		r(fmt.Sprintf("🛑 PR #%d: max CI attempts reached.", pr.Number))
		return nil
	}
	att := e.state.IncrementCIAttempts(pr.Branch)
	r(fmt.Sprintf("🔧 CI fix PR #%d — attempt %d/%d…", pr.Number, att, e.cfg.MaxCIAttempts))
	wt := e.ensureWorktree(pr.Branch)
	p := prompt.FixCI(e.cfg, pr.Number, pr.Branch, failReport, att, e.cfg.MaxCIAttempts)
	if _, err := e.runClaude(ctx, wt, p, pr.Branch); err != nil {
		_ = e.gh.PostPRComment(pr.Number, fmt.Sprintf("🔐 **Cipher** CI fix attempt %d failed.", att))
		return err
	}
	if e.gitPush(wt, pr.Branch, false) {
		e.state.ResetCIAttempts(pr.Branch)
		_ = e.gh.PostPRComment(pr.Number, fmt.Sprintf("🔐 **Cipher** CI fix pushed (attempt %d/%d).", att, e.cfg.MaxCIAttempts))
		r(fmt.Sprintf("✅ CI fix pushed to PR #%d.", pr.Number))
	} else {
		r("⚠️ No changes produced.")
	}
	return nil
}

func (e *Executor) addressReviews(ctx context.Context, pr *gh.PR, feedback string, r Reporter) error {
	last := e.gh.GetPRLastBotComment(pr.Number)
	if strings.Contains(strings.ToLower(last), "addressing review") {
		r(fmt.Sprintf("⚠️ Already processing reviews on PR #%d.", pr.Number))
		return nil
	}
	_ = e.gh.PostPRComment(pr.Number, "🔐 **Cipher** addressing review feedback…")
	r(fmt.Sprintf("🔧 Addressing reviews on PR #%d…", pr.Number))
	wt := e.ensureWorktree(pr.Branch)
	p := prompt.AddressReviews(e.cfg, pr.Number, pr.Branch, feedback)
	if _, err := e.runClaude(ctx, wt, p, pr.Branch); err != nil {
		_ = e.gh.PostPRComment(pr.Number, "🔐 **Cipher** failed to address feedback. Manual help needed.")
		return err
	}
	if e.gitPush(wt, pr.Branch, false) {
		_ = e.gh.PostPRComment(pr.Number, "🔐 **Cipher** addressed all feedback. Please re-review! 🙏")
		r(fmt.Sprintf("✅ Reviews addressed on PR #%d.", pr.Number))
	} else {
		r("⚠️ No changes made.")
	}
	return nil
}

func (e *Executor) rebase(ctx context.Context, pr *gh.PR, r Reporter) error {
	wt := e.ensureWorktree(pr.Branch)
	e.git(wt, "fetch", "origin")
	if _, err := e.git(wt, "rebase", "origin/"+e.cfg.BaseBranch); err == nil {
		e.gitPush(wt, pr.Branch, true)
		r(fmt.Sprintf("✅ PR #%d rebased.", pr.Number))
		return nil
	}
	e.git(wt, "rebase", "--abort")
	r(fmt.Sprintf("⚠️ Rebase conflict on PR #%d — asking Claude Code…", pr.Number))
	p := prompt.RebaseResolve(e.cfg, pr.Branch, e.cfg.BaseBranch)
	if _, err := e.runClaude(ctx, wt, p, pr.Branch); err != nil {
		_ = e.gh.PostPRComment(pr.Number, "🔐 **Cipher** could not resolve conflicts. Manual rebase needed.")
		return err
	}
	if e.gitPush(wt, pr.Branch, true) {
		_ = e.gh.PostPRComment(pr.Number, fmt.Sprintf("🔐 **Cipher** rebased onto %s (conflict resolved).", e.cfg.BaseBranch))
		r(fmt.Sprintf("✅ Conflict resolved on PR #%d.", pr.Number))
	}
	return nil
}

func (e *Executor) runClaude(ctx context.Context, worktree, p, branch string) (string, error) {
	logPath := filepath.Join(e.cfg.LogDir,
		fmt.Sprintf("%s_%d.log", strings.ReplaceAll(branch, "/", "_"), time.Now().Unix()))

	var args []string
	if e.cfg.UseDocker {
		args = []string{"docker", "run", "--rm",
			"-v", worktree + ":/workspace",
			"-v", os.Getenv("HOME") + "/.gitconfig:/root/.gitconfig:ro",
			"-e", "ANTHROPIC_API_KEY=" + e.cfg.AnthropicAPIKey,
			"-e", "GITHUB_TOKEN=" + e.cfg.GitHubToken,
			"-w", "/workspace",
			e.cfg.DockerImage,
		}
	} else {
		args = []string{e.cfg.ClaudeCmd}
	}
	args = append(args, "--allowedTools", "Edit,Write,Bash,Read,Glob,Grep,LS",
		"--dangerouslySkipPermissions", "-p", p)

	f, err := os.Create(logPath)
	if err != nil {
		return logPath, err
	}
	defer f.Close()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if !e.cfg.UseDocker {
		cmd.Dir = worktree
	}
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY="+e.cfg.AnthropicAPIKey)

	if err := cmd.Run(); err != nil {
		return logPath, fmt.Errorf("claude: %w (log: %s)", err, logPath)
	}
	return logPath, nil
}

func (e *Executor) runClaudeCapture(ctx context.Context, p string) (string, error) {
	args := []string{e.cfg.ClaudeCmd, "--print", "-p", p}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY="+e.cfg.AnthropicAPIKey)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("claude: %w\n%s", err, string(out))
	}
	result := string(out)
	if len(result) > 3800 {
		result = result[:3800] + "\n\n… (truncated)"
	}
	return result, nil
}

func (e *Executor) git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (e *Executor) createWorktree(branch string) string {
	path := filepath.Join(e.cfg.WorktreeBase, branch)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	os.MkdirAll(e.cfg.WorktreeBase, 0o755)
	if _, err := e.git(e.cfg.RepoPath, "worktree", "add", path, "-b", branch, "origin/"+e.cfg.BaseBranch); err != nil {
		log.Printf("[cipher] worktree add failed, trying fetch: %v", err)
		e.git(e.cfg.RepoPath, "fetch", "origin", branch)
		e.git(e.cfg.RepoPath, "worktree", "add", path, branch)
	}
	return path
}

func (e *Executor) ensureWorktree(branch string) string {
	path := filepath.Join(e.cfg.WorktreeBase, branch)
	if _, err := os.Stat(path); err == nil {
		e.git(path, "fetch", "origin")
		e.git(path, "reset", "--hard", "origin/"+branch)
		return path
	}
	return e.createWorktree(branch)
}

func (e *Executor) removeWorktree(branch string) {
	path := filepath.Join(e.cfg.WorktreeBase, branch)
	e.git(e.cfg.RepoPath, "worktree", "remove", "--force", path)
	e.git(e.cfg.RepoPath, "worktree", "prune")
}

func (e *Executor) gitPush(worktree, branch string, force bool) bool {
	logOut, _ := e.git(worktree, "log", "origin/"+branch+"..HEAD", "--oneline")
	status, _ := e.git(worktree, "status", "--porcelain")
	if strings.TrimSpace(logOut) == "" && strings.TrimSpace(status) == "" {
		return false
	}
	remoteURL := fmt.Sprintf("https://%s:%s@github.com/%s/%s.git",
		e.cfg.BotUsername, e.cfg.GitHubToken, e.cfg.RepoOwner, e.cfg.RepoName)
	e.git(worktree, "remote", "set-url", "origin", remoteURL)
	args := []string{"push", "-u", "origin", branch}
	if force {
		args = append(args, "--force-with-lease")
	}
	if _, err := e.git(worktree, args...); err != nil && force {
		args[len(args)-1] = "--force"
		_, err = e.git(worktree, args...)
		return err == nil
	} else {
		return err == nil
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
