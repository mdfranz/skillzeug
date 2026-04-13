package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRepoDir(t *testing.T) {
	absolutePath := filepath.Join(string(filepath.Separator), "tmp", "skills")

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "accepts simple name", input: "sec-skillz", want: "sec-skillz"},
		{name: "cleans nested relative path", input: "./nested/skills", want: filepath.Join("nested", "skills")},
		{name: "rejects empty value", input: "", wantErr: true},
		{name: "rejects parent traversal", input: "../skills", wantErr: true},
		{name: "rejects absolute path", input: absolutePath, wantErr: true},
	}

	for _, tt := range tests {
		got, err := validateRepoDir(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("%s: expected error, got nil", tt.name)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.name, err)
		}
		if got != tt.want {
			t.Fatalf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestRunDeleteInDirRejectsUnsafeRepoDir(t *testing.T) {
	restore := stubCommandFns(
		func(string, string, ...string) error {
			t.Fatal("runCommand should not be called for an invalid repo dir")
			return nil
		},
		func(string, string, ...string) ([]byte, error) {
			t.Fatal("runCommandOutput should not be called for an invalid repo dir")
			return nil, nil
		},
	)
	defer restore()

	repoDir = "../danger"

	err := runDeleteInDir(t.TempDir())
	if err == nil {
		t.Fatal("expected delete to reject repoDir traversal")
	}
}

func TestRunSetupInDirFailsWhenSubmoduleAddFails(t *testing.T) {
	restore := stubCommandFns(
		func(_ string, name string, args ...string) error {
			if name == "git" && len(args) >= 2 && args[0] == "submodule" && args[1] == "add" {
				return errors.New("boom")
			}
			return nil
		},
		func(string, string, ...string) ([]byte, error) {
			return nil, errors.New("no gitmodules")
		},
	)
	defer restore()

	repoURL = "https://example.com/skills.git"
	repoBranch = ""
	repoDir = "sec-skillz"

	workspaceDir := t.TempDir()
	err := runSetupInDir(workspaceDir)
	if err == nil {
		t.Fatal("expected setup to fail when submodule add fails")
	}
	if !strings.Contains(err.Error(), "failed to add submodule") {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, dir := range assistantDirs {
		if _, statErr := os.Stat(filepath.Join(workspaceDir, dir)); !os.IsNotExist(statErr) {
			t.Fatalf("assistant dir %s should not be created on setup failure", dir)
		}
	}
}

func TestRunSetupInDirSkipsExistingSubmoduleAndScopesUpdate(t *testing.T) {
	var commands []string
	restore := stubCommandFns(
		func(_ string, name string, args ...string) error {
			commands = append(commands, strings.Join(append([]string{name}, args...), " "))
			return nil
		},
		func(_ string, name string, args ...string) ([]byte, error) {
			if name == "git" && len(args) >= 5 && args[0] == "config" {
				return []byte("submodule.skills.path sec-skillz\n"), nil
			}
			return nil, errors.New("unexpected command")
		},
	)
	defer restore()

	repoURL = "https://example.com/skills.git"
	repoBranch = ""
	repoDir = "sec-skillz"

	workspaceDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspaceDir, repoDir), 0755); err != nil {
		t.Fatalf("failed to create submodule dir: %v", err)
	}

	if err := runSetupInDir(workspaceDir); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	for _, command := range commands {
		if strings.Contains(command, "submodule add") {
			t.Fatalf("setup should not add an already configured submodule: %v", commands)
		}
	}

	wantUpdate := "git submodule update --remote --merge -- sec-skillz"
	foundUpdate := false
	for _, command := range commands {
		if command == wantUpdate {
			foundUpdate = true
		}
	}
	if !foundUpdate {
		t.Fatalf("expected scoped submodule update %q, got %v", wantUpdate, commands)
	}
}

func TestResolveWorkspaceDirUsesGitTopLevel(t *testing.T) {
	rootDir := t.TempDir()
	subDir := filepath.Join(rootDir, "nested", "project")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	restore := stubCommandFns(
		func(string, string, ...string) error {
			return nil
		},
		func(dir string, name string, args ...string) ([]byte, error) {
			if dir != subDir {
				t.Fatalf("gitTopLevel called with dir %q, want %q", dir, subDir)
			}
			if name != "git" || len(args) != 2 || args[0] != "rev-parse" || args[1] != "--show-toplevel" {
				t.Fatalf("unexpected command: %s %v", name, args)
			}
			return []byte(rootDir + "\n"), nil
		},
	)
	defer restore()

	got, err := resolveWorkspaceDir(subDir)
	if err != nil {
		t.Fatalf("resolveWorkspaceDir returned error: %v", err)
	}
	if got != rootDir {
		t.Fatalf("got %q, want %q", got, rootDir)
	}
}

func stubCommandFns(
	run func(string, string, ...string) error,
	output func(string, string, ...string) ([]byte, error),
) func() {
	originalRun := runCommand
	originalOutput := runCommandOutput
	runCommand = run
	runCommandOutput = output

	return func() {
		runCommand = originalRun
		runCommandOutput = originalOutput
	}
}
