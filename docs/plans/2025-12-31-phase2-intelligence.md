# Phase 2 Intelligence Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add LLM-powered analysis to ghmon digest with `--smart` flag for AI-generated focus area insights.

**Architecture:** Ollama client for local LLM inference. Enhanced digest aggregates trending stars (repos starred by multiple follows), language breakdown, and generates focus area narratives via LLM when `--smart` flag is used.

**Tech Stack:** Go, Ollama API (HTTP), existing SQLite database

---

## Task 1: LLM Client Package

**Files:**
- Create: `internal/llm/client.go`
- Create: `internal/llm/client_test.go`

**Step 1: Write tests**

```go
// internal/llm/client_test.go
package llm

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:11434", "llama3.2")
	if client == nil {
		t.Error("expected non-nil client")
	}
}

func TestGeneratePrompt(t *testing.T) {
	prompt := GenerateDigestPrompt(DigestData{
		TotalCommits:   100,
		TotalRepos:     5,
		TotalStars:     50,
		TopLanguages:   []string{"Go", "Rust", "Python"},
		TrendingRepos:  []string{"ollama/ollama", "astral-sh/ruff"},
		MostActiveUser: "torvalds",
	})

	if prompt == "" {
		t.Error("expected non-empty prompt")
	}

	if len(prompt) < 100 {
		t.Error("prompt seems too short")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/julienpequegnot/Code/ghmon && go test ./internal/llm/... -v
```

Expected: FAIL - package doesn't exist

**Step 3: Implement LLM client**

```go
// internal/llm/client.go
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

type DigestData struct {
	TotalCommits   int
	TotalRepos     int
	TotalStars     int
	TopLanguages   []string
	TrendingRepos  []string
	MostActiveUser string
	ActiveUsers    []UserActivity
}

type UserActivity struct {
	Username string
	Commits  int
	Repos    []string
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
}

func NewClient(baseURL, model string) *Client {
	return &Client{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody := generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama error %d: %s", resp.StatusCode, string(body))
	}

	var result generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return strings.TrimSpace(result.Response), nil
}

func GenerateDigestPrompt(data DigestData) string {
	var sb strings.Builder

	sb.WriteString("Analyze this GitHub activity digest and provide 2-3 brief insights about what developers are focusing on.\n\n")
	sb.WriteString("Activity Summary:\n")
	sb.WriteString(fmt.Sprintf("- %d total commits\n", data.TotalCommits))
	sb.WriteString(fmt.Sprintf("- %d new repositories created\n", data.TotalRepos))
	sb.WriteString(fmt.Sprintf("- %d repositories starred\n", data.TotalStars))

	if len(data.TopLanguages) > 0 {
		sb.WriteString(fmt.Sprintf("- Top languages: %s\n", strings.Join(data.TopLanguages, ", ")))
	}

	if len(data.TrendingRepos) > 0 {
		sb.WriteString(fmt.Sprintf("- Trending repos (starred by multiple devs): %s\n", strings.Join(data.TrendingRepos, ", ")))
	}

	if data.MostActiveUser != "" {
		sb.WriteString(fmt.Sprintf("- Most active: %s\n", data.MostActiveUser))
	}

	if len(data.ActiveUsers) > 0 {
		sb.WriteString("\nActive developers:\n")
		for _, u := range data.ActiveUsers {
			repos := strings.Join(u.Repos, ", ")
			if len(repos) > 80 {
				repos = repos[:77] + "..."
			}
			sb.WriteString(fmt.Sprintf("- %s: %d commits in %s\n", u.Username, u.Commits, repos))
		}
	}

	sb.WriteString("\nProvide 2-3 concise bullet points about emerging trends, technology focus areas, or notable patterns. Be specific and actionable. Keep each bullet under 100 characters.")

	return sb.String()
}
```

**Step 4: Run tests**

Run:
```bash
cd /Users/julienpequegnot/Code/ghmon && go test ./internal/llm/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /Users/julienpequegnot/Code/ghmon && git add . && git commit -m "feat: add LLM client for Ollama integration"
```

---

## Task 2: Trending Stars Detection

**Files:**
- Modify: `internal/activity/stars.go`

**Step 1: Add trending detection method**

Add to `internal/activity/stars.go`:

```go
// TrendingRepo represents a repo starred by multiple followed accounts
type TrendingRepo struct {
	RepoFullName    string
	RepoDescription string
	RepoLanguage    string
	StarCount       int
	StarredBy       []string
}

// GetTrendingRepos returns repos starred by multiple followed accounts
func (r *StarRepository) GetTrendingRepos(since time.Time, minStars int) ([]TrendingRepo, error) {
	rows, err := r.db.Query(`
		SELECT
			s.repo_full_name,
			s.repo_description,
			s.repo_language,
			COUNT(DISTINCT s.account_id) as star_count,
			GROUP_CONCAT(a.username, ',') as usernames
		FROM stars s
		JOIN accounts a ON s.account_id = a.id
		WHERE s.starred_at >= ?
		GROUP BY s.repo_full_name
		HAVING star_count >= ?
		ORDER BY star_count DESC
		LIMIT 10
	`, since, minStars)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trending []TrendingRepo
	for rows.Next() {
		var t TrendingRepo
		var usernames string
		if err := rows.Scan(&t.RepoFullName, &t.RepoDescription, &t.RepoLanguage, &t.StarCount, &usernames); err != nil {
			return nil, err
		}
		t.StarredBy = strings.Split(usernames, ",")
		trending = append(trending, t)
	}

	return trending, rows.Err()
}
```

