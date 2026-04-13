package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	force bool
)

type confirmModel struct {
	confirmed bool
	quitting  bool
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit
		case "n", "N", "q", "ctrl+c", "esc":
			m.confirmed = false
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	if m.quitting {
		return ""
	}
	return fmt.Sprintf("Are you sure you want to remove the workspace setup in %s? (y/n) ", repoDir)
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Remove the workspace setup (submodule and directories)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !force {
			p := tea.NewProgram(confirmModel{})
			m, err := p.Run()
			if err != nil {
				return err
			}
			if !m.(confirmModel).confirmed {
				fmt.Println("Delete cancelled.")
				return nil
			}
		}
		return runDelete()
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringVarP(&repoDir, "dir", "d", "sec-skillz", "Directory of the submodule to remove")
	deleteCmd.Flags().BoolVarP(&force, "force", "f", false, "Force removal without confirmation")
}

func runDelete() error {
	// 1. Remove submodule
	fmt.Printf("Removing submodule in %s...\n", repoDir)
	
	// git submodule deinit -f <repoDir>
	if err := runCommand("git", "submodule", "deinit", "-f", repoDir); err != nil {
		fmt.Printf("Warning: git submodule deinit failed: %v\n", err)
	}

	// git rm -f <repoDir>
	if err := runCommand("git", "rm", "-f", repoDir); err != nil {
		fmt.Printf("Warning: git rm failed: %v\n", err)
	}

	// Cleanup .git/modules/<repoDir>
	gitModulesDir := fmt.Sprintf(".git/modules/%s", repoDir)
	if _, err := os.Stat(gitModulesDir); err == nil {
		fmt.Printf("Cleaning up %s...\n", gitModulesDir)
		if err := os.RemoveAll(gitModulesDir); err != nil {
			fmt.Printf("Warning: failed to remove %s: %v\n", gitModulesDir, err)
		}
	}

	// 2. Remove directories (which contain the symlinks)
	dirs := []string{".gemini", ".codex", ".claude"}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); err == nil {
			fmt.Printf("Removing directory %s...\n", dir)
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("failed to remove directory %s: %w", dir, err)
			}
		}
	}

	fmt.Println("Workspace cleanup complete!")
	return nil
}
