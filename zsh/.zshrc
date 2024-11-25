# | ---------------- |
# | Dependencies
# | ---------------- |
# | starship
# | exa
# | zoxide
# | pyenv
# | ---------------- |

export PATH="/usr/local/bin:/usr/bin:/bin"
export PATH="$PATH:/sbin"
export PATH="$HOME/.local/bin:$PATH"

export LC_CTYPE="en_IN.UTF-8"
export LC_ALL="en_IN.UTF-8"

eval "$(starship init zsh)"
eval "$(zoxide init zsh)"

alias v="nvim"

alias ls="exa"
alias la="exa -la"

alias gs="git status --short"

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

