# Project Evolution

Development phases and feature evolution of `skillzeug`.

## Phase 1: Foundation (Initial Commit)
- **Initial Setup:** Project structure initialized with basic Go module and configuration.

## Phase 2: Core Workspace Functionality
- **Workspace Setup:** Implementation of the initial workspace configuration using Git submodules to share a central skills repository across multiple assistant contexts.
- **Interactive Prompts:** Integrated `Bubble Tea` for a polished TUI (Terminal User Interface) during repository selection and configuration.
- **Git Integration:** Built-in checks for existing git initialization, with the ability to trigger `git init` if a repository is not present.
- **Check Command:** Introduced the `check` command to verify system requirements (Git) and the presence of global assistant directories (`.gemini`, `.codex`, `.claude`).

## Phase 3: Project Maintenance & Standards
- **Standardization:** Updated `.gitignore` with standard Go and IDE patterns to ensure a clean development environment.
- **Architecture:** Defined a consistent directory structure where `.gemini`, `.codex`, and `.claude` directories in the workspace are symlinked to the central `skills` directory within the submodule.

## Phase 4: Refinement & UX Improvements
- **Command Hardening:** Improved robustness of workspace-related commands with better error handling for network issues and invalid repository URLs.
- **Dry-Run Mode:** Added a `--dry-run` flag to the `init` command, allowing users to preview file system and Git changes without execution.
- **CLI UX Enhancements:** Focused on providing a better user experience through `Lip Gloss` styling in the TUI and clearer feedback during the submodule addition process.

## Phase 5: Consistency & Refactoring
- **Command Renaming:** Renamed the `setup` command to `init` to better align with common CLI conventions (e.g., `git init`, `npm init`).
