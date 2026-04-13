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
	force      bool
	pruneDirs  bool
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
	if pruneDirs {
		return fmt.Sprintf("Remove the workspace initialization for %s? This will delete %s and the assistant directories (.codex, .gemini, .claude). (y/n) ", repoDir, repoDir)
	}
	return fmt.Sprintf("Remove the workspace initialization for %s? This will delete %s and the 'skills' symlinks in assistant directories. (y/n) ", repoDir, repoDir)
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Remove the workspace initialization (submodule and directories)",
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
	deleteCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without making them")
	deleteCmd.Flags().BoolVar(&pruneDirs, "prune-dirs", false, "Also remove assistant directories (.codex, .gemini, .claude)")
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
	if !dryRun {
		if err := runCommand(workspaceDir, "git", "submodule", "deinit", "-f", "--", repoDir); err != nil {
			fmt.Printf("Warning: git submodule deinit failed: %v\n", err)
		}

		// git rm -f <repoDir>
		if err := runCommand(workspaceDir, "git", "rm", "-f", "--", repoDir); err != nil {
			fmt.Printf("Warning: git rm failed: %v\n", err)
		}
	} else {
		fmt.Printf("[dry-run] would run: git submodule deinit -f -- %s\n", repoDir)
		fmt.Printf("[dry-run] would run: git rm -f -- %s\n", repoDir)
	}

	// Cleanup .git/modules/<repoDir>
	gitModulesDir := filepath.Join(workspaceDir, ".git", "modules", repoDir)
	if _, err := os.Stat(gitModulesDir); err == nil {
		fmt.Printf("Cleaning up %s...\n", gitModulesDir)
		if !dryRun {
			if err := os.RemoveAll(gitModulesDir); err != nil {
				fmt.Printf("Warning: failed to remove %s: %v\n", gitModulesDir, err)
			}
		} else {
			fmt.Printf("[dry-run] would remove: %s\n", gitModulesDir)
		}
	}

	// 2. Remove 'skills' symlinks or entire directories based on --prune-dirs
	if pruneDirs {
		// Remove entire directories (old behavior)
		for _, dir := range assistantDirs {
			dirPath := filepath.Join(workspaceDir, dir)
			if _, err := os.Stat(dirPath); err == nil {
				fmt.Printf("Removing directory %s...\n", dir)
				if !dryRun {
					if err := os.RemoveAll(dirPath); err != nil {
						return fmt.Errorf("failed to remove directory %s: %w", dir, err)
					}
				} else {
					fmt.Printf("[dry-run] would remove: %s\n", dirPath)
				}
			}
		}
	} else {
		// Remove only 'skills' symlinks (new safe behavior)
		for _, dir := range assistantDirs {
			skillPath := filepath.Join(workspaceDir, dir, "skills")
			if _, err := os.Lstat(skillPath); err == nil {
				fmt.Printf("Removing symlink %s...\n", skillPath)
				if !dryRun {
					if err := os.Remove(skillPath); err != nil {
						return fmt.Errorf("failed to remove symlink %s: %w", skillPath, err)
					}
				} else {
					fmt.Printf("[dry-run] would remove: %s\n", skillPath)
				}
			} else if !os.IsNotExist(err) {
				fmt.Printf("Warning: could not check symlink %s: %v\n", skillPath, err)
			}
		}
	}

	if dryRun {
		fmt.Println("\n[dry-run] Cleanup complete. No changes made.")
	} else {
		fmt.Println("Workspace cleanup complete!")
	}
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
