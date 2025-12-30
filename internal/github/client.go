// internal/github/client.go
package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const baseURL = "https://api.github.com"

type Client struct {
	token              string
	httpClient         *http.Client
	rateLimitRemaining int
	rateLimitReset     time.Time
}

type User struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
	Followers int    `json:"followers"`
	Following int    `json:"following"`
}

type Event struct {
	Type      string          `json:"type"`
	Repo      EventRepo       `json:"repo"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type EventRepo struct {
	Name string `json:"name"`
}

type PushPayload struct {
	Commits []Commit `json:"commits"`
}

type Commit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
}

type CreatePayload struct {
	RefType string `json:"ref_type"`
}

type StarredRepo struct {
	FullName    string    `json:"full_name"`
	Description string    `json:"description"`
	Language    string    `json:"language"`
	Stars       int       `json:"stargazers_count"`
	StarredAt   time.Time `json:"starred_at"`
}

type Repo struct {
	Name        string    `json:"name"`
	FullName    string    `json:"full_name"`
	Description string    `json:"description"`
	Language    string    `json:"language"`
	Stars       int       `json:"stargazers_count"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RateLimitRemaining returns the number of API calls remaining
func (c *Client) RateLimitRemaining() int {
	return c.rateLimitRemaining
}

// RateLimitReset returns when the rate limit resets
func (c *Client) RateLimitReset() time.Time {
	return c.rateLimitReset
}

// WaitForRateLimit blocks until rate limit resets if we're near the limit
func (c *Client) WaitForRateLimit() {
	if c.rateLimitRemaining > 0 && c.rateLimitRemaining < 10 && time.Now().Before(c.rateLimitReset) {
		waitTime := time.Until(c.rateLimitReset) + time.Second
		fmt.Printf("Rate limit low (%d remaining), waiting %v...\n", c.rateLimitRemaining, waitTime.Round(time.Second))
		time.Sleep(waitTime)
	}
}

func (c *Client) doRequest(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse rate limit headers
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil {
			c.rateLimitRemaining = val
		}
	}
	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		if val, err := strconv.ParseInt(reset, 10, 64); err == nil {
			c.rateLimitReset = time.Unix(val, 0)
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) GetFollowing() ([]User, error) {
	var allUsers []User
	page := 1

	for {
		url := fmt.Sprintf("%s/user/following?per_page=100&page=%d", baseURL, page)
		data, err := c.doRequest(url)
		if err != nil {
			return nil, err
		}

		var users []User
		if err := json.Unmarshal(data, &users); err != nil {
			return nil, err
		}

		if len(users) == 0 {
			break
		}

		allUsers = append(allUsers, users...)
		page++
	}

	return allUsers, nil
}

func (c *Client) GetUser(username string) (*User, error) {
	url := fmt.Sprintf("%s/users/%s", baseURL, username)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *Client) GetUserEvents(username string) ([]Event, error) {
	url := fmt.Sprintf("%s/users/%s/events/public?per_page=100", baseURL, username)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	return parseEvents(data)
}

func parseEvents(data []byte) ([]Event, error) {
	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *Client) GetUserStarred(username string) ([]StarredRepo, error) {
	url := fmt.Sprintf("%s/users/%s/starred?per_page=100", baseURL, username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.star+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse rate limit headers
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil {
			c.rateLimitRemaining = val
		}
	}
	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		if val, err := strconv.ParseInt(reset, 10, 64); err == nil {
			c.rateLimitReset = time.Unix(val, 0)
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var starResponse []struct {
		StarredAt time.Time `json:"starred_at"`
		Repo      Repo      `json:"repo"`
	}

	if err := json.Unmarshal(data, &starResponse); err != nil {
		return nil, err
	}

	var repos []StarredRepo
	for _, s := range starResponse {
		repos = append(repos, StarredRepo{
			FullName:    s.Repo.FullName,
			Description: s.Repo.Description,
			Language:    s.Repo.Language,
			Stars:       s.Repo.Stars,
			StarredAt:   s.StarredAt,
		})
	}

	return repos, nil
}

func (c *Client) GetUserRepos(username string) ([]Repo, error) {
	url := fmt.Sprintf("%s/users/%s/repos?per_page=100&sort=created&direction=desc", baseURL, username)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	var repos []Repo
	if err := json.Unmarshal(data, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

func ParsePushPayload(payload json.RawMessage) (*PushPayload, error) {
	var p PushPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func ParseCreatePayload(payload json.RawMessage) (*CreatePayload, error) {
	var p CreatePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
