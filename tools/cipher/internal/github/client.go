package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/verana-labs/verana/tools/cipher/internal/config"
)

type Issue struct {
	Number   int
	Title    string
	Body     string
	Labels   []string
	Priority int
}

type PR struct {
	Number  int
	Title   string
	Body    string
	Branch  string
	Base    string
	HeadSHA string
	BaseSHA string
	URL     string
	Draft   bool
}

type CIRun struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HTMLURL    string `json:"html_url"`
}

type Review struct {
	State string `json:"state"`
	Body  string `json:"body"`
	User  struct{ Login string `json:"login"` } `json:"user"`
}

type ReviewComment struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Body string `json:"body"`
	User struct{ Login string `json:"login"` } `json:"user"`
}

type Client struct {
	cfg  *config.Config
	http *http.Client
	repo string
}

func New(cfg *config.Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{},
		repo: fmt.Sprintf("%s/%s", cfg.RepoOwner, cfg.RepoName)}
}

func (c *Client) do(method, path string, body any) ([]byte, error) {
	var rb io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rb = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, "https://api.github.com"+path, rb)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.GitHubToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		return nil, fmt.Errorf("github %s %s → %d: %s", method, path, resp.StatusCode, string(data))
	}
	return data, nil
}

func (c *Client) get(path string) ([]byte, error)         { return c.do("GET", path, nil) }
func (c *Client) post(path string, b any) ([]byte, error) { return c.do("POST", path, b) }
func (c *Client) del(path string) error                   { _, err := c.do("DELETE", path, nil); return err }

func (c *Client) paginate(path string) ([]json.RawMessage, error) {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	var all []json.RawMessage
	for page := 1; ; page++ {
		data, err := c.get(fmt.Sprintf("%s%sper_page=100&page=%d", path, sep, page))
		if err != nil {
			return nil, err
		}
		var chunk []json.RawMessage
		if err := json.Unmarshal(data, &chunk); err != nil {
			return nil, err
		}
		all = append(all, chunk...)
		if len(chunk) < 100 {
			break
		}
	}
	return all, nil
}

// Issues

func (c *Client) GetCipherIssues() ([]Issue, error) {
	rows, err := c.paginate(fmt.Sprintf("/repos/%s/issues?state=open&labels=%s", c.repo, c.cfg.BotLabel))
	if err != nil {
		return nil, err
	}
	var issues []Issue
	for _, row := range rows {
		var raw struct {
			Number      int    `json:"number"`
			Title       string `json:"title"`
			Body        string `json:"body"`
			PullRequest *struct{} `json:"pull_request"`
			Labels      []struct{ Name string `json:"name"` } `json:"labels"`
		}
		if err := json.Unmarshal(row, &raw); err != nil || raw.PullRequest != nil {
			continue
		}
		priority, skip := 99, false
		var lbls []string
		for _, l := range raw.Labels {
			lbls = append(lbls, l.Name)
			if l.Name == "cipher-wip" || l.Name == "cipher-done" {
				skip = true
			}
			if strings.HasPrefix(l.Name, "priority::") {
				fmt.Sscanf(l.Name, "priority::%d", &priority)
			}
		}
		if skip {
			continue
		}
		issues = append(issues, Issue{Number: raw.Number, Title: raw.Title,
			Body: raw.Body, Labels: lbls, Priority: priority})
	}
	for i := 0; i < len(issues)-1; i++ {
		for j := i + 1; j < len(issues); j++ {
			if issues[j].Priority < issues[i].Priority {
				issues[i], issues[j] = issues[j], issues[i]
			}
		}
	}
	return issues, nil
}

