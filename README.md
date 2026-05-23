# Skillzeug

Skillzeug is a Go CLI for setting up a shared `skills/` tree in a workspace. It manages a Git submodule and wires that submodule into assistant-specific directories such as `.codex`, `.gemini`, `.claude`, and `.agents`.

## What it does

- Adds a skills repository as a Git submodule.
- Refreshes only the managed submodule during initialization.
- Creates `.codex`, `.gemini`, `.claude`, and `.agents` in the workspace.
- Creates a `skills` symlink in each assistant directory that points at the shared submodule.
- Resolves the workspace root from anywhere inside the Git repository, so `init`, `show`, and `delete` can be run from subdirectories.

## Commands

### `init`

Initializes the workspace. If `--repo` is omitted, Skillzeug opens an interactive prompt. In non-interactive environments (e.g., CI/CD or scripts where standard input is not a TTY), `--repo` is required.

```bash
skillzeug init --repo git@github.com:org/skills.git
skillzeug init --repo https://github.com/org/skills.git --branch main --dir sec-skillz
```

Notes:

- If the current directory is not inside a Git repository, Skillzeug offers to run `git init` (in interactive mode). In non-interactive environments, this check fails with an error requiring manual repository setup.
- If the target submodule path already exists and is not a configured submodule, initialization stops with an error instead of silently continuing.
- Existing `skills` entries inside the managed assistant directories are replaced during initialization.

### `show`

Displays the current workspace root, managed submodule path, assistant directories, and symlink health.

```bash
skillzeug show
skillzeug show --dir sec-skillz
```

### `update`

Updates an existing workspace initialization. It can refresh the submodule to the latest remote state, or change the repository URL/branch.

```bash
skillzeug update
skillzeug update --branch main
skillzeug update --repo git@github.com:org/new-skills.git
skillzeug update --repo https://github.com/org/skills.git --dry-run
```

Notes:

- If `--repo` is provided, the existing submodule is removed and replaced with the new one.
- Refreshes the `skills` symlinks in all assistant directories (`.codex`, `.gemini`, `.claude`, and `.agents`).

### `check`

Checks whether `git` is installed and whether `.codex`, `.gemini`, `.claude`, and `.agents` exist in the user's home directory.

```bash
skillzeug check
```

This command intentionally inspects the home directory, not the current repository.

### `delete`

Removes the managed submodule and deletes the workspace assistant directories created for the initialization.

```bash
skillzeug delete
skillzeug delete --force --dir sec-skillz
skillzeug delete --prune-dirs
```

Notes:

- In non-interactive environments, the confirmation prompt is skipped, and you must pass the `--force` flag; otherwise, the command will exit with an error.
- By default, `delete` only removes the `skills` symlinks inside the assistant directories.
- Use `--prune-dirs` to remove the entire `.codex`, `.gemini`, `.claude`, and `.agents` directories.

## Installation

```bash
make build
make install
```

This builds the binary and installs it to `~/bin/skillzeug`.

## Development

```bash
make test
make build
```

The test suite covers path validation, initialization failure handling, scoped submodule updates, and workspace-root resolution.
