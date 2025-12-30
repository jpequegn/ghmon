// cmd/daemon.go
package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run in daemon mode with scheduled fetching",
	Long:  `Runs ghmon in the background, fetching activity on a schedule.`,
	RunE:  runDaemon,
}

var (
	daemonInterval int
)

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.Flags().IntVar(&daemonInterval, "interval", 60, "Fetch interval in minutes")
}

func runDaemon(cmd *cobra.Command, args []string) error {
	interval := time.Duration(daemonInterval) * time.Minute

	fmt.Printf("Starting daemon mode (fetch every %v)\n", interval)
	fmt.Println("Press Ctrl+C to stop")

	// Run initial fetch
	fmt.Println("\nRunning initial fetch...")
	if err := runFetch(cmd, args); err != nil {
		fmt.Printf("Initial fetch error: %v\n", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			fmt.Printf("\n[%s] Running scheduled fetch...\n", time.Now().Format("15:04:05"))
			if err := runFetch(cmd, args); err != nil {
				fmt.Printf("Fetch error: %v\n", err)
			}
		case <-sigChan:
			fmt.Println("\nShutting down daemon...")
			return nil
		}
	}
}
