# TMUX

bind g display-popup -E -w 80% -h 80% ~/go/bin/twig
# prefix + X → confirm, kill current session, jump to next session
set -g detach-on-destroy off
bind X confirm-before -p "kill session #S? (y/n)" kill-session
# prefix + K → fzf pick session to kill
bind K display-popup -E "tmux list-sessions -F '#{session_id} #S' | fzf --prompt='kill: ' --with-nth=2.. | cut -d' ' -f1 | xargs -r tmux kill-session -t"
