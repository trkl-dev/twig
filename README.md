# twig

Fuzzy-switch to any git branch across all your repos — from anywhere.

`twig` scans the git repos under your project directories, lists every branch
(local and remote) with [`fzf`](https://github.com/junegunn/fzf), then checks
out the branch you pick — creating a [git worktree](https://git-scm.com/docs/git-worktree)
if it isn't already checked out — and drops you into the directory

No more `cd`-ing manually between repos, stashing changes to switch branches, or losing
your place. Every branch gets its own worktree and can be easily found.

## Features

- **One picker for every repo** — branches across all your projects in a single fzf list.
- **Worktree-per-branch** — switching branches never disturbs your working tree; each branch lives in its own directory under `~/.twig/worktrees/`.
- **Local + remote branches** — remote-only branches are checked out on demand; duplicates are hidden.
- **Create branches on the fly** — `twig new` branches off your repo's default branch (fetched first) and opens it.

## Requirements

`twig` shells out to the following, which must be on your `PATH`:

- [`git`](https://git-scm.com/)
- [`fzf`](https://github.com/junegunn/fzf)

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

### Shell Integration
- bash:
```sh
eval "$(twig init bash)"
```
- zsh (hopefully):
```sh
source <(fzf init bash)
```

## Usage

### Switch to a branch

```sh
twig
```

Discovers every git repo under your [project directories](#configuration), lists all
their branches in an fzf picker, and — when you select one — checks it out into a
worktree (if needed) and takes you to that directory.

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
5. Change directory to that new dir.

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


## License

See [`LICENSE`](LICENSE).
