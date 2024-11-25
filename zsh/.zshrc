export PATH="/usr/local/bin:/usr/bin:/bin"
export PATH="$PATH:/sbin"
export LC_CTYPE="en_IN.UTF-8"
export LC_ALL="en_IN.UTF-8"


# bun completions
[ -s "/home/kvqn/.bun/_bun" ] && source "/home/kvqn/.bun/_bun"

# bun
export BUN_INSTALL="$HOME/.bun"
export PATH="$BUN_INSTALL/bin:$PATH"