**Step 2: Add import for strings**

Add `"strings"` to imports in `internal/activity/stars.go` if not present.

**Step 3: Build and verify**

Run:
```bash
cd /Users/julienpequegnot/Code/ghmon && go build -o ghmon .
```

Expected: Build succeeds

**Step 4: Commit**

```bash
cd /Users/julienpequegnot/Code/ghmon && git add . && git commit -m "feat: add trending repos detection"
```

---

## Task 3: Language Analysis

**Files:**
- Create: `internal/analysis/languages.go`

**Step 1: Create language analyzer**

```go
// internal/analysis/languages.go
package analysis

import (
	"sort"

	"github.com/julienpequegnot/ghmon/internal/activity"
)

type LanguageStats struct {
	Language   string
	Count      int
	Percentage float64
}

// AnalyzeLanguages aggregates language usage from repos and stars
func AnalyzeLanguages(repos []activity.Repo, stars []activity.Star) []LanguageStats {
	counts := make(map[string]int)
	total := 0

	for _, repo := range repos {
		if repo.Language != "" {
			counts[repo.Language]++
			total++
		}
	}

	for _, star := range stars {
		if star.RepoLanguage != "" {
			counts[star.RepoLanguage]++
			total++
		}
	}

	if total == 0 {
		return nil
	}

	var stats []LanguageStats
	for lang, count := range counts {
		stats = append(stats, LanguageStats{
			Language:   lang,
			Count:      count,
			Percentage: float64(count) / float64(total) * 100,
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	// Return top 5
	if len(stats) > 5 {
		stats = stats[:5]
	}

	return stats
}

// GetTopLanguageNames returns just the language names
func GetTopLanguageNames(stats []LanguageStats) []string {
	names := make([]string, len(stats))
	for i, s := range stats {
		names[i] = s.Language
	}
	return names
}
```

**Step 2: Build and verify**

Run:
```bash
cd /Users/julienpequegnot/Code/ghmon && go build -o ghmon .
```

Expected: Build succeeds

**Step 3: Commit**

```bash
cd /Users/julienpequegnot/Code/ghmon && git add . && git commit -m "feat: add language analysis"
```

---

## Task 4: User Activity Aggregation

**Files:**
- Modify: `internal/activity/commits.go`

**Step 1: Add user activity aggregation**

Add to `internal/activity/commits.go`:

```go
// UserCommitActivity holds commit stats per user
type UserCommitActivity struct {
	AccountID int64
	Username  string
	Count     int
	Repos     []string
}

// GetUserActivity returns commit activity grouped by user with repo details
func (r *CommitRepository) GetUserActivity(since time.Time, limit int) ([]UserCommitActivity, error) {
	rows, err := r.db.Query(`
		SELECT
			c.account_id,
			a.username,
			COUNT(*) as commit_count,
			GROUP_CONCAT(DISTINCT c.repo_name) as repos
		FROM commits c
		JOIN accounts a ON c.account_id = a.id
		WHERE c.committed_at >= ?
		GROUP BY c.account_id
		ORDER BY commit_count DESC
		LIMIT ?
	`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []UserCommitActivity
	for rows.Next() {
		var ua UserCommitActivity
		var repos string
		if err := rows.Scan(&ua.AccountID, &ua.Username, &ua.Count, &repos); err != nil {
			return nil, err
		}
		if repos != "" {
			ua.Repos = strings.Split(repos, ",")
		}
		activities = append(activities, ua)
	}

	return activities, rows.Err()
}
```

**Step 2: Add import for strings**

Add `"strings"` to imports in `internal/activity/commits.go` if not present.

**Step 3: Build and verify**

Run:
```bash
cd /Users/julienpequegnot/Code/ghmon && go build -o ghmon .
```

Expected: Build succeeds

**Step 4: Commit**

```bash
cd /Users/julienpequegnot/Code/ghmon && git add . && git commit -m "feat: add user activity aggregation"
```

---

## Task 5: Enhanced Digest with --smart Flag

**Files:**
- Modify: `cmd/digest.go`

**Step 1: Read current digest.go**

Read `/Users/julienpequegnot/Code/ghmon/cmd/digest.go` to understand current structure.

**Step 2: Add --smart flag and LLM integration**

Update `cmd/digest.go` with these changes:

1. Add imports:
```go
"context"
"github.com/julienpequegnot/ghmon/internal/analysis"
"github.com/julienpequegnot/ghmon/internal/llm"
```

