# AGENTS.md

Guide for AI agents working in this repo. Covers conventions, schemas, and workflows for `rice`.

## Project Overview

`rice` is a cross-platform Go CLI dotfile manager. It replaces the previous setup of GNU `stow` plus ad-hoc bash scripts (`install.sh`) with a single binary that:

- Reads per-package `rice.toml` manifests
- Resolves the active **profile** (e.g. `macbook`, `devstick`)
- Composes file **sources** into a flat tree
- Installs files into `$HOME` (or any target) via symlinks
- Tracks every link in a JSON **state file** so uninstall is exact and safe

If you are adding a feature, prefer extending an existing `internal/` package over adding a new top-level dir.

## Repo Structure

```
rice/
├── cli/                 # CLI entrypoint and cobra commands (package main)
│   └── main.go, root.go, install.go, switch.go, uninstall.go, status.go, doctor.go, version.go
├── internal/
│   ├── manifest/        # rice.toml parsing + schema
│   ├── profile/         # profile resolution + source composition
│   ├── plan/            # planned link operations (dry-run model)
│   ├── installer/       # apply install/uninstall plans
│   ├── symlink/         # low-level symlink ops (create, verify, remove)
│   ├── state/           # state.json read/write
│   ├── logger/          # zap-backed leveled logger
│   ├── doctor/          # health checks (broken links, drift)
│   └── prompt/          # interactive yes/no confirmation
├── testdata/            # fixtures for tests (mirrors real package layouts)
├── go.mod
├── README.md
└── AGENTS.md            # you are here
```

Dotfile packages live at the repo root, one directory per package:

```
nvim/      zsh/       ghostty/   hyprland/
waybar/    wofi/      opencode/
```

Each package directory contains a `rice.toml` plus the dotfiles themselves, organized into source subdirectories (e.g. `common/`, `macbook/`).

## rice.toml Schema

Every package directory has a `rice.toml` at its root. Schema lives in `internal/manifest/schema.go`.

```toml
schema_version = 1
name = "ghostty"
description = "Ghostty terminal emulator configuration"
supported_os = ["linux", "darwin"]
target = "$HOME"

[profiles.common]
sources = ["common"]

[profiles.macbook]
sources = ["common", "macbook"]

[profiles.devstick]
sources = ["common", "devstick"]
```

### Fields

| Field             | Type                | Required | Notes                                                            |
|-------------------|---------------------|----------|------------------------------------------------------------------|
| `schema_version`  | int                 | yes      | Currently `1`. Bump only on breaking schema changes.             |
| `name`            | string              | yes      | Package name. Should match the directory name.                   |
| `description`     | string              | no       | Short human-readable description.                                |
| `supported_os`    | []string            | yes      | OS gate at package level. Values: `linux`, `darwin`, `windows`.  |
| `target`          | string              | yes      | Destination root. Usually `"$HOME"`. Env vars are expanded.      |
| `profile_key`     | string              | no       | Reserved for future per-package profile overrides.               |
| `profiles.<name>` | table               | yes      | One or more profiles. At least `common` is conventional.         |
| `profiles.<name>.sources` | []string or []table | yes      | Ordered list of sources: strings (subdirs) or tables (folder-mode). See "Folder-mode sources" below. |

`sources` are relative to the package directory. `["common", "macbook"]` means: take everything under `common/`, then overlay everything under `macbook/`.

## Folder-mode sources

For tools that require their config directory to be a single symlink (not overlaid), use folder-mode:

```toml
[profiles.common]
sources = [{path = ".config/nvim", mode = "folder", target = ".config/nvim"}]
```

This symlinks `<repo>/nvim/.config/nvim` as a single unit to `~/.config/nvim`. Folder-mode sources:
- Cannot be overlaid by other sources in the same profile
- Require both `path` (relative to package dir) and `target` (relative to `target` root)
- Are ideal for tools like nvim, opencode that manage their entire config directory

## Profile Conventions

Standard profile names (use these unless you have a strong reason not to):

- `common`   — shared baseline used by every machine
- `macbook`  — personal MacBook overlay
- `devstick` — Linux dev box / portable USB rig
- `personal` — personal-only tweaks (cross-machine)
- `work`     — work-only tweaks

Profiles compose by listing sources. To make a new machine variant, add a new profile that lists `common` first, then your overlay:

```toml
[profiles.workmac]
sources = ["common", "macbook", "work"]
```

