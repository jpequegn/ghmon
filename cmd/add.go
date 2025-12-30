// cmd/add.go
package cmd

import (
	"fmt"

	"github.com/julienpequegnot/ghmon/internal/account"
	"github.com/julienpequegnot/ghmon/internal/config"
	"github.com/julienpequegnot/ghmon/internal/database"
	"github.com/julienpequegnot/ghmon/internal/github"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <username>",
	Short: "Add a GitHub user to monitor",
	Long:  `Adds a GitHub user to your monitored accounts.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	username := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config (run 'ghmon init' first): %w", err)
	}

	db, err := database.New(config.DBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	accountRepo := account.NewRepository(db)

	if accountRepo.Exists(username) {
		return fmt.Errorf("account '%s' is already being monitored", username)
	}

	var name, avatarURL, bio string
	if cfg.GitHub.Token != "" {
		client := github.NewClient(cfg.GitHub.Token)
		user, err := client.GetUser(username)
		if err != nil {
			fmt.Printf("Warning: couldn't fetch user info: %v\n", err)
		} else {
			name = user.Name
			avatarURL = user.AvatarURL
			bio = user.Bio
		}
	}

	_, err = accountRepo.Add(username, name, avatarURL, bio)
	if err != nil {
		return fmt.Errorf("failed to add account: %w", err)
	}

	fmt.Printf("Added %s to monitored accounts.\n", username)
	fmt.Println("Run 'ghmon fetch' to pull their activity.")

	return nil
}