2. Add flag variable:
```go
var (
	digestDays  int
	digestSmart bool
)
```

3. Update init():
```go
func init() {
	rootCmd.AddCommand(digestCmd)
	digestCmd.Flags().IntVar(&digestDays, "days", 7, "Number of days to include in digest")
	digestCmd.Flags().BoolVar(&digestSmart, "smart", false, "Use LLM for intelligent analysis")
}
```

4. Add trending repos section after the stars section in runDigest():
```go
	// Trending repos (starred by multiple accounts)
	trendingRepos, _ := starRepo.GetTrendingRepos(since, 2)
	if len(trendingRepos) > 0 {
		fmt.Printf("%s\n", sectionStyle.Render("🔥 Trending (starred by multiple follows)"))
		for _, t := range trendingRepos {
			fmt.Printf("  %s\n", repoStyle.Render(t.RepoFullName))
			fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("★ by %s", strings.Join(t.StarredBy, ", "))))
		}
		fmt.Println()
	}
```

5. Add smart analysis section at the end of runDigest() (before final return):
```go
	// Smart analysis with LLM
	if digestSmart {
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("%s\n", dimStyle.Render("Note: Could not load config for LLM analysis"))
		} else {
			fmt.Printf("%s\n", sectionStyle.Render("💡 Focus Areas (AI-generated)"))

			// Prepare data for LLM
			langStats := analysis.AnalyzeLanguages(newRepos, recentStars)
			var trendingNames []string
			for _, t := range trendingRepos {
				trendingNames = append(trendingNames, t.RepoFullName)
			}

			userActivities, _ := commitRepo.GetUserActivity(since, 5)
			var llmUsers []llm.UserActivity
			for _, ua := range userActivities {
				llmUsers = append(llmUsers, llm.UserActivity{
					Username: ua.Username,
					Commits:  ua.Count,
					Repos:    ua.Repos,
				})
			}

			mostActive := ""
			if len(userActivities) > 0 {
				mostActive = userActivities[0].Username
			}

			digestData := llm.DigestData{
				TotalCommits:   totalCommits,
				TotalRepos:     len(newRepos),
				TotalStars:     len(recentStars),
				TopLanguages:   analysis.GetTopLanguageNames(langStats),
				TrendingRepos:  trendingNames,
				MostActiveUser: mostActive,
				ActiveUsers:    llmUsers,
			}

			prompt := llm.GenerateDigestPrompt(digestData)
			client := llm.NewClient("http://localhost:11434", cfg.APIs.LLMModel)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			response, err := client.Generate(ctx, prompt)
			cancel()

			if err != nil {
				fmt.Printf("  %s\n\n", dimStyle.Render(fmt.Sprintf("LLM analysis unavailable: %v", err)))
			} else {
				// Print each line of the response
				for _, line := range strings.Split(response, "\n") {
					line = strings.TrimSpace(line)
					if line != "" {
						fmt.Printf("  %s\n", line)
					}
				}
				fmt.Println()
			}
		}
	}
```

6. Add missing imports at top: `"context"`, `"strings"` if not present

**Step 3: Build and test**

Run:
```bash
cd /Users/julienpequegnot/Code/ghmon && go build -o ghmon . && ./ghmon digest --help
```

Expected: Help shows --smart flag

**Step 4: Commit**

```bash
cd /Users/julienpequegnot/Code/ghmon && git add . && git commit -m "feat: add --smart flag for LLM-powered digest analysis"
```

---

## Task 6: Update README and Final Tests

**Files:**
- Modify: `README.md`

**Step 1: Run all tests**

Run:
```bash
cd /Users/julienpequegnot/Code/ghmon && go test ./... -v
```

Expected: All tests pass

**Step 2: Update README**

Update Development Status section:
```markdown
### Phase 2 (Intelligence) - Complete
- [x] LLM integration (Ollama)
- [x] Language analysis
- [x] Trending stars detection
- [x] Focus area extraction (--smart flag)
```

Update Commands table to show --smart flag:
```markdown
| `ghmon digest` | Show activity summary (--smart for AI insights) |
```

**Step 3: Commit and push**

```bash
cd /Users/julienpequegnot/Code/ghmon && git add . && git commit -m "docs: mark Phase 2 Intelligence as complete"
cd /Users/julienpequegnot/Code/ghmon && git push
```

---

## Summary

**Phase 2 delivers:**
- LLM client for Ollama integration
- Trending repos detection (starred by multiple follows)
- Language usage analysis
- User activity aggregation
- Enhanced digest with `--smart` flag for AI insights

**New capabilities:**
- `ghmon digest --smart` - AI-generated focus area analysis
- Trending repos section in digest
- Richer language breakdown

**Enhanced digest output with --smart:**
```
💡 Focus Areas (AI-generated)
  • Systems programming surge: multiple devs active in Rust/Go tooling
  • AI infrastructure: llama.cpp and ollama gaining traction
  • Developer experience: CLI tools and automation trending
```
