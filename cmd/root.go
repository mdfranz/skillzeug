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
	Use:           "skillzeug",
	Short:         "Manage workspace skill submodules and assistant directories",
	Long:          `Skillzeug configures a workspace by adding a skills repository as a Git submodule and wiring assistant-specific directories to that shared skills tree.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}