## OS Gating

Two layers:

1. **Package level** via `supported_os`. If the current OS is not in the list, the package is skipped entirely (with a warning).
2. **Profile level** via `os` on a profile (reserved field, see schema). Currently profiles inherit from the package-level gate.

Valid OS values: `linux`, `darwin`, `windows`. Detected via Go's `runtime.GOOS`.

## State File

Location (resolved by `state.DefaultPath()`):

- POSIX (`linux`, `darwin`): `~/.config/rice/state.json`
- Windows: `%APPDATA%/rice/state.json`

Override with `--state /path/to/state.json` on any command.

Format: a JSON object keyed by package name.

```json
{
  "ghostty": {
    "profile": "macbook",
    "installed_links": [
      {
        "source": "/Users/me/code/rice/ghostty/common/config",
        "target": "/Users/me/.config/ghostty/config"
      }
    ],
    "installed_at": "2025-05-10T12:34:56Z"
  }
}
```

The state file is the source of truth for `uninstall` and `switch`. Never hand-edit it; use the CLI.

## CLI Commands

All commands accept the persistent flags below.

### Persistent flags

| Flag           | Default                | Purpose                                          |
|----------------|------------------------|--------------------------------------------------|
| `--repo`       | `.`                    | Path to the rice repo                            |
| `--state`      | `state.DefaultPath()`  | Path to state.json                               |
| `--log-level`  | `warn`                 | `debug` / `info` / `warn` / `error` / `critical` |
| `--yes`, `-y`  | `false`                | Skip interactive confirmation prompts            |

Env var: `RICE_LOG_LEVEL` sets the log level. The `--log-level` flag wins over the env var.

### Commands

```sh
rice install <package> --profile <name>   # install a package under a profile
rice uninstall <package>                  # remove all links recorded in state
rice switch <package> --profile <name>    # uninstall current profile, install new
rice status                               # show installed packages, profiles, drift
rice doctor                               # detect broken links, missing sources
```

Examples:

```sh
rice install ghostty --profile macbook --repo ~/code/rice
rice switch nvim --profile work -y
rice status --log-level info
RICE_LOG_LEVEL=debug rice doctor
```

## Logging

Five levels, in order of verbosity:

`debug` < `info` < `warn` (default) < `error` < `critical`

Set with `--log-level` or `RICE_LOG_LEVEL`. Logs are written via `internal/logger` (zap). A persistent log file lives at `logger.DefaultLogPath()`.

## Testing Conventions

- **Always run with the race detector**: `go test -race ./...`
- **Table-driven tests** are the default style. One `for _, tc := range cases` loop per behavior.
- **Fixtures** live under `testdata/`. Mirror the real package layout (`testdata/install/mypkg/rice.toml`, etc.). `testdata/` is ignored by the Go toolchain, so it's safe for arbitrary files.
- **Temp dirs**: use `t.TempDir()`. Never write into the real `$HOME` from tests.
- **State paths in tests**: pass `--state` explicitly to a temp file. Same for `--repo`.

Run a single package's tests:

```sh
go test -race ./internal/installer/...
```

## Adding a New Dotfile Package

1. Create the package directory at the repo root:

   ```sh
   mkdir -p mytool/common
   ```

2. Drop the actual config files into `common/` (and any per-machine overlay dirs you need, like `macbook/`).

3. Add `mytool/rice.toml`:

   ```toml
   schema_version = 1
   name = "mytool"
   description = "My new tool"
   supported_os = ["linux", "darwin"]
   target = "$HOME"

   [profiles.common]
   sources = ["common"]

   [profiles.macbook]
   sources = ["common", "macbook"]
   ```

4. Test the install in dry-run-ish fashion against a temp `$HOME`:

   ```sh
   rice install mytool --profile common --repo . --state /tmp/rice-state.json
   rice status --state /tmp/rice-state.json
   rice uninstall mytool --state /tmp/rice-state.json -y
   ```

5. Commit:

   ```sh
   git add mytool/ && git commit -m "feat(mytool): add package"
   ```

## Conventions Summary

- Go module: `github.com/guneet/rice`
- Go version: see `go.mod`
- All exported types live under `internal/` (the binary is the only consumer)
- Errors wrap with `fmt.Errorf("context: %w", err)`
- No `panic` outside `main`; return errors
- Symlinks are absolute, pointing back into the rice repo
