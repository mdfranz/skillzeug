package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var assistantDirs = []string{".gemini", ".codex", ".claude"}

var runCommand = realRunCommand

var runCommandOutput = realRunCommandOutput

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

func validateRepoDir(dir string) (string, error) {
	cleaned := filepath.Clean(strings.TrimSpace(dir))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("repo directory is required")
	}
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("repo directory must be relative: %s", dir)
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("repo directory must stay inside the workspace: %s", dir)
	}

	return cleaned, nil
}

func gitTopLevel(dir string) (string, error) {
	output, err := runCommandOutput(dir, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func resolveWorkspaceDir(cwd string) (string, error) {
	root, err := gitTopLevel(cwd)
	if err == nil {
		return root, nil
	}

	return cwd, nil
}

func isConfiguredSubmodule(workspaceDir string, submoduleDir string) (bool, error) {
	output, err := runCommandOutput(
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
