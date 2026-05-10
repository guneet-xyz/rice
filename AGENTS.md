# AGENTS.md

Guide for AI agents working in this repo. This is a **personal dotfiles collection** managed by [easyrice](https://github.com/guneet-xyz/easyrice). It contains only dotfile packages — no CLI source code lives here.

For CLI behavior, flags, state file format, and command reference, see the [easyrice repo](https://github.com/guneet-xyz/easyrice).

## Repo Layout

Each top-level directory is a dotfile **package** with a `rice.toml` manifest at its root:

```
rice/
├── nvim/        # Neovim
├── zsh/         # Zsh
├── ghostty/     # Ghostty terminal
├── hyprland/    # Hyprland WM
├── waybar/      # Waybar status bar
├── wofi/        # Wofi launcher
├── opencode/    # OpenCode AI assistant
├── README.md
└── AGENTS.md    # you are here
```

A package directory contains a `rice.toml` plus the dotfiles themselves, organized into source subdirectories (e.g. `common/`, `macbook/`, or a nested `.config/<tool>/` tree).

## rice.toml Schema

Every package directory has a `rice.toml` at its root.

```toml
schema_version = 1
name = "ghostty"
description = "Ghostty terminal emulator configuration"
supported_os = ["linux", "darwin"]

[profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME"}]

[profiles.macbook]
sources = [
  {path = "common", mode = "file", target = "$HOME"},
  {path = "macbook", mode = "file", target = "$HOME"},
]
```

### Fields

| Field             | Type     | Required | Notes                                                            |
|-------------------|----------|----------|------------------------------------------------------------------|
| `schema_version`  | int      | yes      | Currently `1`. Bump only on breaking schema changes.             |
| `name`            | string   | yes      | Package name. Should match the directory name.                   |
| `description`     | string   | no       | Short human-readable description.                                |
| `supported_os`    | []string | yes      | OS gate at package level. Values: `linux`, `darwin`, `windows`.  |
| `profile_key`     | string   | no       | Reserved for future per-package profile overrides.               |
| `profiles.<name>` | table    | yes      | One or more profiles. At least `common` is conventional.         |
| `profiles.<name>.sources` | []table | yes | List of source tables. Each entry requires `path` (relative subdir), `mode` (`"file"` or `"folder"`), and `target` (absolute destination root, env vars expanded). |

`sources` are relative to the package directory. Each source table specifies how files are installed.

## Source Spec

Each source entry in the `sources` list is a table with three required fields:

- **`path`**: Relative path to the source directory within the package (e.g., `"common"`, `".config/nvim"`).
- **`mode`**: Installation mode:
  - `"file"`: Walk the source directory and symlink each file individually under `target`. Files from multiple sources are overlaid.
  - `"folder"`: Symlink the entire source directory as a single unit to `target`. Cannot be overlaid by other sources in the same profile.
- **`target`**: Absolute destination root where files are installed. Supports environment variable expansion (e.g., `"$HOME"`, `"$HOME/.config"`).

### File-mode example (default, overlayable):

```toml
[profiles.common]
sources = [
  {path = "common", mode = "file", target = "$HOME"},
  {path = "macbook", mode = "file", target = "$HOME"},
]
```

This installs files from `common/` and `macbook/` into `$HOME`, with `macbook/` overlaying `common/`.

### Folder-mode example (single symlink, not overlayable):

```toml
[profiles.common]
sources = [{path = ".config/nvim", mode = "folder", target = "$HOME/.config/nvim"}]
```

This symlinks `<repo>/nvim/.config/nvim` as a single unit to `$HOME/.config/nvim`. Folder-mode sources are ideal for tools like nvim and opencode that manage their entire config directory.

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
sources = [
  {path = "common", mode = "file", target = "$HOME"},
  {path = "macbook", mode = "file", target = "$HOME"},
  {path = "work", mode = "file", target = "$HOME"},
]
```

## OS Gating

Two layers:

1. **Package level** via `supported_os`. If the current OS is not in the list, the package is skipped entirely (with a warning).
2. **Profile level** via `os` on a profile (reserved field). Currently profiles inherit from the package-level gate.

Valid OS values: `linux`, `darwin`, `windows`. Detected via Go's `runtime.GOOS` inside easyrice.

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

   [profiles.common]
   sources = [{path = "common", mode = "file", target = "$HOME"}]

   [profiles.macbook]
   sources = [
     {path = "common", mode = "file", target = "$HOME"},
     {path = "macbook", mode = "file", target = "$HOME"},
   ]
   ```

4. Test the install against a temp `$HOME` and state file:

   ```sh
   easyrice install mytool --profile common --repo . --state /tmp/rice-state.json
   easyrice status --state /tmp/rice-state.json
   easyrice uninstall mytool --state /tmp/rice-state.json -y
   ```

5. Commit:

   ```sh
   git add mytool/ && git commit -m "feat(mytool): add package"
   ```

## CLI Reference

The `easyrice` CLI lives in a separate repo. See [github.com/guneet-xyz/easyrice](https://github.com/guneet-xyz/easyrice) for:

- Command reference (`install`, `uninstall`, `switch`, `status`, `doctor`)
- Persistent flags (`--repo`, `--state`, `--log-level`, `--yes`)
- State file format and location
- Logging configuration
