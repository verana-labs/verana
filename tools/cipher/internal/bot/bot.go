package bot

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/verana-labs/verana/tools/cipher/internal/config"
	"github.com/verana-labs/verana/tools/cipher/internal/executor"
	"github.com/verana-labs/verana/tools/cipher/internal/state"
)

// WatcherRef is set by main.go so bot can toggle auto-fix.
var WatcherRef interface{ SetAutoFix(bool) }

type Bot struct {
	cfg     *config.Config
	exec    *executor.Executor
	state   *state.State
	session *discordgo.Session
}

func New(cfg *config.Config, exec *executor.Executor, st *state.State) (*Bot, error) {
	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("discordgo: %w", err)
	}
	b := &Bot{cfg: cfg, exec: exec, state: st, session: dg}
	dg.AddHandler(b.onMessage)
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages
	return b, nil
}

func (b *Bot) Open() error  { return b.session.Open() }
func (b *Bot) Close() error { return b.session.Close() }

func (b *Bot) Notify(msg string) {
	if _, err := b.session.ChannelMessageSend(b.cfg.DiscordChannelID, msg); err != nil {
		log.Printf("[cipher] notify error: %v", err)
	}
}

func (b *Bot) send(chanID, msg string) {
	for len(msg) > 0 {
		n := 1900
		if len(msg) < n {
			n = len(msg)
		}
		b.session.ChannelMessageSend(chanID, msg[:n])
		msg = msg[n:]
	}
}

var reNum = regexp.MustCompile(`^#?(\d+)$`)

func parseNum(s string) int {
	m := reNum.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0
	}
	var n int
	fmt.Sscanf(m[1], "%d", &n)
	return n
}

func (b *Bot) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.ChannelID != b.cfg.DiscordChannelID {
		return
	}
	if !strings.HasPrefix(m.Content, "!cipher ") {
		return
	}

	content := strings.TrimPrefix(m.Content, "!cipher ")
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]
	rest := strings.TrimSpace(content[len(cmd):])

	r := func(msg string) { b.send(m.ChannelID, msg) }

	switch cmd {
	case "help":
		b.send(m.ChannelID, helpText)

	case "status":
		b.exec.CmdStatus(r)

	case "implement":
		arg := strings.Trim(rest, `"' `)
		if arg == "" {
			r("Usage: `!cipher implement #42` or `!cipher implement \"desc\"`")
			return
		}
		if n := parseNum(arg); n > 0 {
			if !b.exec.CmdImplementIssue(n, r) {
				r("⏳ Task already running. Use `!cipher cancel` first.")
			}
		} else {
			if !b.exec.CmdImplementFreeform(arg, r) {
				r("⏳ Task already running.")
			}
		}

	case "fix-ci":
		if n := parseNum(rest); n > 0 {
			if !b.exec.CmdFixCI(n, r) {
				r("⏳ Task already running.")
			}
		} else {
			r("Usage: `!cipher fix-ci #15`")
		}

	case "review":
		if n := parseNum(rest); n > 0 {
			if !b.exec.CmdAddressReviews(n, r) {
				r("⏳ Task already running.")
			}
		} else {
			r("Usage: `!cipher review #15`")
		}

	case "rebase":
		if n := parseNum(rest); n > 0 {
			if !b.exec.CmdRebase(n, r) {
				r("⏳ Task already running.")
			}
		} else {
			r("Usage: `!cipher rebase #15`")
		}

	case "run":
		task := strings.Trim(rest, `"' `)
		if task == "" {
			r("Usage: `!cipher run \"refactor x/trustregistry…\"`")
			return
		}
		if !b.exec.CmdRun(task, r) {
			r("⏳ Task already running.")
		}

	case "review-pr":
		if n := parseNum(rest); n > 0 {
			if !b.exec.CmdReviewPR(n, r) {
				r("⏳ Task already running.")
			}
		} else {
			r("Usage: `!cipher review-pr #15`")
		}

	case "check-spec":
		if n := parseNum(rest); n > 0 {
			if !b.exec.CmdCheckSpec(n, r) {
				r("⏳ Task already running.")
			}
		} else {
			r("Usage: `!cipher check-spec #15`")
		}

	case "feedback":
		fbParts := strings.Fields(rest)
		if len(fbParts) < 2 {
			r("Usage: `!cipher feedback #15 good/bad/mixed [optional notes]`")
			return
		}
		n := parseNum(fbParts[0])
		if n == 0 {
			r("Usage: `!cipher feedback #15 good/bad/mixed [optional notes]`")
			return
		}
		fb := strings.ToLower(fbParts[1])
		if fb != "good" && fb != "bad" && fb != "mixed" {
			r("Feedback must be `good`, `bad`, or `mixed`.")
			return
		}
		notes := ""
		if len(fbParts) > 2 {
			notes = strings.Join(fbParts[2:], " ")
		}
		b.exec.CmdFeedback(n, fb, notes, r)

	case "review-history":
		b.exec.CmdReviewHistory(r)

	case "diff":
		if n := parseNum(rest); n > 0 {
			b.exec.CmdDiff(n, r)
		} else {
			r("Usage: `!cipher diff #15`")
		}

	case "logs":
		branch := strings.TrimSpace(rest)
		if branch == "" {
			tasks := b.state.RecentTasks(1)
			if len(tasks) > 0 {
				branch = tasks[0].Branch
			}
		}
		if branch == "" {
			r("Usage: `!cipher logs cipher-issue-42`")
			return
		}
		r(b.exec.GetLogTail(branch, 50))

	case "cancel":
		b.exec.Cancel()
		r("🛑 Cancel signal sent.")

	case "auto":
		mode := strings.ToLower(strings.TrimSpace(rest))
		if WatcherRef == nil {
			r("Watcher not available.")
			return
		}
		switch mode {
		case "on", "true", "yes", "1":
			WatcherRef.SetAutoFix(true)
			r("✅ Auto-fix **ON** — CI, reviews, rebases handled automatically.")
		case "off", "false", "no", "0":
			WatcherRef.SetAutoFix(false)
			r("🔕 Auto-fix **OFF** — notify only, manual trigger required.")
		default:
			r("Usage: `!cipher auto on` or `!cipher auto off`")
		}

	default:
		r(fmt.Sprintf("Unknown command `%s`. Type `!cipher help`.", cmd))
	}
}

const helpText = `**🔐 Cipher — Autonomous AI Developer for Verana**

` + "```" + `
!cipher implement #42              implement GitHub issue #42
!cipher implement "description"    create issue + implement
!cipher fix-ci #15                 fix CI failures on PR #15
!cipher review #15                 address reviewer feedback on PR #15
!cipher rebase #15                 rebase PR #15 onto main
!cipher run "do X to codebase"     free-form Claude Code task → PR
!cipher review-pr #15              AI code review of PR #15
!cipher check-spec #15             check spec compliance of PR #15
!cipher feedback #15 good/bad      rate a review (good/bad/mixed)
!cipher review-history             past reviews + feedback
!cipher status                     open PRs + CI + attempt counts
!cipher diff #15                   PR diff stat
!cipher logs [branch]              tail of last task log
!cipher cancel                     kill current task
!cipher auto on/off                toggle auto-fix (default: on)
!cipher help                       this message
` + "```" + `
Auto-fix ON: Cipher auto-fixes CI failures, reviewer requests, stale rebases.`
