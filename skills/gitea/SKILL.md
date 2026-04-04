---
name: gitea
description: Interact with a Gitea instance using extea (tea + project boards). Manage repos, issues, PRs, labels, milestones, releases, project boards, wiki pages, and more.
version: 2.0.0
---

# Gitea

Interact with a Gitea instance using `extea` — a drop-in replacement for `tea` that adds project board (kanban) management.

## Authentication

- Use `--login <name>` (or `-l <name>`) for all commands to select a login profile.
- Board password is stored in `~/.config/tea/config.yml` alongside the login. Can also be set via `GITEA_PASSWORD` env var.
- For git push/pull over HTTP, embed the token in the URL: `http://user:TOKEN@<host>/owner/repo.git`
- Run `extea login add` to create a login (API token + optional board password).

## Repos
```bash
extea repos list -l <login>
extea repos create --name <name> --description "<desc>" -l <login>
extea repos delete --name <name> --owner <owner> --force -l <login>
```

## Issues
```bash
extea issues list -l <login>
extea issues create -l <login> --title "<title>" --description "<desc>" \
  --labels "<label1>,<label2>" --milestone "<milestone>" --assignees "<user>"
extea issues close -l <login> <number>
extea issues reopen -l <login> <number>
```

## Pull Requests
```bash
extea pulls list -l <login>
extea pulls create -l <login> --title "<title>" --description "<desc>" \
  --head <branch> --base main --labels "<labels>" --milestone "<ms>"
extea pulls merge -l <login> <number>
extea pulls close -l <login> <number>
```

## Labels
```bash
extea labels list -l <login>
extea labels create -l <login> --name "<name>" --color "#hexcolor" --description "<desc>"
extea labels delete -l <login> <id>
```

## Milestones
```bash
extea milestones list -l <login>
extea milestones create -l <login> --title "<title>" --description "<desc>" --deadline "<YYYY-MM-DD>"
extea milestones close -l <login> <name>
```

## Releases
```bash
extea releases list -l <login>
extea releases create -l <login> --tag <tag> --title "<title>" --note "<body>"
```

## Comments
```bash
extea comment -l <login> <issue-number> "<comment text>"
```

## Notifications
```bash
extea notifications -l <login>
```

## API (raw endpoint access)
```bash
extea api -l <login> <endpoint>
extea api -l <login> -X POST <endpoint> -f key=value
```

## Project Boards

Board commands use web session auth (password stored in config or `GITEA_PASSWORD` env var) because Gitea has no REST API for project boards.

### Projects (`projects`, `project`, `p`)
```bash
# List projects
extea projects -l <login> -r owner/repo
extea projects ls --state all -l <login> -r owner/repo

# View kanban board
extea projects view <id> -l <login> -r owner/repo

# Create (templates: kanban, triage, none)
extea projects create -t "<title>" --template kanban -l <login> -r owner/repo

# Edit / close / reopen / delete
extea projects edit <id> -t "<new title>" -l <login> -r owner/repo
extea projects close <id> -l <login> -r owner/repo
extea projects open <id> -l <login> -r owner/repo
extea projects delete <id> -l <login> -r owner/repo

# Assign issues to a project board
extea projects assign <project-id> -i <issue-num> -i <issue-num> -l <login> -r owner/repo

# Remove issues from a project
extea projects unassign -i <issue-num> -l <login> -r owner/repo

# Move issues between columns
extea projects move <project-id> --column <col-id> -i <issue-num> -l <login> -r owner/repo
```

### Columns (`columns`, `column`, `col`)
```bash
# List columns in a project
extea columns ls -p <project-id> -l <login> -r owner/repo

# Create a column
extea columns create -p <project-id> -t "<title>" --color "#hexcolor" -l <login> -r owner/repo

# Edit a column
extea columns edit -p <project-id> -c <col-id> -t "<new title>" -l <login> -r owner/repo

# Delete a column (issues move to default)
extea columns delete -p <project-id> -c <col-id> -l <login> -r owner/repo

# Set default column
extea columns default -p <project-id> -c <col-id> -l <login> -r owner/repo

# Reorder columns
extea columns move -p <project-id> --order <id1>,<id2>,<id3> -l <login> -r owner/repo
```

### JSON Output
Add `--output json` / `-o json` to `projects list`, `projects view`, or `columns list` for machine-readable output.

## Wiki

The wiki is a separate git repo at `http://<host>/{owner}/{repo}.wiki.git`.

**Always work with the wiki in `/tmp` to avoid polluting the current working directory.** Clone it to `/tmp/{repo}-wiki`, edit markdown files, commit, and push.

```bash
# Clone wiki to /tmp (if not already cloned)
if [ ! -d /tmp/{repo}-wiki ]; then
  git clone "http://user:${TOKEN}@<host>/{owner}/{repo}.wiki.git" /tmp/{repo}-wiki
fi

# Pages are markdown files: Home.md, Page-Name.md (use hyphens for spaces)
cd /tmp/{repo}-wiki
git add -A && git commit -m "Update wiki"
git push
```

Set `user.name` and `user.email` in the wiki repo git config before committing.

## Notes

- `extea` auto-detects the repo from the git remote in `$PWD`, but always pass `-l <login>` explicitly.
- When creating repos, do NOT use `--init` if you plan to push an existing local repo.
- Board commands require a password (stored in config via `extea login add`, or `GITEA_PASSWORD` env var). Standard tea commands use API tokens.
