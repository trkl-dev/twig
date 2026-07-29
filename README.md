# TMUX (via TPM)

Add to your tmux.conf:

```
set -g @plugin 'trkl-dev/twig'
```

Then `prefix + I` to install.

## Custom binary path

```
set -g @twig_binary '/custom/path/twig'
```

## Keybindings

| Key          | Action                                  |
| ------------ | --------------------------------------- |
| `prefix + g` | Pick branch across repos (fzf popup)   |
| `prefix + N` | Create new branch (fzf popup)          |
| `prefix + K` | Fzf pick session(s) to kill (tab=multi) |
