package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var showOpts struct {
	repoDir string
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current workspace configuration and status",
	Args:  cobra.NoArgs,
	Example: strings.Join([]string{
		"  skillzeug show",
		"  skillzeug show --dir sec-skillz",
	}, "\n"),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws := NewWorkspace("")
		ws.RepoDir = showOpts.repoDir

		return ws.Show()
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().StringVarP(&showOpts.repoDir, "dir", "d", "sec-skillz", "Directory for the submodule")
}

// Show resolves the workspace and runs the show configuration logic.
func (w *Workspace) Show() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine current directory: %w", err)
	}

	workspaceDir, err := w.resolveWorkspaceDir(cwd)
	if err != nil {
		return err
	}

	w.Path = workspaceDir
	return w.ShowInDir(workspaceDir)
}

// ShowInDir prints status information for the workspace directory.
func (w *Workspace) ShowInDir(workspaceDir string) error {
	normalizedRepoDir, err := validateRepoDir(w.RepoDir)
	if err != nil {
		return err
	}
	w.RepoDir = normalizedRepoDir

	fmt.Println("Workspace Configuration Status:")
	fmt.Println("-------------------------------")
	fmt.Printf("Workspace root: %s\n", workspaceDir)
	fmt.Printf("Managed submodule: %s\n\n", w.RepoDir)

	// 1. Check Submodule
	submodulePath := filepath.Join(workspaceDir, w.RepoDir)
	if info, err := os.Stat(submodulePath); err == nil && info.IsDir() {
		fmt.Printf("[✓] Submodule directory: %s\n", w.RepoDir)
		// Check for skills subdir
		skillsPath := filepath.Join(submodulePath, "skills")
		if sInfo, sErr := os.Stat(skillsPath); sErr == nil && sInfo.IsDir() {
			fmt.Println("    [✓] Skills directory found")
		} else {
			fmt.Println("    [!] Skills directory MISSING inside submodule")
		}
	} else {
		fmt.Printf("[ ] Submodule directory: %s (NOT FOUND)\n", w.RepoDir)
		fmt.Printf("    Run 'skillzeug init' to initialize\n")
	}

	// 2. Check Assistant Directories and Symlinks
	for _, dir := range assistantDirs {
		dirPath := filepath.Join(workspaceDir, dir)
		if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
			fmt.Printf("[✓] Assistant directory: %s\n", dir)

			skillPath := filepath.Join(dirPath, "skills")
			lInfo, err := os.Lstat(skillPath)
			if err != nil {
				fmt.Printf("    [ ] Symlink 'skills': NOT FOUND\n")
			} else if lInfo.Mode()&os.ModeSymlink != 0 {
				target, _ := os.Readlink(skillPath)
				fmt.Printf("    [✓] Symlink 'skills' -> %s\n", target)

				// Check if target is valid
				if _, err := os.Stat(skillPath); err == nil {
					fmt.Println("        [✓] Symlink target is VALID")
				} else {
					fmt.Println("        [!] Symlink target is BROKEN")
					fmt.Println("            Run 'skillzeug update' to repair")
				}
			} else {
				fmt.Printf("    [!] 'skills' exists but is NOT a symlink\n")
			}
		} else {
			fmt.Printf("[ ] Assistant directory: %s (NOT FOUND)\n", dir)
		}
	}

	return nil
}
