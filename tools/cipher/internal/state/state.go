package state

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type TaskRecord struct {
	Type   string `json:"type"`
	Branch string `json:"branch,omitempty"`
	Issue  int    `json:"issue,omitempty"`
	PR     int    `json:"pr,omitempty"`
	OK     bool   `json:"ok"`
	TS     string `json:"ts"`
}

type stateData struct {
	CIAttempts     map[string]int    `json:"ci_attempts"`
	KnownPRCI      map[string]string `json:"known_pr_ci"`
	KnownPRReviews map[string]bool   `json:"known_pr_reviews"`
	TaskHistory    []TaskRecord      `json:"task_history"`
}

type State struct {
	mu   sync.Mutex
	path string
	d    stateData
}

func New(path string) *State {
	s := &State{
		path: path,
		d: stateData{
			CIAttempts:     make(map[string]int),
			KnownPRCI:      make(map[string]string),
			KnownPRReviews: make(map[string]bool),
		},
	}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &s.d)
		if s.d.CIAttempts == nil {
			s.d.CIAttempts = make(map[string]int)
		}
		if s.d.KnownPRCI == nil {
			s.d.KnownPRCI = make(map[string]string)
		}
		if s.d.KnownPRReviews == nil {
			s.d.KnownPRReviews = make(map[string]bool)
		}
	}
	return s
}

func (s *State) save() {
	b, _ := json.MarshalIndent(s.d, "", "  ")
	_ = os.WriteFile(s.path, b, 0o644)
}

func (s *State) GetCIAttempts(branch string) int {
	s.mu.Lock(); defer s.mu.Unlock()
	return s.d.CIAttempts[branch]
}

func (s *State) IncrementCIAttempts(branch string) int {
	s.mu.Lock(); defer s.mu.Unlock()
	s.d.CIAttempts[branch]++
	n := s.d.CIAttempts[branch]
	s.save()
	return n
}

func (s *State) ResetCIAttempts(branch string) {
	s.mu.Lock(); defer s.mu.Unlock()
	delete(s.d.CIAttempts, branch)
	s.save()
}

func (s *State) GetKnownPRCI(prNumber int) string {
	s.mu.Lock(); defer s.mu.Unlock()
	return s.d.KnownPRCI[fmt.Sprintf("%d", prNumber)]
}

func (s *State) SetKnownPRCI(prNumber int, conclusion string) {
	s.mu.Lock(); defer s.mu.Unlock()
	s.d.KnownPRCI[fmt.Sprintf("%d", prNumber)] = conclusion
	s.save()
}

func (s *State) GetKnownPRReviews(prNumber int) bool {
	s.mu.Lock(); defer s.mu.Unlock()
	return s.d.KnownPRReviews[fmt.Sprintf("%d", prNumber)]
}

func (s *State) SetKnownPRReviews(prNumber int, hasCR bool) {
	s.mu.Lock(); defer s.mu.Unlock()
	s.d.KnownPRReviews[fmt.Sprintf("%d", prNumber)] = hasCR
	s.save()
}

func (s *State) RecordTask(r TaskRecord) {
	s.mu.Lock(); defer s.mu.Unlock()
	r.TS = time.Now().UTC().Format(time.RFC3339)
	s.d.TaskHistory = append(s.d.TaskHistory, r)
	if len(s.d.TaskHistory) > 50 {
		s.d.TaskHistory = s.d.TaskHistory[len(s.d.TaskHistory)-50:]
	}
	s.save()
}

func (s *State) RecentTasks(n int) []TaskRecord {
	s.mu.Lock(); defer s.mu.Unlock()
	h := s.d.TaskHistory
	if len(h) <= n {
		cp := make([]TaskRecord, len(h))
		copy(cp, h)
		return cp
	}
	cp := make([]TaskRecord, n)
	copy(cp, h[len(h)-n:])
	return cp
}
