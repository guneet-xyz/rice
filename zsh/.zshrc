# | ---------------- |
# | Dependencies
# | ---------------- |
# | starship
# | exa
# | zoxide
# | pyenv
# | ---------------- |

export TERM="xterm"
export XDG_CONFIG_HOME="$HOME/.config"
 
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

# ----------- homebrew -----------
if [ -s "/opt/homebrew" ]; then
  eval "$(/opt/homebrew/bin/brew shellenv)"
  export HOMEBREW_NO_ENV_HINTS=1
fi
# -------------------------------

# ----------- bun ---------------
if [ -s "$HOME/.bun/_bun" ]; then 
  source "$HOME/.bun/_bun"
  export BUN_INSTALL="$HOME/.bun"
  export PATH="$PATH:$BUN_INSTALL/bin"
fi
# -------------------------------

# ----------- pyenv -------------
if which pyenv >/dev/null 2>&1; then
  export PYENV_ROOT="$HOME/.pyenv"
  export PATH="$PYENV_ROOT/bin:$PATH"
  export VIRTUAL_ENV_DISABLE_PROMPT=1
  eval "$(pyenv init -)"
  eval "$(pyenv virtualenv-init -)"
fi
# -------------------------------

# ----------- nvm ---------------
if [ -s "/usr/share/nvm/init-nvm.sh" ]; then
  . /usr/share/nvm/init-nvm.sh
else
  NVM_DIR="$HOME/.nvm"
  [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"
  [ -s "$NVM_DIR/bash_completion" ] && . "$NVM_DIR/bash_completion"
fi
# -------------------------------

# ----------- rust --------------
[ -s "$HOME/.cargo/env" ] && source "$HOME/.cargo/env"
# -------------------------------

# ----------- starship ------------
if which starship >/dev/null 2>&1; then
  eval "$(starship init zsh)"
else
  echo "woah. starship is not installed. what's wrong with you?"
fi
# -------------------------------

# ----------- zoxide ------------
if which zoxide >/dev/null 2>&1; then
  eval "$(zoxide init zsh)"
else
  echo "woah. zoxide is not installed. what's wrong with you?"
fi
# -------------------------------
