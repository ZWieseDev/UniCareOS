package cmd

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
	"unicare-cli/api"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Query node health summary",
	Run: func(cmd *cobra.Command, args []string) {
		health, err := api.GetHealthMetrics()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Node Health: %s\n", health.Status)
		fmt.Printf("Uptime: %ds\n", health.Metrics.UptimeSeconds)
		fmt.Printf("Block Height: %d\n", health.Metrics.BlockHeight)
		fmt.Printf("Peer Count: %d\n", health.Metrics.PeerCount)
		fmt.Printf("CPU Load: %.2f%%\n", health.Metrics.CPULoadPercent)
		fmt.Printf("Memory Usage: %.2f MB\n", health.Metrics.MemoryMB)
		fmt.Printf("Disk Free: %.2f MB\n", health.Metrics.DiskFreeMB)
		fmt.Printf("Sync Lag: %ds\n", health.Metrics.SyncLagSeconds)
		fmt.Printf("Last Block Time: %s\n", health.Metrics.LastBlockTime)

	},
}

var livenessCmd = &cobra.Command{
	Use:   "liveness",
	Short: "Check node liveness",
	Run: func(cmd *cobra.Command, args []string) {
		alive, err := api.GetLiveness()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Liveness: %v\n", alive)
	},
}

var readinessCmd = &cobra.Command{
	Use:   "readiness",
	Short: "Check node readiness",
	Run: func(cmd *cobra.Command, args []string) {
		ready, err := api.GetReadiness()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Readiness: %v\n", ready)
	},
}

func init() {
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(livenessCmd)
	rootCmd.AddCommand(readinessCmd)
}
