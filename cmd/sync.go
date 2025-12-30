// cmd/sync.go
package cmd

import (
	"fmt"

	"github.com/julienpequegnot/ghmon/internal/account"
	"github.com/julienpequegnot/ghmon/internal/config"
	"github.com/julienpequegnot/ghmon/internal/database"
	"github.com/julienpequegnot/ghmon/internal/github"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Import accounts from GitHub following",
	Long:  `Syncs your monitored accounts with your GitHub following list.`,
	RunE:  runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config (run 'ghmon init' first): %w", err)
	}

	if cfg.GitHub.Token == "" {
		return fmt.Errorf("GitHub token not set. Add your token to %s", config.ConfigPath())
	}

	db, err := database.New(config.DBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Println("Fetching your GitHub following list...")

	client := github.NewClient(cfg.GitHub.Token)
	following, err := client.GetFollowing()
	if err != nil {
		return fmt.Errorf("failed to fetch following: %w", err)
	}

	fmt.Printf("Found %d accounts you follow.\n", len(following))

	accountRepo := account.NewRepository(db)
	added := 0
	skipped := 0

	for _, user := range following {
		if accountRepo.Exists(user.Login) {
			skipped++
			continue
		}

		_, err := accountRepo.Add(user.Login, user.Name, user.AvatarURL, user.Bio)
		if err != nil {
			fmt.Printf("  Warning: failed to add %s: %v\n", user.Login, err)
			continue
		}
		added++
		fmt.Printf("  + %s\n", user.Login)
	}

	fmt.Printf("\nSync complete: %d added, %d already tracked\n", added, skipped)
	fmt.Println("Run 'ghmon fetch' to pull their activity.")

	return nil
}
