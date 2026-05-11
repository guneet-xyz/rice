# AGENTS.md

Guide for AI agents working in this repo. This is a **personal dotfiles collection** managed by [easyrice](https://github.com/guneet-xyz/easyrice). It contains only dotfile packages — no CLI source code lives here.

For CLI behavior, flags, state file format, and command reference, see the [easyrice repo](https://github.com/guneet-xyz/easyrice).

## Repo Layout

All packages are declared in a **single root `rice.toml`** at the repo root. Each top-level directory is a dotfile package directory containing the actual dotfiles:

```
rice/
├── rice.toml    # single root manifest — declares ALL packages
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

There are **no per-package `rice.toml` files**. The single root `rice.toml` is the only manifest easyrice reads.

## rice.toml Schema

The root `rice.toml` uses a `[packages.<name>]` map to declare all packages:

```toml
schema_version = 1

[packages.ghostty]
description = "Ghostty terminal emulator configuration"
supported_os = ["linux", "darwin"]

[packages.ghostty.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME"}]

[packages.ghostty.profiles.macbook]
sources = [
  {path = "common",  mode = "file", target = "$HOME"},
  {path = "macbook", mode = "file", target = "$HOME"},
]
```

### Fields

| Field                       | Type     | Required | Notes                                                                 |
|-----------------------------|----------|----------|-----------------------------------------------------------------------|
| `schema_version`            | int      | yes      | Must be `1`.                                                          |
| `packages.<name>`           | table    | yes      | One entry per package. Name must have no `/` or whitespace.           |
| `packages.<name>.description` | string | no       | Short human-readable description.                                     |
| `packages.<name>.supported_os` | []string | yes  | OS gate. Values: `linux`, `darwin`, `windows`.                        |
| `packages.<name>.root`      | string   | no       | Subdirectory within the package dir to use as root. Defaults to package name. No leading `/` or `..`. |
| `packages.<name>.profiles.<p>` | table | yes     | One or more profiles per package.                                     |
| `profiles.<p>.sources`      | []table  | yes      | List of source specs. Each requires `path`, `mode`, `target`.         |

**Forbidden fields** (not in schema — silently ignored but misleading):
- `name = "..."` — do NOT add this anywhere
- Top-level `profiles` tables (old schema) — not supported

### Source Spec

Each source entry requires three fields:

- **`path`**: Relative path within the package directory. No `..`, no leading `/`.
- **`mode`**: `"file"` (walk and symlink each file) or `"folder"` (symlink the entire directory).
- **`target`**: Absolute destination root. Supports env var expansion (`$HOME`, `$XDG_CONFIG_HOME`).

#### File mode (overlayable):

```toml
[packages.ghostty.profiles.macbook]
sources = [
  {path = "common",  mode = "file", target = "$HOME"},
  {path = "macbook", mode = "file", target = "$HOME"},
]
```

Files from multiple sources are overlaid (macbook overlays common).

#### Folder mode (single symlink):

```toml
[packages.nvim.profiles.default]
sources = [{path = ".config/nvim", mode = "folder", target = "$HOME/.config/nvim"}]
```

Symlinks the entire directory as one unit. Cannot be overlaid.

## Profile Conventions

| Package    | Profiles                        | Notes                                      |
|------------|---------------------------------|--------------------------------------------|
| ghostty    | `common`, `macbook`, `devstick` | file mode; macbook/devstick overlay common |
| nvim       | `default`                       | folder mode                                |
| zsh        | `common`                        | file mode; `secrets.zsh` is gitignored     |
| hyprland   | `common`                        | folder mode; Linux-only                    |
| waybar     | `common`                        | folder mode; Linux-only                    |
| wofi       | `common`                        | folder mode; Linux-only                    |
| opencode   | `personal`, `work`              | file mode; `work` is a placeholder         |

## OS Gating

`supported_os` gates the package at install time. If the current OS is not in the list, easyrice skips the package.

Valid values: `linux`, `darwin`, `windows` (detected via Go's `runtime.GOOS`).

| Package  | supported_os                    |
|----------|---------------------------------|
| ghostty  | linux, darwin                   |
| nvim     | linux, darwin                   |
| zsh      | linux, darwin                   |
| hyprland | linux                           |
| waybar   | linux                           |
| wofi     | linux                           |
| opencode | linux, darwin, windows          |

## Adding a New Dotfile Package

1. Create the package directory at the repo root and add dotfiles:
   ```sh
   mkdir -p mytool/common
   # add config files under mytool/common/
   ```

2. Add a `[packages.mytool]` block to the root `rice.toml`:
   ```toml
   [packages.mytool]
   description = "My new tool"
   supported_os = ["linux", "darwin"]

   [packages.mytool.profiles.common]
   sources = [{path = "common", mode = "file", target = "$HOME"}]
   ```

3. Test the install in a sandbox:
   ```sh
   SANDBOX=$(mktemp -d)
   HOME=$SANDBOX/home XDG_CONFIG_HOME=$SANDBOX/home/.config rice install mytool --profile common -y --state $SANDBOX/state.json
   find $SANDBOX/home -type l -ls
   rice uninstall mytool --state $SANDBOX/state.json -y
   ```

4. Commit:
   ```sh
   git add mytool/ rice.toml && git commit -m "feat(mytool): add package"
   ```

## CLI Reference

The `rice` CLI lives in a separate repo. See [github.com/guneet-xyz/easyrice](https://github.com/guneet-xyz/easyrice) for:

- Command reference (`install`, `uninstall`, `switch`, `status`, `doctor`)
- State file format and location
- Logging configuration
