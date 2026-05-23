package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mattn/go-isatty"
)

var assistantDirs = []string{".gemini", ".codex", ".claude", ".agents"}

// Workspace encapsulates the environment and configuration for a skillzeug workspace operation.
type Workspace struct {
	Path        string
	RepoURL     string
	RepoBranch  string
	RepoDir     string
	Interactive bool
	DryRun      bool
	Force       bool
	PruneDirs   bool

	// Function pointers to allow stubbing in tests without mutating global variables.
	runCmd       func(dir string, name string, args ...string) error
	runCmdOutput func(dir string, name string, args ...string) ([]byte, error)
}

// NewWorkspace creates a new Workspace instance with real execution commands by default.
func NewWorkspace(path string) *Workspace {
	return &Workspace{
		Path:         path,
		runCmd:       realRunCommand,
		runCmdOutput: realRunCommandOutput,
	}
}

func realRunCommand(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func realRunCommandOutput(dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}

	return stdout.Bytes(), nil
}

// isInputTerminal checks if standard input is a terminal/TTY.
func isInputTerminal() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

func validateRepoURL(url string) error {
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		return fmt.Errorf("repo URL is required; use --repo or run without flags for interactive mode")
	}
	if strings.ContainsRune(trimmed, '\x00') || strings.ContainsRune(trimmed, '\n') {
		return fmt.Errorf("repo URL contains invalid characters")
	}

	// Check for valid git URL schemes
	validSchemes := []string{"https://", "http://", "git@", "ssh://", "git://", "file://"}
	hasValidScheme := false
	for _, scheme := range validSchemes {
		if strings.HasPrefix(trimmed, scheme) {
			hasValidScheme = true
			break
		}
	}
	if !hasValidScheme {
		return fmt.Errorf("repo URL must start with https://, http://, git@, ssh://, git://, or file://")
	}

	return nil
}

func validateRepoDir(dir string) (string, error) {
	cleaned := filepath.Clean(strings.TrimSpace(dir))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("repo directory is required (use --dir or accept the default 'sec-skillz')")
	}
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("repo directory must be relative: %s", dir)
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("repo directory must stay inside the workspace: %s", dir)
	}

	return cleaned, nil
}

func (w *Workspace) gitTopLevel(dir string) (string, error) {
	output, err := w.runCmdOutput(dir, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func (w *Workspace) resolveWorkspaceDir(cwd string) (string, error) {
	root, err := w.gitTopLevel(cwd)
	if err == nil {
		return root, nil
	}

	return "", fmt.Errorf("not inside a git repository; run 'git init' first or 'skillzeug init' to initialize one")
}

func (w *Workspace) isConfiguredSubmodule(workspaceDir string, submoduleDir string) (bool, error) {
	output, err := w.runCmdOutput(
		workspaceDir,
		"git",
		"config",
		"--file",
		".gitmodules",
		"--get-regexp",
		`^submodule\..*\.path$`,
	)
	if err != nil {
		return false, nil
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && filepath.Clean(fields[1]) == submoduleDir {
			return true, nil
		}
	}

	return false, nil
}

func (w *Workspace) runCommand(dir string, name string, args ...string) error {
	if w.DryRun {
		fmt.Printf("[dry-run] would run in %s: %s %s\n", dir, name, strings.Join(args, " "))
		return nil
	}
	return w.runCmd(dir, name, args...)
}

func (w *Workspace) removePath(path string) error {
	if w.DryRun {
		fmt.Printf("[dry-run] would remove: %s\n", path)
		return nil
	}
	return os.Remove(path)
}

func (w *Workspace) removeAllPath(path string) error {
	if w.DryRun {
		fmt.Printf("[dry-run] would remove recursively: %s\n", path)
		return nil
	}
	return os.RemoveAll(path)
}

func (w *Workspace) createSymlink(target string, link string) error {
	fmt.Printf("Creating symlink %s -> %s\n", link, target)
	if w.DryRun {
		return nil
	}
	err := os.Symlink(target, link)
	if err != nil {
		if runtime.GOOS == "windows" {
			return fmt.Errorf("failed to create symlink: %w\nNote: On Windows, creating symlinks requires Administrator privileges or Developer Mode enabled.", err)
		}
		return fmt.Errorf("failed to create symlink %s: %w", link, err)
	}
	return nil
}
