export LC_CTYPE="en_IN.UTF-8"
export LC_ALL="en_IN.UTF-8"

# ----------- homebrew -----------
if [ -s "/opt/homebrew" ]; then
  eval "$(/opt/homebrew/bin/brew shellenv)"
  export HOMEBREW_NO_ENV_HINTS=1
fi
# -------------------------------
