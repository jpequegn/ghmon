// cmd/fetch.go
package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/julienpequegnot/ghmon/internal/account"
	"github.com/julienpequegnot/ghmon/internal/activity"
	"github.com/julienpequegnot/ghmon/internal/config"
	"github.com/julienpequegnot/ghmon/internal/database"
	"github.com/julienpequegnot/ghmon/internal/github"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch activity from monitored accounts",
	Long:  `Downloads recent commits, new repos, and stars from all monitored accounts.`,
	RunE:  runFetch,
}

func init() {
	rootCmd.AddCommand(fetchCmd)
}

func runFetch(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.GitHub.Token == "" {
		return fmt.Errorf("GitHub token not set. Add your token to %s", config.ConfigPath())
	}

	db, err := database.New(config.DBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	accountRepo := account.NewRepository(db)
	accounts, err := accountRepo.List()
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}

	if len(accounts) == 0 {
		fmt.Println("No accounts to fetch. Run 'ghmon sync' or 'ghmon add <username>' first.")
		return nil
	}

	client := github.NewClient(cfg.GitHub.Token)
	commitRepo := activity.NewCommitRepository(db)
	repoRepo := activity.NewRepoRepository(db)
	starRepo := activity.NewStarRepository(db)

	fmt.Printf("Fetching activity for %d accounts...\n\n", len(accounts))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, cfg.Fetch.Concurrency)

	var mu sync.Mutex
	totalCommits := 0
	totalRepos := 0
	totalStars := 0

	for _, acc := range accounts {
		wg.Add(1)
		go func(acc account.Account) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check rate limit before fetching
			client.WaitForRateLimit()

			commits, repos, stars := fetchAccountActivity(client, &acc, commitRepo, repoRepo, starRepo)

			mu.Lock()
			totalCommits += commits
			totalRepos += repos
			totalStars += stars
			mu.Unlock()

			accountRepo.UpdateLastFetched(acc.ID)

			fmt.Printf("  %s: %d commits, %d repos, %d stars\n", acc.Username, commits, repos, stars)
		}(acc)
	}

	wg.Wait()

	fmt.Printf("\nFetch complete: %d commits, %d new repos, %d stars\n", totalCommits, totalRepos, totalStars)

	// Show rate limit status
	if client.RateLimitRemaining() > 0 {
		fmt.Printf("Rate limit: %d requests remaining (resets %s)\n",
			client.RateLimitRemaining(),
			client.RateLimitReset().Format("15:04"))
	}

	fmt.Println("Run 'ghmon digest' to see the summary.")

	return nil
}

func fetchAccountActivity(
	client *github.Client,
	acc *account.Account,
	commitRepo *activity.CommitRepository,
	repoRepo *activity.RepoRepository,
	starRepo *activity.StarRepository,
) (commits, repos, stars int) {

	// Fetch events (commits)
	events, err := client.GetUserEvents(acc.Username)
	if err == nil {
		for _, event := range events {
			if event.Type == "PushEvent" {
				payload, err := github.ParsePushPayload(event.Payload)
				if err != nil {
					continue
				}
				for _, commit := range payload.Commits {
					if commitRepo.Add(acc.ID, event.Repo.Name, commit.SHA, commit.Message, event.CreatedAt) == nil {
						commits++
					}
				}
			}
		}
	}

	// Fetch repos
	userRepos, err := client.GetUserRepos(acc.Username)
	if err == nil {
		cutoff := time.Now().AddDate(0, 0, -90)
		for _, repo := range userRepos {
			if repo.CreatedAt.After(cutoff) {
				if repoRepo.Add(acc.ID, repo.Name, repo.FullName, repo.Description, repo.Language, repo.Stars, repo.CreatedAt) == nil {
					repos++
				}
			}
		}
	}

	// Fetch starred repos
	starred, err := client.GetUserStarred(acc.Username)
	if err == nil {
		cutoff := time.Now().AddDate(0, 0, -90)
		for _, star := range starred {
			if star.StarredAt.After(cutoff) {
				if starRepo.Add(acc.ID, star.FullName, star.Description, star.Language, star.Stars, star.StarredAt) == nil {
					stars++
				}
			}
		}
	}

	return commits, repos, stars
}
