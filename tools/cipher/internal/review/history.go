package review

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Entry struct {
	PR        int       `json:"pr"`
	Title     string    `json:"title"`
	Type      string    `json:"type"` // "review-pr" or "check-spec"
	Findings  string    `json:"findings"`
	Feedback  string    `json:"feedback"` // "good", "bad", "mixed", ""
	Notes     string    `json:"notes,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type History struct {
	mu      sync.Mutex
	path    string
	Entries []Entry `json:"entries"`
}

func NewHistory(path string) *History {
	h := &History{path: path}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, h)
	}
	return h
}

func (h *History) save() {
	b, _ := json.MarshalIndent(h, "", "  ")
	_ = os.WriteFile(h.path, b, 0o644)
}

func (h *History) Record(pr int, title, reviewType, findings string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Entries = append(h.Entries, Entry{
		PR:        pr,
		Title:     title,
		Type:      reviewType,
		Findings:  findings,
		Timestamp: time.Now().UTC(),
	})
	if len(h.Entries) > 100 {
		h.Entries = h.Entries[len(h.Entries)-100:]
	}
	h.save()
}

func (h *History) SetFeedback(pr int, feedback, notes string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	found := false
	for i := len(h.Entries) - 1; i >= 0; i-- {
		if h.Entries[i].PR == pr {
			h.Entries[i].Feedback = feedback
			h.Entries[i].Notes = notes
			found = true
			break
		}
	}
	if found {
		h.save()
	}
	return found
}

// LearningContext returns a summary of past feedback for prompt injection.
func (h *History) LearningContext(reviewType string, maxEntries int) string {
	h.mu.Lock()
	defer h.mu.Unlock()

	var relevant []Entry
	for _, e := range h.Entries {
		if e.Feedback != "" && e.Type == reviewType {
			relevant = append(relevant, e)
		}
	}
	if len(relevant) == 0 {
		return ""
	}
	if len(relevant) > maxEntries {
		relevant = relevant[len(relevant)-maxEntries:]
	}

	var lines string
	for _, e := range relevant {
		quality := "GOOD"
		if e.Feedback == "bad" {
			quality = "BAD — avoid these patterns"
		} else if e.Feedback == "mixed" {
			quality = "MIXED — some findings were wrong"
		}
		lines += fmt.Sprintf("- PR #%d (%s) — Quality: %s", e.PR, e.Title, quality)
		if e.Notes != "" {
			lines += fmt.Sprintf(" — Notes: %s", e.Notes)
		}
		lines += "\n"
	}
	return lines
}

func (h *History) RecentSummary(n int) string {
	h.mu.Lock()
	defer h.mu.Unlock()

	start := 0
	if len(h.Entries) > n {
		start = len(h.Entries) - n
	}

	if len(h.Entries) == 0 {
		return "No review history yet."
	}

	var lines string
	for _, e := range h.Entries[start:] {
		fb := "⏳"
		switch e.Feedback {
		case "good":
			fb = "👍"
		case "bad":
			fb = "👎"
		case "mixed":
			fb = "🤷"
		}
		lines += fmt.Sprintf("• PR #%d `%s` [%s] %s %s\n",
			e.PR, e.Title, e.Type, fb, e.Timestamp.Format("2006-01-02"))
	}
	return lines
}
