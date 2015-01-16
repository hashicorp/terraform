#!/bin/bash
set -e

ZSH_FUNC_DIR="/usr/share/zsh/site-functions"

if [ -d "$ZSH_FUNC_DIR" ]; then
    echo "Installing into ${ZSH_FUNC_DIR}..."
    sudo cp ./_terraform "$ZSH_FUNC_DIR"
    echo "Installed! Make sure that ${ZSH_FUNC_DIR} is in your \$fpath."
else
    echo "Could not find ${ZSH_FUNC_DIR}. Please install manually."
    exit 1
fi
