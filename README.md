# rice — personal dotfiles

Personal dotfile packages managed by [easyrice](https://github.com/guneet-xyz/easyrice).

## Packages

| Package     | Description                              | OS              |
|-------------|------------------------------------------|-----------------|
| `ghostty/`  | Ghostty terminal emulator configuration  | linux, darwin   |
| `nvim/`     | Neovim configuration (Kickstart-based)   | linux, darwin   |
| `zsh/`      | Zsh shell configuration                  | linux, darwin   |
| `hyprland/` | Hyprland wayland compositor              | linux           |
| `waybar/`   | Waybar status bar                        | linux           |
| `wofi/`     | Wofi application launcher                | linux           |
| `opencode/` | OpenCode AI assistant configuration      | linux, darwin, windows |

## Setup

1. Install [easyrice](https://github.com/guneet-xyz/easyrice).
2. Clone this repo as your easyrice dotfiles source:
   ```sh
   rice init https://github.com/guneet-xyz/rice
   ```
3. Install a package:
   ```sh
   rice install ghostty --profile macbook -y
   rice install nvim --profile default -y
   rice install zsh --profile macbook -y
   ```
4. Check status:
   ```sh
   rice status
   ```

## Profiles

| Package    | Profiles                        |
|------------|---------------------------------|
| ghostty    | `common`, `macbook`, `devstick` |
| nvim       | `default`                       |
| zsh        | `common`, `linux`, `macbook`    |
| hyprland   | `common`                        |
| waybar     | `default`, `experimental`, `line`, `zen` |
| wofi       | `common`                        |
| opencode   | `personal`, `work`              |

## Schema

All packages are declared in a single root `rice.toml` at the repo root. See [AGENTS.md](./AGENTS.md) for the schema reference and package conventions.
