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

func TestRunInitInDirFailsWhenSubmoduleAddFails(t *testing.T) {
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
	err := runInitInDir(workspaceDir)
	if err == nil {
		t.Fatal("expected initialization to fail when submodule add fails")
	}
	if !strings.Contains(err.Error(), "failed to add submodule") {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, dir := range assistantDirs {
		if _, statErr := os.Stat(filepath.Join(workspaceDir, dir)); !os.IsNotExist(statErr) {
			t.Fatalf("assistant dir %s should not be created on initialization failure", dir)
		}
	}
}

func TestRunInitInDirSkipsExistingSubmoduleAndScopesUpdate(t *testing.T) {
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

	if err := runInitInDir(workspaceDir); err != nil {
		t.Fatalf("initialization failed: %v", err)
	}

	for _, command := range commands {
		if strings.Contains(command, "submodule add") {
			t.Fatalf("initialization should not add an already configured submodule: %v", commands)
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

func TestResolveWorkspaceDirReturnsErrorWhenGitFails(t *testing.T) {
	workDir := t.TempDir()

	restore := stubCommandFns(
		func(string, string, ...string) error {
			return errors.New("git not found")
		},
		func(string, string, ...string) ([]byte, error) {
			return nil, errors.New("git not found")
		},
	)
	defer restore()

	_, err := resolveWorkspaceDir(workDir)
	if err == nil {
		t.Fatal("expected resolveWorkspaceDir to return error when git fails")
	}
	if !strings.Contains(err.Error(), "not inside a git repository") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateRepoURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "accepts https URL", input: "https://github.com/org/repo.git", wantErr: false},
		{name: "accepts http URL", input: "http://github.com/org/repo.git", wantErr: false},
		{name: "accepts git@ SSH", input: "git@github.com:org/repo.git", wantErr: false},
		{name: "accepts ssh:// URL", input: "ssh://git@github.com/org/repo.git", wantErr: false},
		{name: "accepts git:// URL", input: "git://github.com/org/repo.git", wantErr: false},
		{name: "accepts file:// URL", input: "file:///path/to/repo.git", wantErr: false},
		{name: "rejects empty URL", input: "", wantErr: true},
		{name: "rejects invalid scheme", input: "ftp://github.com/repo.git", wantErr: true},
		{name: "rejects URL with null byte", input: "https://github.com/repo\x00.git", wantErr: true},
		{name: "rejects URL with newline", input: "https://github.com/repo\n.git", wantErr: true},
	}

	for _, tt := range tests {
		err := validateRepoURL(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("%s: expected error, got nil", tt.name)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.name, err)
		}
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
