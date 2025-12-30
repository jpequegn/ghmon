// cmd/show.go
package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/julienpequegnot/ghmon/internal/account"
	"github.com/julienpequegnot/ghmon/internal/activity"
	"github.com/julienpequegnot/ghmon/internal/config"
	"github.com/julienpequegnot/ghmon/internal/database"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <username>",
	Short: "Show activity for a specific user",
	Long:  `Displays detailed activity for a monitored GitHub user.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

var showDays int

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().IntVar(&showDays, "days", 7, "Number of days to show")
}

func runShow(cmd *cobra.Command, args []string) error {
	username := args[0]

	db, err := database.New(config.DBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	accountRepo := account.NewRepository(db)
	acc, err := accountRepo.Get(username)
	if err != nil {
		return fmt.Errorf("account '%s' not found. Run 'ghmon add %s' first.", username, username)
	}

	since := time.Now().AddDate(0, 0, -showDays)

	commitRepo := activity.NewCommitRepository(db)
	repoRepo := activity.NewRepoRepository(db)
	starRepo := activity.NewStarRepository(db)

	commits, _ := commitRepo.GetForAccount(acc.ID, since)
	newRepos, _ := repoRepo.GetNewSince(since)
	stars, _ := starRepo.GetSince(since)

	var accountRepos []activity.Repo
	for _, r := range newRepos {
		if r.AccountID == acc.ID {
			accountRepos = append(accountRepos, r)
		}
	}
	var accountStars []activity.Star
	for _, s := range stars {
		if s.AccountID == acc.ID {
			accountStars = append(accountStars, s)
		}
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	repoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	shaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	name := acc.Name
	if name == "" {
		name = acc.Username
	}
	fmt.Printf("\n%s\n", titleStyle.Render(name))
	if acc.Bio != "" {
		fmt.Printf("%s\n", dimStyle.Render(acc.Bio))
	}
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

	fmt.Printf("\n📊 Last %d days: %d commits · %d new repos · %d stars\n\n",
		showDays, len(commits), len(accountRepos), len(accountStars))

	if len(commits) > 0 {
		fmt.Printf("%s\n", sectionStyle.Render("📝 Recent Commits"))

		repoCommits := make(map[string][]activity.Commit)
		for _, c := range commits {
			repoCommits[c.RepoName] = append(repoCommits[c.RepoName], c)
		}

		for repo, repoCs := range repoCommits {
			fmt.Printf("  %s (%d commits)\n", repoStyle.Render(repo), len(repoCs))
			limit := 3
			if len(repoCs) < limit {
				limit = len(repoCs)
			}
			for i := 0; i < limit; i++ {
				c := repoCs[i]
				msg := c.Message
				if len(msg) > 50 {
					msg = msg[:47] + "..."
				}
				fmt.Printf("    %s %s\n", shaStyle.Render(c.SHA[:7]), msg)
			}
			if len(repoCs) > 3 {
				fmt.Printf("    %s\n", dimStyle.Render(fmt.Sprintf("... and %d more", len(repoCs)-3)))
			}
		}
		fmt.Println()
	}

	if len(accountRepos) > 0 {
		fmt.Printf("%s\n", sectionStyle.Render("🆕 New Repositories"))
		for _, repo := range accountRepos {
			fmt.Printf("  %s\n", repoStyle.Render(repo.FullName))
			if repo.Description != "" {
				desc := repo.Description
				if len(desc) > 60 {
					desc = desc[:57] + "..."
				}
				fmt.Printf("    %s\n", dimStyle.Render(desc))
			}
			if repo.Language != "" {
				fmt.Printf("    %s\n", dimStyle.Render(repo.Language))
			}
		}
		fmt.Println()
	}

	if len(accountStars) > 0 {
		fmt.Printf("%s\n", sectionStyle.Render("⭐ Starred Repos"))
		limit := 10
		if len(accountStars) < limit {
			limit = len(accountStars)
		}
		for i := 0; i < limit; i++ {
			s := accountStars[i]
			fmt.Printf("  %s\n", repoStyle.Render(s.RepoFullName))
			if s.RepoLanguage != "" {
				fmt.Printf("    %s · %d ★\n", dimStyle.Render(s.RepoLanguage), s.RepoStars)
			}
		}
		if len(accountStars) > limit {
			fmt.Printf("  %s\n", dimStyle.Render(fmt.Sprintf("... and %d more", len(accountStars)-limit)))
		}
		fmt.Println()
	}

	return nil
}
