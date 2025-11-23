#!/bin/sh

# common config
stow -v -t $HOME common

# platform specific config
if [[ "$OSTYPE" == "linux"* ]]; then
  stow -v -t $HOME devstick
elif [[ "$OSTYPE" == "darwin"* ]]; then
  stow -v -t $HOME macbook
else
  echo "unknown platform: $OSTYPE"
  exit 1
fi
