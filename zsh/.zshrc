# | ---------------- |
# | Dependencies
# | ---------------- |
# | starship
# | exa
# | zoxide
# | pyenv
# | ---------------- |

export HOME="/home/kvqn"
export PATH="/usr/local/bin:/usr/bin:/bin"
export PATH="/sbin:$PATH"
export PATH="$HOME/.local/bin:$PATH"
export PATH="$HOME/.deno/bin:$PATH"
export PATH="$HOME/.local/bin/nvim-linux64/bin:$PATH"
export PATH="$PATH:/opt/nvim-linux-x86_64/bin"
export PATH="$PATH:/usr/local/go/bin"
export PATH="$PATH:${HOME}/sdk/go1.23.6/bin"

export LC_CTYPE="en_IN.UTF-8"
export LC_ALL="en_IN.UTF-8"

eval "$(starship init zsh)"
eval "$(zoxide init zsh)"

alias v="nvim"
alias ls="exa"
alias la="exa -la"
alias gs="git status --short"
alias gl="git log --oneline"

# ----------- bun ---------------
[ -s "/home/kvqn/.bun/_bun" ] && source "/home/kvqn/.bun/_bun"
export BUN_INSTALL="$HOME/.bun"
export PATH="$BUN_INSTALL/bin:$PATH"
# -------------------------------

# ----------- pyenv -------------
export PYENV_ROOT="$HOME/.pyenv"
export PATH="$PYENV_ROOT/bin:$PATH"
export VIRTUAL_ENV_DISABLE_PROMPT=1

eval "$(pyenv init -)"
eval "$(pyenv virtualenv-init -)"
# -------------------------------

# ----------- nvm ---------------
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"
# -------------------------------

# pnpm
export PNPM_HOME="/home/kvqn/.local/share/pnpm"
case ":$PATH:" in
  *":$PNPM_HOME:"*) ;;
  *) export PATH="$PNPM_HOME:$PATH" ;;
esac
# pnpm eno

# ----------- rust --------------
source "$HOME/.cargo/env"
# -------------------------------
