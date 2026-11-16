# cpr — Create Pull Request

A small terminal UI for opening GitHub pull requests without leaving your shell.
Under the hood it's just `git` and the [GitHub CLI](https://cli.github.com/).

## Features

- **Guided flow:** Source → Target → Title → Body → Confirm, with a progress
  breadcrumb so you always know where you are.
- **Branch picker:** lists the branches on `origin`, with `/` to fuzzy-search.
- **New target branch on the fly:** press `ctrl+n` on the target step to create
  a branch from the latest `origin/main` and push it, without checking it out.
- **Smart default title:** pre-filled from your commits (the commit subject if
  there's one commit, or the first commit if there are several).
- **Review before creating:** a summary screen where `1`–`4` jumps back to edit
  any field, and `c` creates the PR.
- **PR link on success:** prints the URL of the new pull request.
- Works from any subdirectory of a Git repository.

## Requirements

- [Go](https://go.dev/dl/), to build it
- [GitHub CLI (`gh`)](https://cli.github.com/), signed in with `gh auth login`
- `git`, inside a repository that has a GitHub `origin` remote

## Install

```sh
git clone https://github.com/pasindu-vishmika/gh-cpr
cd gh-cpr
```

**Linux / macOS**

```sh
./install.sh
```

This installs to `/usr/local/bin`. To install somewhere else without `sudo`:

```sh
INSTALL_DIR=~/.local/bin ./install.sh
```

**Windows (PowerShell)**

```powershell
.\install.ps1
```

This installs to `%LOCALAPPDATA%\Programs\cpr` and adds it to your user `PATH`.
To install somewhere else:

```powershell
$env:CPR_INSTALL_DIR = "C:\tools\cpr"; .\install.ps1
```

The installers check that Go and `gh` are installed and that `gh` is signed in,
then build the binary and put it on your `PATH`.

### Manual build

```sh
go build -o cpr .        # use -o cpr.exe on Windows
```

Then move the `cpr` binary to any folder on your `PATH`.

## Usage

From inside any Git repository:

```sh
cpr
```

| Key | Action |
| --- | --- |
| `↑` / `↓` | Move through the list |
| `↵` | Select / confirm |
| `/` | Search branches |
| `ctrl+n` | Create a new target branch from `origin/main` |
| `ctrl+j` / `ctrl+↵` | New line in the PR body |
| `1`–`4` | Edit a field on the review screen |
| `c` | Create the pull request |
| `shift+tab` | Go back a step |
| `esc` | Clear search, cancel, or quit |

## Updating

Re-run the installer. It pulls the latest source, rebuilds, and replaces the
old binary:

```sh
./install.sh        # or .\install.ps1 on Windows
```

## Built with

- [Bubble Tea](https://github.com/charmbracelet/bubbletea),
  [Bubbles](https://github.com/charmbracelet/bubbles) and
  [Lip Gloss](https://github.com/charmbracelet/lipgloss) for the terminal UI
- [GitHub CLI](https://cli.github.com/) for creating the pull request

## FAQ (Frequently Avoided Questions)

**Couldn't you just type `gh pr create`?**
Sure. And I could also memorize all six flags, their order, and which branch is
`--base` and which is `--head`. Instead I spent a few evenings writing a whole
Go program. Clearly the efficient choice.

**Why not just use the GitHub website?**
And open a browser, find the tab, click "Compare & pull request", pick the
branches from a dropdown? Absolutely exhausting. Way too many clicks.

**Why not just use the VS Code extension?**
I do, when it politely pops up after a push. But I'm a CLI person, and
reaching for the mouse feels like a betrayal. If you're the same kind of
stubborn, feel free to use this.

**Is this production-grade enterprise software?**
It has made many pull requests on my machine, so yes, it's battle-tested.

**Will you add more features?**
Only if typing them out is less effort than remembering another `gh` command.

**Contributing?**
PRs are welcome. You can even open them with `cpr`. Isn't that poetic?

---

<p align="center">Made with ❤️ and laziness</p>
