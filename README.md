# rice — cross-platform dotfile manager

A single Go binary that installs, switches, and tracks dotfile packages across machines.

## What it does

`rice` replaces the old setup of GNU `stow` plus ad-hoc bash scripts (`install.sh`) with one CLI:

- Symlink-based installs into `$HOME` (or any target)
- Per-package `rice.toml` manifests with **profiles** so one repo serves many machines
- Cross-platform: `linux`, `darwin`, `windows`
- Exact, safe uninstall via a JSON state file
- Health checks for broken links and drift

If you've ever run `stow nvim` on one machine and `bash install.sh` on another, this is the consolidation.

## Quick start

```sh
# Build the binary
go build -o rice ./cmd/rice

# Install a package using the macbook profile
./rice install ghostty --profile macbook --repo .

# Check what's installed
./rice status

# Switch profiles later
./rice switch ghostty --profile devstick -y
```

## Commands

| Command     | Purpose                                          |
|-------------|--------------------------------------------------|
| `install`   | Install a package under a profile                |
| `uninstall` | Remove all links recorded in state               |
| `switch`    | Uninstall current profile, install a new one     |
| `status`    | Show installed packages, profiles, drift         |
| `doctor`    | Detect broken links and missing sources          |

Examples:

```sh
rice install nvim --profile macbook
rice uninstall nvim -y
rice switch zsh --profile work --log-level info
rice status
rice doctor
```

### Persistent flags

| Flag           | Default                | Purpose                                          |
|----------------|------------------------|--------------------------------------------------|
| `--profile`    | (required for install) | Which profile to install / switch to             |
| `--repo`       | `.`                    | Path to the rice repo                            |
| `--state`      | `~/.config/rice/state.json` | Path to state.json (Windows: `%APPDATA%/rice/`) |
| `--log-level`  | `warn`                 | `debug` / `info` / `warn` / `error` / `critical` |
| `--yes`, `-y`  | `false`                | Skip interactive confirmation prompts            |

## Dotfile package layout

Each package is a directory at the repo root with a `rice.toml` manifest and one subdirectory per **source**:

```
ghostty/
├── rice.toml
├── common/
│   └── config
├── macbook/
│   └── config
└── devstick/
    └── config
```

Example `rice.toml`:

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

Sources are composed in order. `["common", "macbook"]` lays down everything in `common/` first, then overlays `macbook/`. Later wins on conflict.

## Profiles

Standard profile names:

- `common`   shared baseline used by every machine
- `macbook`  personal MacBook overlay
- `devstick` Linux dev box / portable USB rig
- `personal` personal-only tweaks
- `work`     work-only tweaks

To add a new machine variant, add a profile that lists `common` first, then your overlay:

```toml
[profiles.workmac]
sources = ["common", "macbook", "work"]
```

Then: `rice install <pkg> --profile workmac`.

## Packages in this repo

| Package    | What it is                          |
|------------|-------------------------------------|
| `ghostty`  | Ghostty terminal emulator           |
| `nvim`     | Neovim configuration                |
| `zsh`      | Zsh shell configuration             |
| `hyprland` | Hyprland window manager (Linux)     |
| `waybar`   | Waybar status bar (Linux)           |
| `wofi`     | Wofi launcher (Linux)               |
| `opencode` | OpenCode agent configuration        |

Packages declare `supported_os` in their manifest, so Linux-only packages are skipped automatically on macOS.

## Logging

Set the log level with the `--log-level` flag or the `RICE_LOG_LEVEL` environment variable. The flag wins.

```sh
RICE_LOG_LEVEL=debug rice doctor
rice install nvim --profile macbook --log-level info
```

A persistent log file is written under the user config dir.

## More

See `AGENTS.md` for the full schema reference, state-file format, and contribution conventions.
