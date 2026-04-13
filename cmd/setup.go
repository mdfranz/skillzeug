package cmd

import (
	"fmt"
	"os"
	"os/exec"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if it's a git repository
		if _, err := os.Stat(".git"); os.IsNotExist(err) {
			fmt.Print("Current directory is not a git repository. Would you like to run 'git init'? (y/n): ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
				if err := runCommand("git", "init"); err != nil {
					return fmt.Errorf("failed to initialize git repository: %w", err)
				}
				fmt.Println("[✓] Git repository initialized.")
			} else {
				return fmt.Errorf("git repository required for submodule setup")
			}
		}

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

func runSetup() error {
	// 1. Git submodule add
	fmt.Printf("Adding submodule %s to %s...\n", repoURL, repoDir)
	gitArgs := []string{"submodule", "add"}
	if repoBranch != "" {
		gitArgs = append(gitArgs, "-b", repoBranch)
	}
	gitArgs = append(gitArgs, repoURL, repoDir)

	if err := runCommand("git", gitArgs...); err != nil {
		fmt.Printf("Warning: git submodule add failed (it may already exist): %v\n", err)
	}

	// 2. Git submodule update
	fmt.Println("Refreshing submodule...")
	if err := runCommand("git", "submodule", "update", "--remote", "--merge"); err != nil {
		return fmt.Errorf("failed to update submodule: %w", err)
	}

	// 3. Create directories
	dirs := []string{".gemini", ".codex", ".claude"}
	for _, dir := range dirs {
		fmt.Printf("Creating directory %s...\n", dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// 4. Create symlinks
		skillPath := filepath.Join(dir, "skills")
		targetPath := filepath.Join("..", repoDir, "skills")

		// Check if symlink already exists
		if _, err := os.Lstat(skillPath); err == nil {
			fmt.Printf("Symlink %s already exists, removing...\n", skillPath)
			if err := os.Remove(skillPath); err != nil {
				return fmt.Errorf("failed to remove existing symlink %s: %w", skillPath, err)
			}
		}

		fmt.Printf("Creating symlink %s -> %s\n", skillPath, targetPath)
		if err := os.Symlink(targetPath, skillPath); err != nil {
			return fmt.Errorf("failed to create symlink %s: %w", skillPath, err)
		}
	}

	fmt.Println("Workspace setup complete!")
	return nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
