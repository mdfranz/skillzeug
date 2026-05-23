package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var deleteOpts struct {
	repoDir   string
	force     bool
	dryRun    bool
	pruneDirs bool
}

type confirmModel struct {
	confirmed bool
	quitting  bool
	repoDir   string
	pruneDirs bool
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
	if m.pruneDirs {
		return fmt.Sprintf("Remove the workspace initialization for %s? This will delete %s and the assistant directories (.codex, .gemini, .claude, .agents). (y/n) ", m.repoDir, m.repoDir)
	}
	return fmt.Sprintf("Remove the workspace initialization for %s? This will delete %s and the 'skills' symlinks in assistant directories. (y/n) ", m.repoDir, m.repoDir)
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
		if !deleteOpts.force {
			if !isInputTerminal() {
				return fmt.Errorf("confirmation required; use --force in non-interactive environments")
			}
			p := tea.NewProgram(confirmModel{
				repoDir:   deleteOpts.repoDir,
				pruneDirs: deleteOpts.pruneDirs,
			})
			m, err := p.Run()
			if err != nil {
				return err
			}
			if !m.(confirmModel).confirmed {
				fmt.Println("Delete cancelled.")
				return nil
			}
		}

		ws := NewWorkspace("")
		ws.RepoDir = deleteOpts.repoDir
		ws.Force = deleteOpts.force
		ws.DryRun = deleteOpts.dryRun
		ws.PruneDirs = deleteOpts.pruneDirs

		return ws.Delete()
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringVarP(&deleteOpts.repoDir, "dir", "d", "sec-skillz", "Directory of the submodule to remove")
	deleteCmd.Flags().BoolVarP(&deleteOpts.force, "force", "f", false, "Force removal without confirmation")
	deleteCmd.Flags().BoolVar(&deleteOpts.dryRun, "dry-run", false, "Preview changes without making them")
	deleteCmd.Flags().BoolVar(&deleteOpts.pruneDirs, "prune-dirs", false, "Also remove assistant directories (.codex, .gemini, .claude, .agents)")
}

// Delete resolves the workspace and runs the deletion logic.
func (w *Workspace) Delete() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine current directory: %w", err)
	}

	workspaceDir, err := w.resolveWorkspaceDir(cwd)
	if err != nil {
		return err
	}

	w.Path = workspaceDir
	return w.DeleteInDir(workspaceDir)
}

// DeleteInDir deletes the submodule configuration and associated symlinks in the designated workspace directory.
func (w *Workspace) DeleteInDir(workspaceDir string) error {
	normalizedRepoDir, err := validateRepoDir(w.RepoDir)
	if err != nil {
		return err
	}
	w.RepoDir = normalizedRepoDir

	// 1. Remove submodule
	fmt.Printf("Removing submodule in %s...\n", w.RepoDir)

	// git submodule deinit -f <repoDir>
	if err := w.runCommand(workspaceDir, "git", "submodule", "deinit", "-f", "--", w.RepoDir); err != nil {
		fmt.Printf("Warning: git submodule deinit failed: %v\n", err)
	}

	// git rm -f <repoDir>
	if err := w.runCommand(workspaceDir, "git", "rm", "-f", "--", w.RepoDir); err != nil {
		fmt.Printf("Warning: git rm failed: %v\n", err)
	}

	// Cleanup .git/modules/<repoDir>
	gitModulesDir := filepath.Join(workspaceDir, ".git", "modules", w.RepoDir)
	if _, err := os.Stat(gitModulesDir); err == nil {
		fmt.Printf("Cleaning up %s...\n", gitModulesDir)
		if err := w.removeAllPath(gitModulesDir); err != nil {
			fmt.Printf("Warning: failed to remove %s: %v\n", gitModulesDir, err)
		}
	}

	// 2. Remove 'skills' symlinks or entire directories based on --prune-dirs
	if w.PruneDirs {
		// Remove entire directories
		for _, dir := range assistantDirs {
			dirPath := filepath.Join(workspaceDir, dir)
			if _, err := os.Stat(dirPath); err == nil {
				fmt.Printf("Removing directory %s...\n", dir)
				if err := w.removeAllPath(dirPath); err != nil {
					return fmt.Errorf("failed to remove directory %s: %w", dir, err)
				}
			}
		}
	} else {
		// Remove only 'skills' symlinks
		for _, dir := range assistantDirs {
			skillPath := filepath.Join(workspaceDir, dir, "skills")
			if _, err := os.Lstat(skillPath); err == nil {
				fmt.Printf("Removing symlink %s...\n", skillPath)
				if err := w.removePath(skillPath); err != nil {
					return fmt.Errorf("failed to remove symlink %s: %w", skillPath, err)
				}
			} else if !os.IsNotExist(err) {
				fmt.Printf("Warning: could not check symlink %s: %v\n", skillPath, err)
			}
		}
	}

	if w.DryRun {
		fmt.Println("\n[dry-run] Cleanup complete. No changes made.")
	} else {
		fmt.Println("Workspace cleanup complete!")
	}
	return nil
}
