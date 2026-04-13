package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

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
		return runUpdate()
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringVarP(&repoURL, "repo", "r", "", "New Git repository URL (if changing)")
	updateCmd.Flags().StringVarP(&repoBranch, "branch", "b", "", "New Git branch (if changing)")
	updateCmd.Flags().StringVarP(&repoDir, "dir", "d", "sec-skillz", "Directory of the submodule")
	updateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without making them")
}

func runUpdateInDir(workspaceDir string) error {
	normalizedRepoDir, err := validateRepoDir(repoDir)
	if err != nil {
		return err
	}
	repoDir = normalizedRepoDir

	// Check if submodule is configured
	submoduleConfigured, err := isConfiguredSubmodule(workspaceDir, repoDir)
	if err != nil {
		return fmt.Errorf("failed to inspect existing submodules: %w", err)
	}

	if !submoduleConfigured {
		return fmt.Errorf("submodule %s is not configured; run 'skillzeug init' to initialize first", repoDir)
	}

	submodulePath := filepath.Join(workspaceDir, repoDir)
	if _, err := os.Stat(submodulePath); err != nil {
		return fmt.Errorf("submodule directory %s not found: %w", repoDir, err)
	}

	// If repo URL is being changed, we need to remove and re-add the submodule
	if repoURL != "" {
		if err := validateRepoURL(repoURL); err != nil {
			return err
		}

		fmt.Printf("Updating submodule repository to %s...\n", repoURL)

		if !dryRun {
			// Deinit the old submodule
			if err := runCommand(workspaceDir, "git", "submodule", "deinit", "-f", "--", repoDir); err != nil {
				fmt.Printf("Warning: git submodule deinit failed: %v\n", err)
			}

			// Remove it from git
			if err := runCommand(workspaceDir, "git", "rm", "-f", "--", repoDir); err != nil {
				fmt.Printf("Warning: git rm failed: %v\n", err)
			}

			// Clean up git modules directory
			gitModulesDir := filepath.Join(workspaceDir, ".git", "modules", repoDir)
			if _, err := os.Stat(gitModulesDir); err == nil {
				if err := os.RemoveAll(gitModulesDir); err != nil {
					fmt.Printf("Warning: failed to clean up git modules: %v\n", err)
				}
			}

			// Re-add with new URL and optional new branch
			gitArgs := []string{"submodule", "add"}
			if repoBranch != "" {
				gitArgs = append(gitArgs, "-b", repoBranch)
			}
			gitArgs = append(gitArgs, repoURL, repoDir)

			if err := runCommand(workspaceDir, "git", gitArgs...); err != nil {
				return fmt.Errorf("failed to update submodule to %s: %w\nCheck the repo URL is correct and you have network access", repoURL, err)
			}

			fmt.Println("[✓] Submodule repository updated")
		} else {
			fmt.Printf("[dry-run] would remove old submodule\n")
			fmt.Printf("[dry-run] would add new submodule: %s\n", repoURL)
		}
	} else if repoBranch != "" {
		// If only updating branch (without changing URL), just update the config
		fmt.Printf("Updating submodule branch to %s...\n", repoBranch)

		if !dryRun {
			if err := runCommand(workspaceDir, "git", "config", "--file", ".gitmodules", fmt.Sprintf("submodule.%s.branch", repoDir), repoBranch); err != nil {
				return fmt.Errorf("failed to update submodule branch: %w", err)
			}
		} else {
			fmt.Printf("[dry-run] would update branch in .gitmodules to %s\n", repoBranch)
		}
	}

	// Update the submodule
	fmt.Printf("Refreshing submodule %s...\n", repoDir)
	if !dryRun {
		if err := runCommand(workspaceDir, "git", "submodule", "update", "--remote", "--merge", "--", repoDir); err != nil {
			return fmt.Errorf("failed to update submodule: %w", err)
		}
	} else {
		fmt.Printf("[dry-run] would run: git submodule update --remote --merge -- %s\n", repoDir)
	}

	// Refresh symlinks in all assistant directories
	for _, dir := range assistantDirs {
		skillPath := filepath.Join(workspaceDir, dir, "skills")
		targetPath := filepath.Join("..", repoDir, "skills")

		// Check if symlink exists and needs updating
		if linkTarget, err := os.Readlink(skillPath); err == nil {
			if linkTarget != targetPath {
				fmt.Printf("Updating symlink in %s...\n", dir)
				if !dryRun {
					if err := os.Remove(skillPath); err != nil {
						return fmt.Errorf("failed to remove symlink %s: %w", skillPath, err)
					}
					if err := os.Symlink(targetPath, skillPath); err != nil {
						return fmt.Errorf("failed to create symlink %s: %w", skillPath, err)
					}
				} else {
					fmt.Printf("[dry-run] would update symlink %s -> %s\n", skillPath, targetPath)
				}
			}
		} else if !os.IsNotExist(err) {
			fmt.Printf("Warning: could not check symlink in %s: %v\n", dir, err)
		}
	}

	if dryRun {
		fmt.Println("\n[dry-run] Update complete. No changes made.")
	} else {
		fmt.Printf("Workspace updated in %s.\n", workspaceDir)
		fmt.Println("Run 'skillzeug show' to inspect the updated configuration.")
	}
	return nil
}

func runUpdate() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine current directory: %w", err)
	}

	workspaceDir, err := resolveWorkspaceDir(cwd)
	if err != nil {
		return err
	}

	return runUpdateInDir(workspaceDir)
}
