# Skillzeug

Skillzeug is a Go CLI for setting up a shared `skills/` tree in a workspace. It manages a Git submodule and wires that submodule into assistant-specific directories such as `.codex`, `.gemini`, and `.claude`.

## What it does

- Adds a skills repository as a Git submodule.
- Refreshes only the managed submodule during initialization.
- Creates `.codex`, `.gemini`, and `.claude` in the workspace.
- Creates a `skills` symlink in each assistant directory that points at the shared submodule.
- Resolves the workspace root from anywhere inside the Git repository, so `init`, `show`, and `delete` can be run from subdirectories.

## Commands

### `init`

Initializes the workspace. If `--repo` is omitted, Skillzeug opens an interactive prompt.

```bash
skillzeug init --repo git@github.com:org/skills.git
skillzeug init --repo https://github.com/org/skills.git --branch main --dir sec-skillz
```

Notes:

- If the current directory is not inside a Git repository, Skillzeug offers to run `git init`.
- If the target submodule path already exists and is not a configured submodule, initialization stops with an error instead of silently continuing.
- Existing `skills` entries inside the managed assistant directories are replaced during initialization.

### `show`

Displays the current workspace root, managed submodule path, assistant directories, and symlink health.

```bash
skillzeug show
skillzeug show --dir sec-skillz
```

### `check`

Checks whether `git` is installed and whether `.codex`, `.gemini`, and `.claude` exist in the user's home directory.

```bash
skillzeug check
```

This command intentionally inspects the home directory, not the current repository.

### `delete`

Removes the managed submodule and deletes the workspace assistant directories created for the initialization.

```bash
skillzeug delete
skillzeug delete --force --dir sec-skillz
```

Warning:

- `delete` removes the entire `.codex`, `.gemini`, and `.claude` directories in the workspace, not only the `skills` symlink inside them.

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
