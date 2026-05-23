package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var updateOpts struct {
	repoURL    string
	repoBranch string
	repoDir    string
	dryRun     bool
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing workspace initialization",
	Args:  cobra.NoArgs,
	Example: strings.Join([]string{
		"  skillzeug update",
		"  skillzeug update --branch main",
		"  skillzeug update --repo git@github.com:org/new-skills.git",
		"  skillzeug update --repo https://github.com/org/skills.git --dry-run",
	}, "\n"),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws := NewWorkspace("")
		ws.RepoURL = updateOpts.repoURL
		ws.RepoBranch = updateOpts.repoBranch
		ws.RepoDir = updateOpts.repoDir
		ws.DryRun = updateOpts.dryRun

		return ws.Update()
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringVarP(&updateOpts.repoURL, "repo", "r", "", "New Git repository URL (if changing)")
	updateCmd.Flags().StringVarP(&updateOpts.repoBranch, "branch", "b", "", "New Git branch (if changing)")
	updateCmd.Flags().StringVarP(&updateOpts.repoDir, "dir", "d", "sec-skillz", "Directory of the submodule")
	updateCmd.Flags().BoolVar(&updateOpts.dryRun, "dry-run", false, "Preview changes without making them")
}

// Update resolves the workspace and runs the update logic.
func (w *Workspace) Update() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine current directory: %w", err)
	}

	workspaceDir, err := w.resolveWorkspaceDir(cwd)
	if err != nil {
		return err
	}

	w.Path = workspaceDir
	return w.UpdateInDir(workspaceDir)
}

// UpdateInDir updates the submodule and assistant symlinks in the designated workspace directory.
func (w *Workspace) UpdateInDir(workspaceDir string) error {
	normalizedRepoDir, err := validateRepoDir(w.RepoDir)
	if err != nil {
		return err
	}
	w.RepoDir = normalizedRepoDir

	// Check if submodule is configured
	submoduleConfigured, err := w.isConfiguredSubmodule(workspaceDir, w.RepoDir)
	if err != nil {
		return fmt.Errorf("failed to inspect existing submodules: %w", err)
	}

	if !submoduleConfigured {
		return fmt.Errorf("submodule %s is not configured; run 'skillzeug init' to initialize first", w.RepoDir)
	}

	submodulePath := filepath.Join(workspaceDir, w.RepoDir)
	if _, err := os.Stat(submodulePath); err != nil {
		return fmt.Errorf("submodule directory %s not found: %w", w.RepoDir, err)
	}

	// If repo URL is being changed, we need to remove and re-add the submodule
	if w.RepoURL != "" {
		if err := validateRepoURL(w.RepoURL); err != nil {
			return err
		}

		fmt.Printf("Updating submodule repository to %s...\n", w.RepoURL)

		// Deinit the old submodule
		if err := w.runCommand(workspaceDir, "git", "submodule", "deinit", "-f", "--", w.RepoDir); err != nil {
			fmt.Printf("Warning: git submodule deinit failed: %v\n", err)
		}

		// Remove it from git
		if err := w.runCommand(workspaceDir, "git", "rm", "-f", "--", w.RepoDir); err != nil {
			fmt.Printf("Warning: git rm failed: %v\n", err)
		}

		// Clean up git modules directory
		gitModulesDir := filepath.Join(workspaceDir, ".git", "modules", w.RepoDir)
		if _, err := os.Stat(gitModulesDir); err == nil {
			if err := w.removeAllPath(gitModulesDir); err != nil {
				fmt.Printf("Warning: failed to clean up git modules: %v\n", err)
			}
		}

		// Re-add with new URL and optional new branch
		gitArgs := []string{"submodule", "add"}
		if w.RepoBranch != "" {
			gitArgs = append(gitArgs, "-b", w.RepoBranch)
		}
		gitArgs = append(gitArgs, w.RepoURL, w.RepoDir)

		if err := w.runCommand(workspaceDir, "git", gitArgs...); err != nil {
			return fmt.Errorf("failed to update submodule to %s: %w\nCheck the repo URL is correct and you have network access", w.RepoURL, err)
		}

		if !w.DryRun {
			fmt.Println("[✓] Submodule repository updated")
		}
	} else if w.RepoBranch != "" {
		// If only updating branch (without changing URL), just update the config
		fmt.Printf("Updating submodule branch to %s...\n", w.RepoBranch)

		if err := w.runCommand(workspaceDir, "git", "config", "--file", ".gitmodules", fmt.Sprintf("submodule.%s.branch", w.RepoDir), w.RepoBranch); err != nil {
			return fmt.Errorf("failed to update submodule branch: %w", err)
		}
	}

	// Update the submodule
	fmt.Printf("Refreshing submodule %s...\n", w.RepoDir)
	if err := w.runCommand(workspaceDir, "git", "submodule", "update", "--remote", "--merge", "--", w.RepoDir); err != nil {
		return fmt.Errorf("failed to update submodule: %w", err)
	}

	// Refresh symlinks in all assistant directories
	for _, dir := range assistantDirs {
		skillPath := filepath.Join(workspaceDir, dir, "skills")
		targetPath := filepath.Join("..", w.RepoDir, "skills")

		// Check if symlink exists and needs updating
		if linkTarget, err := os.Readlink(skillPath); err == nil {
			if linkTarget != targetPath {
				fmt.Printf("Updating symlink in %s...\n", dir)
				if err := w.removePath(skillPath); err != nil {
					return fmt.Errorf("failed to remove symlink %s: %w", skillPath, err)
				}
				if err := w.createSymlink(targetPath, skillPath); err != nil {
					return err
				}
			}
		} else if !os.IsNotExist(err) {
			fmt.Printf("Warning: could not check symlink in %s: %v\n", dir, err)
		}
	}

	if w.DryRun {
		fmt.Println("\n[dry-run] Update complete. No changes made.")
	} else {
		fmt.Printf("Workspace updated in %s.\n", workspaceDir)
		fmt.Println("Run 'skillzeug show' to inspect the updated configuration.")
	}
	return nil
}
