// cmd/digest.go
package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/julienpequegnot/ghmon/internal/account"
	"github.com/julienpequegnot/ghmon/internal/activity"
	"github.com/julienpequegnot/ghmon/internal/analysis"
	"github.com/julienpequegnot/ghmon/internal/config"
	"github.com/julienpequegnot/ghmon/internal/database"
	"github.com/julienpequegnot/ghmon/internal/llm"
	"github.com/spf13/cobra"
)

var digestCmd = &cobra.Command{
	Use:   "digest",
	Short: "Show activity digest",
	Long:  `Displays a summary of recent activity from monitored accounts.`,
	RunE:  runDigest,
}

var (
	digestDays  int
	digestSmart bool
)

func init() {
	rootCmd.AddCommand(digestCmd)
	digestCmd.Flags().IntVar(&digestDays, "days", 7, "Number of days to include in digest")
	digestCmd.Flags().BoolVar(&digestSmart, "smart", false, "Use LLM for intelligent analysis")
}

func runDigest(cmd *cobra.Command, args []string) error {
	db, err := database.New(config.DBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	since := time.Now().AddDate(0, 0, -digestDays)
	endDate := time.Now()

	accountRepo := account.NewRepository(db)
	commitRepo := activity.NewCommitRepository(db)
	repoRepo := activity.NewRepoRepository(db)
	starRepo := activity.NewStarRepository(db)

	accounts, _ := accountRepo.List()
	accountMap := make(map[int64]*account.Account)
	for i := range accounts {
		accountMap[accounts[i].ID] = &accounts[i]
	}

	commitCounts, _ := commitRepo.CountByAccount(since)
	newRepos, _ := repoRepo.GetNewSince(since)
	recentStars, _ := starRepo.GetSince(since)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	repoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	fmt.Printf("\n%s (%s - %s)\n",
		titleStyle.Render("GITHUB DIGEST"),
		since.Format("Jan 2"),
		endDate.Format("Jan 2, 2006"))
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

	totalCommits := 0
	for _, c := range commitCounts {
		totalCommits += c
	}

	fmt.Printf("\n📊 Summary: %d accounts · %d commits · %d new repos · %d stars\n\n",
		len(accounts), totalCommits, len(newRepos), len(recentStars))

	if len(commitCounts) > 0 {
		fmt.Printf("%s\n", sectionStyle.Render("🔥 Most Active"))

		type accountCommits struct {
			username string
			count    int
		}
		var sorted []accountCommits
		for accID, count := range commitCounts {
			if acc, ok := accountMap[accID]; ok {
				sorted = append(sorted, accountCommits{acc.Username, count})
			}
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].count > sorted[j].count
		})

		limit := 5
		if len(sorted) < limit {
			limit = len(sorted)
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("  %-20s %d commits\n",
				userStyle.Render(sorted[i].username),
				sorted[i].count)
		}
		fmt.Println()
	}

	if len(newRepos) > 0 {
		fmt.Printf("%s\n", sectionStyle.Render("🆕 New Repositories"))

		limit := 5
		if len(newRepos) < limit {
			limit = len(newRepos)
		}
		for i := 0; i < limit; i++ {
			repo := newRepos[i]
			desc := repo.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			if desc == "" {
				desc = "(no description)"
			}
			fmt.Printf("  %s\n", repoStyle.Render(repo.FullName))
			fmt.Printf("    %s\n", dimStyle.Render(desc))
		}
		fmt.Println()
	}

	if len(recentStars) > 0 {
		fmt.Printf("%s\n", sectionStyle.Render("⭐ Recent Stars"))

		starCounts := make(map[string][]string)
		for _, s := range recentStars {
			if acc, ok := accountMap[s.AccountID]; ok {
				starCounts[s.RepoFullName] = append(starCounts[s.RepoFullName], acc.Username)
			}
		}

		type repoStars struct {
			repo  string
			users []string
		}
		var sorted []repoStars
		for repo, users := range starCounts {
			sorted = append(sorted, repoStars{repo, users})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return len(sorted[i].users) > len(sorted[j].users)
		})

		limit := 5
		if len(sorted) < limit {
			limit = len(sorted)
		}
		for i := 0; i < limit; i++ {
			item := sorted[i]
			fmt.Printf("  %s\n", repoStyle.Render(item.repo))
			if len(item.users) > 1 {
				fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("★ by %v", item.users)))
			} else {
				fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("★ by %s", item.users[0])))
			}
		}
		fmt.Println()
	}

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

	languages := make(map[string]int)
	for _, repo := range newRepos {
		if repo.Language != "" {
			languages[repo.Language]++
		}
	}
	for _, star := range recentStars {
		if star.RepoLanguage != "" {
			languages[star.RepoLanguage]++
		}
	}

	if len(languages) > 0 {
		fmt.Printf("%s\n", sectionStyle.Render("🏷️ Languages"))

		type langCount struct {
			lang  string
			count int
		}
		var sorted []langCount
		total := 0
		for lang, count := range languages {
			sorted = append(sorted, langCount{lang, count})
			total += count
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].count > sorted[j].count
		})

		var parts []string
		limit := 5
		if len(sorted) < limit {
			limit = len(sorted)
		}
		for i := 0; i < limit; i++ {
			pct := float64(sorted[i].count) / float64(total) * 100
			parts = append(parts, fmt.Sprintf("%s (%.0f%%)", sorted[i].lang, pct))
		}
		fmt.Printf("  %s\n\n", dimStyle.Render(joinStrings(parts, " · ")))
	}

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

	return nil
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
