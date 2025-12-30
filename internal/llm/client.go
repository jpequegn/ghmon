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
