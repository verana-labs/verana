package watcher

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/verana-labs/verana/tools/cipher/internal/config"
	"github.com/verana-labs/verana/tools/cipher/internal/executor"
	gh "github.com/verana-labs/verana/tools/cipher/internal/github"
	"github.com/verana-labs/verana/tools/cipher/internal/state"
)

type Watcher struct {
	cfg     *config.Config
	gh      *gh.Client
	state   *state.State
	exec    *executor.Executor
	notify  func(string)
	autoFix atomic.Bool
}

func New(cfg *config.Config, ghc *gh.Client, st *state.State, exec *executor.Executor, notify func(string)) *Watcher {
	w := &Watcher{cfg: cfg, gh: ghc, state: st, exec: exec, notify: notify}
	w.autoFix.Store(true)
	return w
}

func (w *Watcher) SetAutoFix(v bool) { w.autoFix.Store(v) }

func (w *Watcher) Start() {
	log.Printf("[cipher] watcher started (interval=%ds)", w.cfg.WatcherInterval)
	for {
		if err := w.poll(); err != nil {
			log.Printf("[cipher] watcher poll error: %v", err)
		}
		time.Sleep(time.Duration(w.cfg.WatcherInterval) * time.Second)
	}
}

func (w *Watcher) poll() error {
	prs, err := w.gh.GetBotPRs()
	if err != nil {
		return err
	}
	for i := range prs {
		w.checkCI(&prs[i])
		w.checkReviews(&prs[i])
		w.checkRebase(&prs[i])
	}
	return nil
}

func (w *Watcher) checkCI(pr *gh.PR) {
	runs, _ := w.gh.GetCIRuns(pr.HeadSHA)
	var failed, allDone bool
	passed := true
	for _, r := range runs {
		if r.Conclusion == "failure" || r.Conclusion == "timed_out" {
			failed = true
		}
		if r.Status != "completed" {
			passed = false
		}
		allDone = true
	}

	cur := "pending"
	if failed {
		cur = "failure"
	} else if allDone && passed {
		cur = "success"
	}

	prev := w.state.GetKnownPRCI(pr.Number)
	if cur == prev {
		return
	}
	w.state.SetKnownPRCI(pr.Number, cur)

	switch cur {
	case "failure":
		att := w.state.GetCIAttempts(pr.Branch)
		if att >= w.cfg.MaxCIAttempts {
			w.notify(fmt.Sprintf("🛑 **PR #%d** CI failing — max attempts. Manual fix needed.\n%s", pr.Number, pr.URL))
			return
		}
		suffix := fmt.Sprintf("Use `!cipher fix-ci %d`.", pr.Number)
		if w.autoFix.Load() {
			suffix = "Auto-fixing…"
		}
		w.notify(fmt.Sprintf("❌ **PR #%d** (`%s`) CI failed — attempt %d/%d. %s", pr.Number, pr.Branch, att+1, w.cfg.MaxCIAttempts, suffix))
		if w.autoFix.Load() {
			summary, _ := w.gh.GetCIFailureSummary(pr.HeadSHA)
			go w.exec.AutoFixCI(pr, summary, w.notify)
		}
	case "success":
		if prev == "failure" {
			w.notify(fmt.Sprintf("✅ **PR #%d** CI now passing! %s", pr.Number, pr.URL))
		}
	}
}

func (w *Watcher) checkReviews(pr *gh.PR) {
	crs, _ := w.gh.GetChangeRequests(pr.Number)
	hasCR := len(crs) > 0
	prev := w.state.GetKnownPRReviews(pr.Number)
	if hasCR == prev {
		return
	}
	w.state.SetKnownPRReviews(pr.Number, hasCR)
	if hasCR {
		reviewers := ""
		for i, r := range crs {
			if i > 0 {
				reviewers += ", "
			}
			reviewers += r.User.Login
		}
		suffix := fmt.Sprintf("Use `!cipher review %d`.", pr.Number)
		if w.autoFix.Load() {
			suffix = "Auto-addressing…"
		}
		w.notify(fmt.Sprintf("🔴 **PR #%d** — changes by %s. %s", pr.Number, reviewers, suffix))
		if w.autoFix.Load() {
			if fb, err := w.gh.FormatReviewFeedback(pr.Number); err == nil && fb != "" {
				go w.exec.AutoAddressReviews(pr, fb, w.notify)
			}
		}
	} else {
		w.notify(fmt.Sprintf("👍 **PR #%d** — change requests resolved! %s", pr.Number, pr.URL))
	}
}

func (w *Watcher) checkRebase(pr *gh.PR) {
	if !w.gh.IsBranchBehind(pr.Branch, w.cfg.BaseBranch) {
		return
	}
	suffix := fmt.Sprintf("Use `!cipher rebase %d`.", pr.Number)
	if w.autoFix.Load() {
		suffix = "Auto-rebasing…"
	}
	w.notify(fmt.Sprintf("📐 **PR #%d** (`%s`) behind `%s`. %s", pr.Number, pr.Branch, w.cfg.BaseBranch, suffix))
	if w.autoFix.Load() {
		go w.exec.AutoRebase(pr, w.notify)
	}
}
