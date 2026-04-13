package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	return fmt.Sprintf("Remove the workspace setup for %s? This deletes %s and the managed assistant directories in the workspace. (y/n) ", repoDir, repoDir)
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Remove the workspace setup (submodule and directories)",
	Args:  cobra.NoArgs,
	Example: strings.Join([]string{
		"  skillzeug delete",
		"  skillzeug delete --force --dir sec-skillz",
	}, "\n"),
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

func runDeleteInDir(workspaceDir string) error {
	normalizedRepoDir, err := validateRepoDir(repoDir)
	if err != nil {
		return err
	}
	repoDir = normalizedRepoDir

	// 1. Remove submodule
	fmt.Printf("Removing submodule in %s...\n", repoDir)

	// git submodule deinit -f <repoDir>
	if err := runCommand(workspaceDir, "git", "submodule", "deinit", "-f", "--", repoDir); err != nil {
		fmt.Printf("Warning: git submodule deinit failed: %v\n", err)
	}

	// git rm -f <repoDir>
	if err := runCommand(workspaceDir, "git", "rm", "-f", "--", repoDir); err != nil {
		fmt.Printf("Warning: git rm failed: %v\n", err)
	}

	// Cleanup .git/modules/<repoDir>
	gitModulesDir := filepath.Join(workspaceDir, ".git", "modules", repoDir)
	if _, err := os.Stat(gitModulesDir); err == nil {
		fmt.Printf("Cleaning up %s...\n", gitModulesDir)
		if err := os.RemoveAll(gitModulesDir); err != nil {
			fmt.Printf("Warning: failed to remove %s: %v\n", gitModulesDir, err)
		}
	}

	// 2. Remove directories (which contain the symlinks)
	for _, dir := range assistantDirs {
		dirPath := filepath.Join(workspaceDir, dir)
		if _, err := os.Stat(dirPath); err == nil {
			fmt.Printf("Removing directory %s...\n", dir)
			if err := os.RemoveAll(dirPath); err != nil {
				return fmt.Errorf("failed to remove directory %s: %w", dir, err)
			}
		}
	}

	fmt.Println("Workspace cleanup complete!")
	return nil
}

func runDelete() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine current directory: %w", err)
	}

	workspaceDir, err := resolveWorkspaceDir(cwd)
	if err != nil {
		return err
	}

	return runDeleteInDir(workspaceDir)
}
