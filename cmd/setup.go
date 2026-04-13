package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	interactive bool
)

type model struct {
	inputs  []textinput.Model
	focused int
	err     error
	done    bool
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyTab, tea.KeyShiftTab, tea.KeyEnter:
			s := msg.String()

			if s == "enter" && m.focused == len(m.inputs)-1 {
				m.done = true
				return m, tea.Quit
			}

			if s == "shift+tab" {
				m.focused--
			} else {
				m.focused++
			}

			if m.focused > len(m.inputs)-1 {
				m.focused = 0
			} else if m.focused < 0 {
				m.focused = len(m.inputs) - 1
			}

			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focused {
					cmds = append(cmds, m.inputs[i].Focus())
				} else {
					m.inputs[i].Blur()
				}
			}

			return m, tea.Batch(cmds...)
		}
	}

	for i := range m.inputs {
		m.inputs[i], _ = m.inputs[i].Update(msg)
	}

	return m, nil
}

func (m model) View() string {
	if m.done {
		return ""
	}
	var b strings.Builder

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n")
	}

	b.WriteString("\n(ctrl+c to quit, enter to submit)\n")
	b.WriteString("Tab and Shift+Tab move between fields. Press Enter on the last field to continue.\n")

	return b.String()
}

func initialModel() model {
	m := model{
		inputs: make([]textinput.Model, 3),
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = lipgloss.NewStyle()
		t.CharLimit = 128

		switch i {
		case 0:
			t.Placeholder = "Repo URL (required)"
			t.Focus()
			t.SetValue(repoURL)
		case 1:
			t.Placeholder = "Branch (main)"
			t.SetValue(repoBranch)
		case 2:
			t.Placeholder = "Submodule Directory (sec-skillz)"
			t.SetValue(repoDir)
		}

		m.inputs[i] = t
	}

	return m
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Initialize workspace with submodules and symlinks",
	Args:  cobra.NoArgs,
	Example: strings.Join([]string{
		"  skillzeug setup --repo git@github.com:org/skills.git",
		"  skillzeug setup --repo https://github.com/org/skills.git --branch main --dir sec-skillz",
	}, "\n"),
	RunE: func(cmd *cobra.Command, args []string) error {
		if interactive || repoURL == "" {
			p := tea.NewProgram(initialModel())
			m, err := p.Run()
			if err != nil {
				return err
			}

			finalModel := m.(model)
			if finalModel.done {
				if finalModel.inputs[0].Value() != "" {
					repoURL = finalModel.inputs[0].Value()
				}
				if finalModel.inputs[1].Value() != "" {
					repoBranch = finalModel.inputs[1].Value()
				}
				if finalModel.inputs[2].Value() != "" {
					repoDir = finalModel.inputs[2].Value()
				}
			} else {
				fmt.Println("Setup cancelled.")
				return nil
			}
		}

		if repoURL == "" {
			return fmt.Errorf("repo URL is required")
		}

		return runSetup()
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)

	setupCmd.Flags().StringVarP(&repoURL, "repo", "r", "", "Git repository URL for skills")
	setupCmd.Flags().StringVarP(&repoBranch, "branch", "b", "", "Git branch to use for submodule")
	setupCmd.Flags().StringVarP(&repoDir, "dir", "d", "sec-skillz", "Directory for the submodule")
	setupCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Use interactive prompts for setup")
}

func runSetupInDir(workspaceDir string) error {
	normalizedRepoDir, err := validateRepoDir(repoDir)
	if err != nil {
		return err
	}
	repoDir = normalizedRepoDir

	// 1. Git submodule add
	fmt.Printf("Adding submodule %s to %s...\n", repoURL, repoDir)
	submodulePath := filepath.Join(workspaceDir, repoDir)
	submoduleConfigured, err := isConfiguredSubmodule(workspaceDir, repoDir)
	if err != nil {
		return fmt.Errorf("failed to inspect existing submodules: %w", err)
	}

	if _, err := os.Stat(submodulePath); err == nil && submoduleConfigured {
		fmt.Printf("Submodule %s is already configured, skipping add.\n", repoDir)
	} else {
		if _, err := os.Stat(submodulePath); err == nil && !submoduleConfigured {
			return fmt.Errorf("path %s already exists and is not a configured submodule; choose a different --dir or remove the existing path", repoDir)
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to inspect submodule path %s: %w", repoDir, err)
		}

		gitArgs := []string{"submodule", "add"}
		if repoBranch != "" {
			gitArgs = append(gitArgs, "-b", repoBranch)
		}
		gitArgs = append(gitArgs, repoURL, repoDir)

		if err := runCommand(workspaceDir, "git", gitArgs...); err != nil {
			return fmt.Errorf("failed to add submodule %s at %s: %w", repoURL, repoDir, err)
		}
	}

	// 2. Git submodule update
	fmt.Printf("Refreshing submodule %s...\n", repoDir)
	if err := runCommand(workspaceDir, "git", "submodule", "update", "--remote", "--merge", "--", repoDir); err != nil {
		return fmt.Errorf("failed to update submodule: %w", err)
	}

	// 3. Create directories
	for _, dir := range assistantDirs {
		fmt.Printf("Creating directory %s...\n", dir)
		dirPath := filepath.Join(workspaceDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// 4. Create symlinks
		skillPath := filepath.Join(dirPath, "skills")
		targetPath := filepath.Join("..", repoDir, "skills")

		// Replace any existing skills link or file in the managed assistant directory.
		if _, err := os.Lstat(skillPath); err == nil {
			fmt.Printf("Replacing existing %s...\n", skillPath)
			if err := os.Remove(skillPath); err != nil {
				return fmt.Errorf("failed to remove existing symlink %s: %w", skillPath, err)
			}
		}

		fmt.Printf("Creating symlink %s -> %s\n", skillPath, targetPath)
		if err := os.Symlink(targetPath, skillPath); err != nil {
			return fmt.Errorf("failed to create symlink %s: %w", skillPath, err)
		}
	}

	fmt.Printf("Workspace setup complete in %s.\n", workspaceDir)
	fmt.Println("Run 'skillzeug show' to inspect the current configuration.")
	return nil
}

func runSetup() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine current directory: %w", err)
	}

	workspaceDir, err := gitTopLevel(cwd)
	if err != nil {
		fmt.Print("Current directory is not inside a Git repository. Run 'git init' here and continue? (y/n): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
			if err := runCommand(cwd, "git", "init"); err != nil {
				return fmt.Errorf("failed to initialize git repository: %w", err)
			}
			fmt.Println("[✓] Git repository initialized.")
			workspaceDir = cwd
		} else {
			return fmt.Errorf("git repository required for submodule setup")
		}
	}

	return runSetupInDir(workspaceDir)
}
