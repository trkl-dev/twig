# twig

Fuzzy-switch to any git branch across all your repos — from anywhere.

`twig` scans the git repos under your project directories, lists every branch
(local and remote) with [`fzf`](https://github.com/junegunn/fzf), then checks
out the branch you pick — creating a [git worktree](https://git-scm.com/docs/git-worktree)
if it isn't already checked out — and drops you into a dedicated
[`tmux`](https://github.com/tmux/tmux) session for it.

No more `cd`-ing between repos, stashing changes to switch branches, or losing
your place. Every branch gets its own worktree and its own tmux session.

## Features

- **One picker for every repo** — branches across all your projects in a single fzf list.
- **Worktree-per-branch** — switching branches never disturbs your working tree; each branch lives in its own directory under `~/.twig/worktrees/`.
- **Session-per-branch** — each branch opens in a named tmux session, so your layout and running processes persist.
- **Local + remote branches** — remote-only branches are checked out on demand; duplicates are hidden.
- **Create branches on the fly** — `twig new` branches off your repo's default branch (fetched first) and opens it.
- **tmux plugin** — bind the picker to a key and drive everything without leaving tmux.

## Requirements

`twig` shells out to the following, which must be on your `PATH`:

- [`git`](https://git-scm.com/)
- [`fzf`](https://github.com/junegunn/fzf)
- [`tmux`](https://github.com/tmux/tmux)

Building from source requires **Go 1.26+**.

## Install

```sh
go install github.com/trkl-dev/twig@latest
```

This installs the `twig` binary to `$(go env GOPATH)/bin` (typically
`~/go/bin`). Make sure that directory is on your `PATH`:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

Or build from a checkout:

```sh
git clone https://github.com/trkl-dev/twig.git
cd twig
go build ./...        # or: go install .
```

## Usage

### Switch to a branch

```sh
twig
```

Discovers every repo under your [project directories](#configuration), lists all
their branches in an fzf picker, and — when you select one — checks it out into a
worktree (if needed) and opens a tmux session named `<repo> » <branch>`.

The picker shows two columns:

```
REPO      BRANCH (r = remote, wt = worktree)
```

Branch display suffixes:

| Suffix     | Meaning                                    |
| ---------- | ------------------------------------------ |
| `(wt)`     | branch already checked out in a worktree   |
| `(r)`      | remote-only branch (no local counterpart)  |
| `(r), wt`  | remote-only branch that has a worktree      |

When a local branch and its matching `origin/<name>` both exist, the remote entry
is hidden.

### Create a new branch

```sh
twig new
```

Pick a repo, type a name for the new branch, and `twig`:

1. Determines the repo's default base branch (`origin/HEAD`, falling back to
   `origin/main`, `origin/master`, the current branch, then `HEAD`).
2. Fetches that base concurrently while you type.
3. Validates the branch name with `git check-ref-format`.
4. Creates the branch off the base in a new worktree — or switches to it if a
   branch by that name already exists locally or on `origin`.
5. Opens a tmux session for it.

## Configuration

### Project directories

By default `twig` scans these directories for repos (walking up to 5 levels deep):

- `$HOME/Projects`
- `$HOME/dotfiles`

These are defined in `cmd/root.go` (`ProjectDirs` and `MaxDepth`). Adjust them to
match your layout and rebuild.

### Worktree layout

Worktrees are created under:

```
~/.twig/worktrees/<repo>/<branch>
```

Slashes in branch names are replaced with dashes (e.g. `feat/login` →
`feat-login`).

### Environment variables

| Variable     | Effect                                                                 |
| ------------ | ---------------------------------------------------------------------- |
| `TWIG_DEBUG` | Set to any non-empty value to enable `slog` debug logging (timings, discovery, and a performance summary). |
| `TMUX`       | Detected automatically. Inside tmux, `twig` uses `switch-client` and fzf's popup; outside, it attaches and draws a bordered fzf. |

## tmux plugin

`twig` ships as a [TPM](https://github.com/tmux-plugins/tpm)-compatible tmux
plugin so you can launch the pickers from a keybinding.

Add to your `tmux.conf`:

```tmux
set -g @plugin 'trkl-dev/twig'
```

Then press `prefix + I` to install.

### Keybindings

| Key          | Action                                   |
| ------------ | ---------------------------------------- |
| `prefix + g` | Pick a branch across repos (fzf popup)   |
| `prefix + N` | Create a new branch (fzf popup)          |
| `prefix + K` | Pick session(s) to kill (tab = multi)    |

### Custom binary path

The plugin looks for the binary at `$HOME/go/bin/twig`. Override it with:

```tmux
set -g @twig_binary '/custom/path/twig'
```

## How it works

```
WalkDir finds .git dirs → concurrent `git branch -a` (with worktree detection)
  → filter/dedup → fzf → ensureWorktree (checkout or create) → openTmux (create/attach)
```

- **Discovery** walks each project directory with `fs.WalkDir`, treating any
  directory containing a `.git` directory as a repo (worktree `.git` *files* are
  skipped so worktrees aren't listed as separate repos).
- **Branch listing** runs `git branch -a` per repo in parallel, using a git
  format string to detect which branches have active worktrees.
- **Worktree handling** reuses an existing worktree if the branch is already
  checked out; otherwise it creates one (remote branches get
  `-b <local> <remote-ref>`).
- **tmux** creates a detached session (name-sanitized so `.` and `:` become `-`),
  then switches to it if you're inside tmux or attaches if you're not.

See [`AGENTS.md`](AGENTS.md) for a deeper architecture reference.

## License

See [`LICENSE`](LICENSE).
