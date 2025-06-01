package cmd

import (
    "fmt"
    "os"
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "unicare",
    Short: "UniCareOS Blockchain CLI",
    Long:  "A command-line tool for managing and interacting with UniCareOS blockchain nodes.",
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}