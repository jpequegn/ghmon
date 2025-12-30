// cmd/accounts.go
package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/julienpequegnot/ghmon/internal/account"
	"github.com/julienpequegnot/ghmon/internal/config"
	"github.com/julienpequegnot/ghmon/internal/database"
	"github.com/spf13/cobra"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "List monitored accounts",
	Long:  `Shows all GitHub accounts being monitored.`,
	RunE:  runAccounts,
}

func init() {
	rootCmd.AddCommand(accountsCmd)
}

func runAccounts(cmd *cobra.Command, args []string) error {
	db, err := database.New(config.DBPath())
	if err != nil {
		return fmt.Errorf("failed to open database (run 'ghmon init' first): %w", err)
	}
	defer db.Close()

	accountRepo := account.NewRepository(db)
	accounts, err := accountRepo.List()
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}

	if len(accounts) == 0 {
		fmt.Println("No accounts monitored yet.")
		fmt.Println("Run 'ghmon sync' to import from your following list, or 'ghmon add <username>' to add manually.")
		return nil
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	fmt.Printf("\n%s (%d)\n\n", titleStyle.Render("MONITORED ACCOUNTS"), len(accounts))

	for _, acc := range accounts {
		name := acc.Name
		if name == "" {
			name = acc.Username
		}
		fmt.Printf("  %s", userStyle.Render(acc.Username))
		if acc.Name != "" && acc.Name != acc.Username {
			fmt.Printf(" %s", dimStyle.Render("("+acc.Name+")"))
		}
		fmt.Println()

		if acc.Bio != "" {
			bio := acc.Bio
			if len(bio) > 60 {
				bio = bio[:57] + "..."
			}
			fmt.Printf("    %s\n", dimStyle.Render(bio))
		}
	}

	fmt.Println()
	return nil
}
