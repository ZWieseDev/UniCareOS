package cmd

import (
    "fmt"
    "os"
    "github.com/spf13/cobra"
    "unicare-cli/api"
)

var eventCmd = &cobra.Command{
    Use:   "event",
    Short: "Event-related operations (submit, finalize, etc)",
}

var eventSubmitCmd = &cobra.Command{
    Use:   "submit",
    Short: "Submit a new memory event",
    Run: func(cmd *cobra.Command, args []string) {
        desc, _ := cmd.Flags().GetString("description")
        author, _ := cmd.Flags().GetString("author")
        emotion, _ := cmd.Flags().GetString("emotion")
        parent, _ := cmd.Flags().GetString("parent")
        if desc == "" || author == "" {
            fmt.Println("Description and author are required.")
            os.Exit(1)
        }
        err := api.SubmitMemory(desc, author, emotion, parent)
        if err != nil {
            fmt.Println("Submission failed:", err)
            os.Exit(1)
        }
        fmt.Println("Memory event submitted successfully!")
    },
}

func init() {
    rootCmd.AddCommand(eventCmd)
    eventCmd.AddCommand(eventSubmitCmd)
    eventSubmitCmd.Flags().String("description", "", "Event description (required)")
    eventSubmitCmd.Flags().String("author", "", "Author (required)")
    eventSubmitCmd.Flags().String("emotion", "", "Emotion")
    eventSubmitCmd.Flags().String("parent", "", "Parent ID")
}
