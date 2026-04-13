package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current workspace configuration and status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShow()
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().StringVarP(&repoDir, "dir", "d", "sec-skillz", "Directory for the submodule")
}

func runShow() error {
	fmt.Println("Workspace Configuration Status:")
	fmt.Println("-------------------------------")

	// 1. Check Submodule
	if info, err := os.Stat(repoDir); err == nil && info.IsDir() {
		fmt.Printf("[✓] Submodule directory: %s\n", repoDir)
		// Check for skills subdir
		skillsPath := filepath.Join(repoDir, "skills")
		if sInfo, sErr := os.Stat(skillsPath); sErr == nil && sInfo.IsDir() {
			fmt.Println("    [✓] Skills directory found")
		} else {
			fmt.Println("    [!] Skills directory MISSING inside submodule")
		}
	} else {
		fmt.Printf("[ ] Submodule directory: %s (NOT FOUND)\n", repoDir)
	}

	// 2. Check Assistant Directories and Symlinks
	dirs := []string{".gemini", ".codex", ".claude"}
	for _, dir := range dirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			fmt.Printf("[✓] Assistant directory: %s\n", dir)
			
			skillPath := filepath.Join(dir, "skills")
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
