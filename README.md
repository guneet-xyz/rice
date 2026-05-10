# rice — personal dotfiles

Personal dotfile packages managed by [easyrice](https://github.com/guneet-xyz/easyrice).

## Packages

| Package    | Description                        |
|------------|------------------------------------|
| `nvim/`    | Neovim configuration               |
| `zsh/`     | Zsh shell configuration            |
| `ghostty/` | Ghostty terminal emulator config   |
| `hyprland/`| Hyprland window manager config     |
| `waybar/`  | Waybar status bar config           |
| `wofi/`    | Wofi launcher config               |
| `opencode/`| OpenCode AI assistant config       |

## Usage

Install the [easyrice](https://github.com/guneet-xyz/easyrice) CLI, then:

```sh
easyrice install ghostty --profile macbook --repo ~/rice
easyrice status
```

Each package has a `rice.toml` manifest describing its profiles and sources. See [easyrice docs](https://github.com/guneet-xyz/easyrice) for the full schema and CLI reference. See [AGENTS.md](./AGENTS.md) for package conventions used in this repo.
