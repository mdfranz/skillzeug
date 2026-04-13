# Skillzeug

Skillzeug is a Golang CLI tool designed to simplify workspace setup for security researchers and developers. It automates the process of managing git submodules and creating symlinks for specialized agent configurations.

## Features

- **`setup`**: Initializes a workspace by:
    - Checking for an existing Git repository (and offering to `git init` if missing).
    - Adding a skills repository as a git submodule.
    - Creating standard directories (`.gemini`, `.codex`, `.claude`).
    - Symlinking a `skills/` directory into each agent folder.
    - Interactive TUI prompts if required parameters (like Repo URL) are missing.
- **`check`**: Validates system requirements:
    - Verifies `git` is installed.
    - Reviews the presence of agent configurations (`.codex`, `.gemini`, `.claude`) in the user's home directory.
- **`show`**: Displays current workspace status and submodule configuration.
- **`delete`**: Safely removes the workspace setup (submodule and created directories).

## Installation

```bash
make
make install
```
This builds the binary and installs it to `~/bin/skillzeug`.

## Usage

### Workspace Setup
To set up a new workspace, run:
```bash
skillzeug setup
```
If you don't provide a repository URL via `--repo`, the tool will launch an interactive prompt.

### System Check
To verify your environment:
```bash
skillzeug check
```
