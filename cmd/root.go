package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	repoURL    string
	repoBranch string
	repoDir    string
)

var rootCmd = &cobra.Command{
	Use:   "skillzeug",
	Short: "Skillzeug is a CLI tool for workspace setup",
	Long:  `A Golang CLI tool that implements workspace setup based on sec-skillz requirements.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}
