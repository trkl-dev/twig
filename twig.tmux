#!/usr/bin/env bash

TWIG_BIN="$(tmux show-option -gqv @twig_binary 2>/dev/null)"
TWIG_BIN="${TWIG_BIN:-$HOME/go/bin/twig}"

tmux bind g run-shell -b "$TWIG_BIN"
tmux bind N run-shell -b "$TWIG_BIN new"
tmux bind K display-popup -E "tmux list-sessions -F '#{session_id} #S' | fzf --prompt='kill: ' --with-nth=2.. --multi | cut -d' ' -f1 | xargs -n 1 -r tmux kill-session -t"