func (c *Client) GetIssue(n int) (*Issue, error) {
	data, err := c.get(fmt.Sprintf("/repos/%s/issues/%d", c.repo, n))
	if err != nil {
		return nil, err
	}
	var raw struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		Labels []struct{ Name string `json:"name"` } `json:"labels"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	priority := 99
	var lbls []string
	for _, l := range raw.Labels {
		lbls = append(lbls, l.Name)
		if strings.HasPrefix(l.Name, "priority::") {
			fmt.Sscanf(l.Name, "priority::%d", &priority)
		}
	}
	return &Issue{Number: raw.Number, Title: raw.Title, Body: raw.Body,
		Labels: lbls, Priority: priority}, nil
}

func (c *Client) CreateIssue(title, body string, labels []string) (*Issue, error) {
	data, err := c.post(fmt.Sprintf("/repos/%s/issues", c.repo),
		map[string]any{"title": title, "body": body, "labels": labels})
	if err != nil {
		return nil, err
	}
	var raw struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	_ = json.Unmarshal(data, &raw)
	return &Issue{Number: raw.Number, Title: raw.Title, Body: body, Labels: labels}, nil
}

func (c *Client) AddLabel(n int, label string) error {
	_, err := c.post(fmt.Sprintf("/repos/%s/issues/%d/labels", c.repo, n),
		map[string]any{"labels": []string{label}})
	return err
}

func (c *Client) RemoveLabel(n int, label string) error {
	return c.del(fmt.Sprintf("/repos/%s/issues/%d/labels/%s", c.repo, n, label))
}

func (c *Client) PostIssueComment(n int, body string) error {
	_, err := c.post(fmt.Sprintf("/repos/%s/issues/%d/comments", c.repo, n),
		map[string]any{"body": body})
	return err
}

// Pull Requests

func (c *Client) GetBotPRs() ([]PR, error) {
	rows, err := c.paginate(fmt.Sprintf("/repos/%s/pulls?state=open", c.repo))
	if err != nil {
		return nil, err
	}
	var prs []PR
	for _, row := range rows {
		var raw struct {
			Number  int    `json:"number"`
			Title   string `json:"title"`
			Body    string `json:"body"`
			HTMLURL string `json:"html_url"`
			Draft   bool   `json:"draft"`
			User    struct{ Login string `json:"login"` } `json:"user"`
			Head    struct{ Ref, SHA string } `json:"head"`
			Base    struct{ Ref, SHA string } `json:"base"`
		}
		if err := json.Unmarshal(row, &raw); err != nil || raw.User.Login != c.cfg.BotUsername {
			continue
		}
		prs = append(prs, PR{Number: raw.Number, Title: raw.Title, Body: raw.Body,
			Branch: raw.Head.Ref, Base: raw.Base.Ref,
			HeadSHA: raw.Head.SHA, BaseSHA: raw.Base.SHA,
			URL: raw.HTMLURL, Draft: raw.Draft})
	}
	return prs, nil
}

func (c *Client) GetPR(n int) (*PR, error) {
	data, err := c.get(fmt.Sprintf("/repos/%s/pulls/%d", c.repo, n))
	if err != nil {
		return nil, err
	}
	var raw struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		Draft   bool   `json:"draft"`
		Head    struct{ Ref, SHA string } `json:"head"`
		Base    struct{ Ref, SHA string } `json:"base"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return &PR{Number: raw.Number, Title: raw.Title, Body: raw.Body,
		Branch: raw.Head.Ref, Base: raw.Base.Ref,
		HeadSHA: raw.Head.SHA, BaseSHA: raw.Base.SHA,
		URL: raw.HTMLURL, Draft: raw.Draft}, nil
}

func (c *Client) CreatePR(title, body, branch, base string) (*PR, error) {
	data, err := c.post(fmt.Sprintf("/repos/%s/pulls", c.repo),
		map[string]any{"title": title, "body": body, "head": branch, "base": base})
	if err != nil {
		return nil, err
	}
	var raw struct{ Number int `json:"number"` }
	_ = json.Unmarshal(data, &raw)
	return c.GetPR(raw.Number)
}

func (c *Client) PostPRComment(n int, body string) error {
	_, err := c.post(fmt.Sprintf("/repos/%s/issues/%d/comments", c.repo, n),
		map[string]any{"body": body})
	return err
}

func (c *Client) GetPRLastBotComment(n int) string {
	rows, _ := c.paginate(fmt.Sprintf("/repos/%s/issues/%d/comments", c.repo, n))
	var last string
	for _, row := range rows {
		var r struct {
			Body string `json:"body"`
			User struct{ Login string `json:"login"` } `json:"user"`
		}
		if err := json.Unmarshal(row, &r); err == nil && r.User.Login == c.cfg.BotUsername {
			last = r.Body
		}
	}
	return last
}

// Reviews

func (c *Client) GetChangeRequests(n int) ([]Review, error) {
	data, err := c.get(fmt.Sprintf("/repos/%s/pulls/%d/reviews", c.repo, n))
	if err != nil {
		return nil, err
	}
	var all []Review
	_ = json.Unmarshal(data, &all)
	var out []Review
	for _, r := range all {
		if r.State == "CHANGES_REQUESTED" && r.User.Login != c.cfg.BotUsername {
			out = append(out, r)
		}
	}
	return out, nil
}

func (c *Client) GetReviewComments(n int) ([]ReviewComment, error) {
	rows, err := c.paginate(fmt.Sprintf("/repos/%s/pulls/%d/comments", c.repo, n))
	if err != nil {
		return nil, err
	}
	var out []ReviewComment
	for _, row := range rows {
		var r ReviewComment
		if err := json.Unmarshal(row, &r); err == nil && r.User.Login != c.cfg.BotUsername {
			out = append(out, r)
		}
	}
	return out, nil
}

