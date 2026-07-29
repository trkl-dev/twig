# Twig

CLI tool that scans git repos under `$HOME/Projects` (and `$HOME/dotfiles`), lists branches via `fzf`, then checks out the selected branch (creating a git worktree if needed) and opens a tmux session for it.

## Build/Run

```
go build ./...
go run .          # runs the fzf picker
```

## Architecture

Multi-file cobra commands in `cmd/*.go`. Entry point: `main.go`.

All cobra commands live as separate files in `cmd/` (e.g. `root.go`, `new.go`). Each registers itself via `rootCmd.AddCommand` in its own `init()`. Always check `cmd/` for all available commands before assuming only one exists.

Entry points into `main.go`: sets up `TWIG_DEBUG` env var for `slog` debug logging, then calls `cmd.Execute()`.

### Data flow

```
WalkDir finds .git dirs → concurrent `git branch -a` with worktree detection →
filter/dedup → fzf → ensureWorktree (checkout or create) → openTmux (create/attach)
```

### Runtime details

- **Walk**: `fs.WalkDir` from each dir in `ProjectDirs`, capped at `MaxDepth = 5` levels.
- **Branch listing**: 4 tab-separated fzf fields: `repoName\tfullBranch\tdisplayName\trepoPath`. fzf shows fields 1 and 3 via `--with-nth=1,3`.
- **Concurrency**: branch listings across repos run in parallel via `sync.WaitGroup`.
- **Performance**: `runCmdIn` wraps all `exec.Command` calls, tracking cumulative git time and count via `atomic.Int64`.
- **Worktree**: if the branch isn't checked out, a worktree is created at `~/.twig/worktrees/<repo>/<branch-sanitized>`. Remote branches get `-b <local> <remote-ref>`.
- **Tmux**: creates a detached session named `<repo> » <branch>` (with `.` and `:` replaced by `-`), then switches to it if already inside tmux, or attaches.

## Key decisions

- **`-a` flag**: lists both local and remote branches (from `refs/heads/` and `refs/remotes/`)
- **Remote HEAD filtering**: `refs/remotes/origin/HEAD` shortens to `origin` with `%(refname:short)` — filtered out since it's not a branch
- **Dedup**: when a local branch exists, the matching remote branch (`origin/<name>`) is hidden. Remote-only branches show with `(r)` suffix.
- **Worktree detection**: the format string `%(refname:short)%09%(if)%(worktreepath)%(then)wt%(end)` marks branches with an active worktree as `(wt)` for local, or `(r), wt` for remote-only.
- **Display suffixes**: `(wt)` = worktree is checked out, `(r)` = remote-only branch, `(r), wt` = remote-only with a worktree.

## Git format quirks

- `git branch -a --format=%(refname:short)` shortens `refs/remotes/origin/HEAD` to `origin` (the remote name alone). Filtered via `b == "origin"`.
- `refs/remotes/origin/master` → `origin/master`, `refs/heads/main` → `main`

## Tmux plugin

`twig.tmux` binds:
- `g` → fzf branch picker via `--popup center,80%`
- `N` → fzf new branch creator via `--popup center,60%`
- `K` → kill sessions via fzf multi-select

Set `@twig_binary` in tmux to override the default `$HOME/go/bin/twig`.
