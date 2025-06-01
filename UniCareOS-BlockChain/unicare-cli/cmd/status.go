package cmd

import (
	"os"
    "fmt"
    "github.com/spf13/cobra"
    "unicare-cli/api"
)

var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "Query node status and health",
    Example: `  unicare status
  unicare status --output json`,
    Run: func(cmd *cobra.Command, args []string) {
        output, _ := cmd.Flags().GetString("output")
        status, err := api.GetStatus()
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            os.Exit(1)
        }
        if output == "json" {
            fmt.Println(status.ToJSON())
        } else {
            fmt.Printf("Node: %s\nStatus: %s\nHeight: %d\n", status.Name, status.Status, status.Height)
        }
    },
}

func init() {
    rootCmd.AddCommand(statusCmd)
    statusCmd.Flags().StringP("output", "o", "plain", "Output format: plain|json")
}