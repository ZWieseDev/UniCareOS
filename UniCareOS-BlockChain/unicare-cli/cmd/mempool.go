package cmd

import (
    "fmt"
    "os"
    "encoding/json"
    "github.com/spf13/cobra"
    "unicare-cli/api"
)

var mempoolCmd = &cobra.Command{
    Use:   "mempool",
    Short: "Query the current mempool contents",
    Run: func(cmd *cobra.Command, args []string) {
        txs, err := api.GetMempool()
        if err != nil {
            fmt.Println("Failed to fetch mempool:", err)
            os.Exit(1)
        }
        output, _ := cmd.Flags().GetString("output")
        if output == "json" {
            b, _ := json.MarshalIndent(txs, "", "  ")
            fmt.Println(string(b))
        } else {
            fmt.Printf("%d transactions in mempool:\n", len(txs))
            for i, tx := range txs {
                fmt.Printf("%d. Author: %s | Desc: %.40s... | Emotion: %s | Parent: %s\n", i+1, tx.Author, tx.Description, tx.Emotion, tx.ParentID)
            }
        }
    },
}

func init() {
    rootCmd.AddCommand(mempoolCmd)
    mempoolCmd.Flags().StringP("output", "o", "plain", "Output format: plain|json")
}
