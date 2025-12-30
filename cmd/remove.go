// cmd/remove.go
package cmd

import (
	"fmt"

	"github.com/julienpequegnot/ghmon/internal/account"
	"github.com/julienpequegnot/ghmon/internal/config"
	"github.com/julienpequegnot/ghmon/internal/database"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <username>",
	Short: "Remove a GitHub user from monitoring",
	Long:  `Removes a GitHub user from your monitored accounts.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	username := args[0]

	db, err := database.New(config.DBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	accountRepo := account.NewRepository(db)

	if !accountRepo.Exists(username) {
		return fmt.Errorf("account '%s' is not being monitored", username)
	}

	if err := accountRepo.Remove(username); err != nil {
		return fmt.Errorf("failed to remove account: %w", err)
	}

	fmt.Printf("Removed %s from monitored accounts.\n", username)
	return nil
}
