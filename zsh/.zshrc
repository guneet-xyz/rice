# | ---------------- |
# | Dependencies
# | ---------------- |
# | starship
# | exa
# | zoxide
# | pyenv
# | ---------------- |

export TERM="xterm"
 
#export HOME="/home/$USER"
eval "$(/opt/homebrew/bin/brew shellenv)"
export HOMEBREW_NO_ENV_HINTS=1

export PATH="$PATH:/usr/local/bin:/usr/bin:/bin"
export PATH="$PATH:/sbin"
export PATH="$PATH:$HOME/.local/bin"
export PATH="$PATH:$HOME/.deno/bin"
export PATH="$PATH:$HOME/.local/bin/nvim/bin"
export PATH="$PATH:/opt/nvim-linux-x86_64/bin"
export PATH="$PATH:/usr/local/go/bin"
export PATH="$PATH:$HOME/sdk/go1.23.6/bin"
export PATH="$PATH:/Users/guneet/.opencode/bin"
export PATH="$PATH:$HOME/go/bin"

export LC_CTYPE="en_IN.UTF-8"
export LC_ALL="en_IN.UTF-8"

export KUBE_EDITOR="nvim"
export EDITOR="nvim"


alias v="nvim"
alias ls="eza --icons"
alias la="ls -la"
alias gs="git status --short"
alias gl="git log --oneline"
alias gd="git diff"
alias gc="git commit"

# ----------- bun ---------------
[ -s "$HOME/.bun/_bun" ] && source "$HOME/.bun/_bun"
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
export PNPM_HOME="$HOME/.local/share/pnpm"
case ":$PATH:" in
  *":$PNPM_HOME:"*) ;;
  *) export PATH="$PNPM_HOME:$PATH" ;;
esac
# pnpm eno

# ----------- rust --------------
[ -s "$HOME/.cargo/env" ] && source "$HOME/.cargo/env"
# -------------------------------

eval "$(starship init zsh)"
eval "$(zoxide init zsh)"

# bun completions
[ -s "/Users/guneet/.bun/_bun" ] && source "/Users/guneet/.bun/_bun"

# opencode

source $(brew --prefix)/share/zsh-autosuggestions/zsh-autosuggestions.zsh
export ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE="fg=#444444"
