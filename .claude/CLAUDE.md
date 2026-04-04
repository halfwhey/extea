# extea

Drop-in replacement for `tea` (Gitea CLI) that adds project board (kanban) management. Wraps tea as a Go library dependency — all tea commands work, plus `projects` and `columns`.

## Architecture

- **Framework:** urfave/cli v3 (same as tea)
- **Tea integration:** imports `code.gitea.io/tea/cmd` and appends board commands to tea's app
- **Board auth:** Web session login via `GITEA_PASSWORD` env var (Gitea has no API for boards; API tokens return 404)
- **Board parsing:** goquery HTML parsing for board state

### Package Structure

- `main.go` — creates tea app via `cmd.App()`, appends board commands
- `cmd/board/` — project board commands (projects, columns) using urfave/cli v3
- `internal/config/` — reads tea's `~/.config/tea/config.yml` for login metadata
- `internal/client/` — HTTP client with web login, CSRF handling
- `internal/parser/` — HTML parsing for project lists and board state
- `internal/git/` — git remote detection for auto repo resolution

## Development

- Build: `CGO_ENABLED=0 go build -o extea .`
- Test: `GITEA_PASSWORD='...' ./extea projects -r claude/extea-test`
