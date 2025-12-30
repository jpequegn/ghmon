// cmd/init.go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/julienpequegnot/ghmon/internal/config"
	"github.com/julienpequegnot/ghmon/internal/database"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize ghmon configuration",
	Long:  `Creates the config file and database for ghmon.`,
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if config.Exists() {
		fmt.Println("ghmon is already initialized.")
		fmt.Printf("Config: %s\n", config.ConfigPath())
		fmt.Printf("Database: %s\n", config.DBPath())
		return nil
	}

	fmt.Println("Initializing ghmon...")

	// Prompt for GitHub token
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your GitHub Personal Access Token (or press Enter to skip): ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	cfg := config.DefaultConfig()
	cfg.GitHub.Token = token

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Printf("Created config: %s\n", config.ConfigPath())

	// Initialize database
	db, err := database.New(config.DBPath())
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	db.Close()
	fmt.Printf("Created database: %s\n", config.DBPath())

	fmt.Println("\nghmon initialized successfully!")
	if token == "" {
		fmt.Println("\nNote: No GitHub token set. Run 'ghmon sync' after adding your token to config.")
		fmt.Println("Create a token at: https://github.com/settings/tokens")
	} else {
		fmt.Println("\nRun 'ghmon sync' to import accounts you follow.")
	}

	return nil
}
