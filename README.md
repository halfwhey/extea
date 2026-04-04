# extea

A drop-in replacement for the Gitea CLI (`tea`) that adds project board (kanban) management. All standard tea commands work as-is, plus new `projects` and `columns` commands.

## Install

### Prebuilt binary (recommended)

```bash
gh release download v0.1.0 --repo halfwhey/extea --pattern '*linux*amd64*'
tar xzf extea_0.1.0_linux_amd64.tar.gz
mv extea ~/.local/bin/
```

Available platforms: `linux`, `darwin` (macOS), `windows`. Architectures: `amd64`, `arm64`.

See all assets at [Releases](https://github.com/halfwhey/extea/releases).

### From source

```bash
go install github.com/halfwhey/extea@latest
```

Or clone and build:

```bash
git clone https://github.com/halfwhey/extea.git
cd extea
CGO_ENABLED=0 go build -o extea .
```

## Authentication

```bash
extea login add  # set up API token + optionally store board password
```

During `extea login add`, after the token is created you'll be prompted:
```
Enable project board access? (stores password in plaintext in config) [y/N]:
```

If yes, the password is stored in `~/.config/tea/config.yml` alongside the login entry. You can also provide it via `GITEA_PASSWORD` env var (overrides config).

**Login resolution:** `--login` / `-l` flag > `GITEA_USERNAME` env var > tea default > sole login.
**Repo resolution:** `--repo` / `-r` flag > auto-detected from git remote in `$PWD`.

## Quick Start

```bash
extea login add                                # set up login + board password

extea projects -l mylogin -r owner/repo        # list project boards
extea projects view 5 -l mylogin -r owner/repo # view kanban board
extea issues -l mylogin -r owner/repo          # list issues (standard tea)
extea pulls -l mylogin -r owner/repo           # list PRs (standard tea)
```

## Commands

### Standard Tea Commands

All tea commands are included: `issues`, `pulls`, `repos`, `labels`, `milestones`, `releases`, `branches`, `actions`, `webhooks`, `organizations`, `times`, `notifications`, `comment`, `open`, `clone`, `whoami`, `admin`, `api`, `login`, `logout`.

See `extea <command> --help` for usage.

### Project Boards (`projects`, `project`, `p`)

| Command | Aliases | Description |
|---------|---------|-------------|
| `extea projects` | | List projects (default) |
| `extea projects list [--state open\|closed\|all] [--keyword TEXT]` | `ls` | List projects |
| `extea projects view ID` | `v` | View kanban board |
| `extea projects create --title TITLE [--template kanban\|triage\|none]` | `c` | Create project |
| `extea projects edit ID [--title TITLE] [--description TEXT]` | `e` | Edit project |
| `extea projects close ID` | | Close project |
| `extea projects open ID` | | Reopen project |
| `extea projects delete ID` | `rm` | Delete project |
| `extea projects assign ID --issue NUM [--issue NUM...]` | `a` | Assign issues to project |
| `extea projects unassign --issue NUM [--issue NUM...]` | `ua` | Remove issues from project |
| `extea projects move ID --column COL --issue NUM [--issue NUM...]` | `m` | Move issues between columns |

### Columns (`columns`, `column`, `col`)

| Command | Aliases | Description |
|---------|---------|-------------|
| `extea columns list --project ID` | `ls` | List columns |
| `extea columns create --project ID --title TITLE [--color HEX]` | | Create column |
| `extea columns edit --project ID --column ID [--title TITLE] [--color HEX]` | `e` | Edit column |
| `extea columns delete --project ID --column ID` | `rm` | Delete column |
| `extea columns default --project ID --column ID` | | Set default column |
| `extea columns move --project ID --order ID1,ID2,ID3` | | Reorder columns |

### Common Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--repo` | `-r` | Repository (owner/repo) |
| `--login` | `-l` | Login profile name |
| `--output` | `-o` | Output format: `simple`, `json` (board commands) |

## How It Works

extea wraps [tea](https://gitea.com/gitea/tea) by importing it as a Go library (`code.gitea.io/tea@v0.12.0`) and appending project board commands. Board operations use web session auth because Gitea has no REST API for project boards — API tokens return 404 on all board endpoints.

## Claude Code Skill

A Claude Code skill is included at `skills/gitea/SKILL.md` for AI-assisted Gitea interaction.
