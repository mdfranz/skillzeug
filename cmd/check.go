package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check system requirements and installed agents",
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Check for git
		_, err := exec.LookPath("git")
		if err != nil {
			fmt.Println("[ ] git: NOT FOUND (Please install git)")
		} else {
			fmt.Println("[✓] git: Installed")
		}

		// 2. Check for agents in home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("[!] Could not determine home directory: %v\n", err)
			return
		}

		agents := []string{".codex", ".gemini", ".claude"}
		fmt.Printf("Reviewing agents in %s:\n", homeDir)
		for _, agent := range agents {
			agentPath := filepath.Join(homeDir, agent)
			if _, err := os.Stat(agentPath); err == nil {
				fmt.Printf("[✓] %s: Found\n", agent)
			} else if os.IsNotExist(err) {
				fmt.Printf("[ ] %s: Not found\n", agent)
			} else {
				fmt.Printf("[!] %s: Error checking path: %v\n", agent, err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
