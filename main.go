package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"code.gitea.io/tea/cmd"
	"github.com/halfwhey/extea/cmd/board"
	"github.com/halfwhey/extea/internal/config"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

var Version = "0.1.0-dev"

func main() {
	app := cmd.App()
	app.Name = "extea"
	app.Usage = "Gitea CLI with project board support"
	app.Description = appDescription
	app.Version = Version

	// Wrap login add to prompt for board password
	wrapLoginAdd(app)

	// Append board commands
	app.Commands = append(app.Commands,
		&board.CmdProjects,
		&board.CmdColumns,
	)

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// wrapLoginAdd finds the login → add subcommand and wraps its action
// to prompt for a board password after the login is created.
func wrapLoginAdd(app *cli.Command) {
	for _, c := range app.Commands {
		if c.Name != "logins" {
			continue
		}
		for _, sub := range c.Commands {
			if sub.Name != "add" {
				continue
			}
			origAction := sub.Action
			sub.Action = func(ctx context.Context, cmd *cli.Command) error {
				// Run tea's original login add
				if err := origAction(ctx, cmd); err != nil {
					return err
				}

				// Determine the login name that was just created
				loginName := cmd.String("name")
				if loginName == "" {
					// Tea auto-generates name from URL hostname; find the newest login
					logins, err := config.LoadTeaLogins()
					if err != nil || len(logins) == 0 {
						return nil
					}
					loginName = logins[len(logins)-1].Name
				}

				// Prompt for board access
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("\nEnable project board access? (stores password in plaintext in config) [y/N]: ")
				answer, _ := reader.ReadString('\n')
				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer != "y" && answer != "yes" {
					return nil
				}

				fmt.Print("Password: ")
				pw, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Println()
				if err != nil {
					return fmt.Errorf("failed to read password: %w", err)
				}

				if err := config.SetLoginPassword(loginName, string(pw)); err != nil {
					return fmt.Errorf("failed to save password: %w", err)
				}

				fmt.Printf("Board password saved for login %q.\n", loginName)
				return nil
			}
			return
		}
	}
}

var appDescription = `extea wraps tea and adds project board (kanban) management.

All standard tea commands work as expected (issues, pulls, repos, labels, etc.).
Project board commands (projects, columns) use web session auth. Passwords can be
stored in the tea config via 'extea login add' or provided via GITEA_PASSWORD.

  extea login add                       # set up login + optional board password
  extea projects -r owner/repo          # list project boards
  extea projects view 5 -r owner/repo   # view a kanban board
`