func (c *Client) FormatReviewFeedback(n int) (string, error) {
	reviews, err := c.GetChangeRequests(n)
	if err != nil {
		return "", err
	}
	comments, err := c.GetReviewComments(n)
	if err != nil {
		return "", err
	}
	var parts []string
	for _, r := range reviews {
		if r.Body != "" {
			parts = append(parts, fmt.Sprintf("**%s** (summary):\n%s", r.User.Login, r.Body))
		}
	}
	for _, c := range comments {
		loc := c.Path
		if c.Line > 0 {
			loc = fmt.Sprintf("%s:%d", c.Path, c.Line)
		}
		parts = append(parts, fmt.Sprintf("**%s** @ `%s`:\n%s", c.User.Login, loc, c.Body))
	}
	return strings.Join(parts, "\n\n---\n\n"), nil
}

// CI

func (c *Client) GetCIRuns(sha string) ([]CIRun, error) {
	data, err := c.get(fmt.Sprintf("/repos/%s/actions/runs?head_sha=%s", c.repo, sha))
	if err != nil {
		return nil, err
	}
	var resp struct{ Runs []CIRun `json:"workflow_runs"` }
	_ = json.Unmarshal(data, &resp)
	return resp.Runs, nil
}

func (c *Client) HasCIFailure(sha string) bool {
	runs, _ := c.GetCIRuns(sha)
	for _, r := range runs {
		if r.Conclusion == "failure" || r.Conclusion == "timed_out" {
			return true
		}
	}
	return false
}

func (c *Client) GetCIFailureSummary(sha string) (string, error) {
	runs, err := c.GetCIRuns(sha)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	found := false
	for _, run := range runs {
		if run.Conclusion != "failure" && run.Conclusion != "timed_out" {
			continue
		}
		found = true
		fmt.Fprintf(&b, "### Workflow: %s\nURL: %s\n", run.Name, run.HTMLURL)
		jobData, _ := c.get(fmt.Sprintf("/repos/%s/actions/runs/%d/jobs", c.repo, run.ID))
		if jobData != nil {
			var resp struct {
				Jobs []struct {
					Name       string `json:"name"`
					Conclusion string `json:"conclusion"`
					Steps      []struct {
						Name       string `json:"name"`
						Conclusion string `json:"conclusion"`
					} `json:"steps"`
				} `json:"jobs"`
			}
			_ = json.Unmarshal(jobData, &resp)
			for _, job := range resp.Jobs {
				if job.Conclusion == "failure" {
					fmt.Fprintf(&b, "\n**Failed Job:** %s\n", job.Name)
					for _, step := range job.Steps {
						if step.Conclusion == "failure" {
							fmt.Fprintf(&b, "  - `%s`\n", step.Name)
						}
					}
				}
			}
		}
	}
	if !found {
		return "", nil
	}
	return b.String(), nil
}

func (c *Client) GetRecentCommits(branch string, n int) (string, error) {
	data, err := c.get(fmt.Sprintf("/repos/%s/commits?sha=%s&per_page=%d", c.repo, branch, n))
	if err != nil {
		return "", err
	}
	var commits []struct {
		SHA    string `json:"sha"`
		Commit struct{ Message string `json:"message"` } `json:"commit"`
	}
	_ = json.Unmarshal(data, &commits)
	var lines []string
	for _, cm := range commits {
		msg := cm.Commit.Message
		if i := strings.Index(msg, "\n"); i >= 0 {
			msg = msg[:i]
		}
		if len(cm.SHA) >= 7 {
			lines = append(lines, fmt.Sprintf("%s %s", cm.SHA[:7], msg))
		}
	}
	return strings.Join(lines, "\n"), nil
}

func (c *Client) GetPRDiff(n int) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d", c.repo, n), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.GitHubToken)
	req.Header.Set("Accept", "application/vnd.github.v3.diff")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	diff := string(data)
	if len(diff) > 30000 {
		diff = diff[:30000] + "\n\n... (diff truncated at 30k chars)"
	}
	return diff, nil
}

func (c *Client) IsBranchBehind(branch, base string) bool {
	data, _ := c.get(fmt.Sprintf("/repos/%s/compare/%s...%s", c.repo, base, branch))
	if data == nil {
		return false
	}
	var resp struct{ BehindBy int `json:"behind_by"` }
	_ = json.Unmarshal(data, &resp)
	return resp.BehindBy > 0
}
