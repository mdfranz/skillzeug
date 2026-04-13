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
	inputs     []textinput.Model
	focused    int
	err        error
	done       bool
	urlErr     string
}

type gitInitConfirmModel struct {
	confirmed bool
	quitting  bool
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m gitInitConfirmModel) Init() tea.Cmd {
	return nil
}

func (m gitInitConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit
		case "n", "N", "q", "ctrl+c", "esc":
			m.confirmed = false
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m gitInitConfirmModel) View() string {
	if m.quitting {
		return ""
	}
	return "Current directory is not inside a Git repository. Run 'git init' here and continue? (y/n) "
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

			// Validate URL when leaving the repo URL field (field 0)
			if (s == "tab" || s == "shift+tab") && m.focused == 0 {
				m.urlErr = ""
				if url := m.inputs[0].Value(); url != "" {
					if err := validateRepoURL(url); err != nil {
						m.urlErr = err.Error()
					}
				}
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
		if i == 0 && m.urlErr != "" {
			b.WriteString("\n  ✗ " + m.urlErr)
		}
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

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize workspace with submodules and symlinks",
	Args:  cobra.NoArgs,
	Example: strings.Join([]string{
		"  skillzeug init --repo git@github.com:org/skills.git",
		"  skillzeug init --repo https://github.com/org/skills.git --branch main --dir sec-skillz",
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
				fmt.Println("Initialization cancelled.")
				return nil
			}
		}

		if repoURL == "" {
			return fmt.Errorf("repo URL is required; use --repo or run without flags for interactive mode")
		}

		// Validate URL before proceeding
		if err := validateRepoURL(repoURL); err != nil {
			return err
		}

		return runInit()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVarP(&repoURL, "repo", "r", "", "Git repository URL for skills")
	initCmd.Flags().StringVarP(&repoBranch, "branch", "b", "", "Git branch to use for submodule")
	initCmd.Flags().StringVarP(&repoDir, "dir", "d", "sec-skillz", "Directory for the submodule")
	initCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Use interactive prompts for initialization")
	initCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without making them")
}

func runInitInDir(workspaceDir string) error {
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

		if dryRun {
			fmt.Printf("[dry-run] would run: git %s\n", strings.Join(gitArgs, " "))
		} else {
			if err := runCommand(workspaceDir, "git", gitArgs...); err != nil {
				return fmt.Errorf("failed to add submodule %s at %s: %w\nCheck the repo URL is correct and you have network access", repoURL, repoDir, err)
			}
		}
	}

	// 2. Git submodule update
	fmt.Printf("Refreshing submodule %s...\n", repoDir)
	if dryRun {
		fmt.Printf("[dry-run] would run: git submodule update --remote --merge -- %s\n", repoDir)
	} else {
		if err := runCommand(workspaceDir, "git", "submodule", "update", "--remote", "--merge", "--", repoDir); err != nil {
			return fmt.Errorf("failed to update submodule: %w", err)
		}
	}

	// 3. Create directories and symlinks
	for _, dir := range assistantDirs {
		fmt.Printf("Creating directory %s...\n", dir)
		dirPath := filepath.Join(workspaceDir, dir)
		if !dryRun {
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}

		// 4. Create symlinks
		skillPath := filepath.Join(dirPath, "skills")
		targetPath := filepath.Join("..", repoDir, "skills")

		// Replace any existing skills link or file in the managed assistant directory.
		if _, err := os.Lstat(skillPath); err == nil {
			fmt.Printf("Replacing existing %s...\n", skillPath)
			if !dryRun {
				if err := os.Remove(skillPath); err != nil {
					return fmt.Errorf("failed to remove existing symlink %s: %w", skillPath, err)
				}
			}
		}

		fmt.Printf("Creating symlink %s -> %s\n", skillPath, targetPath)
		if !dryRun {
			if err := os.Symlink(targetPath, skillPath); err != nil {
				return fmt.Errorf("failed to create symlink %s: %w", skillPath, err)
			}
		}
	}

	if dryRun {
		fmt.Println("\n[dry-run] Initialization complete. No changes made.")
	} else {
		fmt.Printf("Workspace initialization complete in %s.\n", workspaceDir)
		fmt.Println("Run 'skillzeug show' to inspect the current configuration.")
	}
	return nil
}

func runInit() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine current directory: %w", err)
	}

	workspaceDir, err := gitTopLevel(cwd)
	if err != nil {
		p := tea.NewProgram(gitInitConfirmModel{})
		m, err := p.Run()
		if err != nil {
			return err
		}
		if !m.(gitInitConfirmModel).confirmed {
			return fmt.Errorf("git repository required for submodule initialization")
		}

		if dryRun {
			fmt.Println("[dry-run] would run: git init")
		} else {
			if err := runCommand(cwd, "git", "init"); err != nil {
				return fmt.Errorf("failed to initialize git repository: %w\nCheck that 'git' is installed and you have write permission to this directory", err)
			}
			fmt.Println("[✓] Git repository initialized.")
		}
		workspaceDir = cwd
	}

	return runInitInDir(workspaceDir)
}
