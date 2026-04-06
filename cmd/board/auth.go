package board

import (
	"fmt"
	"strings"

	"github.com/halfwhey/extea/internal/client"
	"github.com/halfwhey/extea/internal/config"
	"github.com/halfwhey/extea/internal/git"
	"github.com/urfave/cli/v3"
)

// getClient creates an authenticated web session client for board operations.
func getClient(cmd *cli.Command) (*client.Client, error) {
	login, err := config.ResolveLogin(cmd.String("login"), "")
	if err != nil {
		return nil, err
	}

	password, err := config.Password(login.Password)
	if err != nil {
		return nil, err
	}

	token := config.Token(login.Token)

	return client.New(login.URL, login.User, password, token)
}

// getRepoPath returns /{owner}/{repo} from --repo flag or git remote.
func getRepoPath(cmd *cli.Command) (string, error) {
	owner, repo, err := getRepo(cmd)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/%s/%s", owner, repo), nil
}

// getRepo returns (owner, repo) from flag or git remote detection.
func getRepo(cmd *cli.Command) (string, string, error) {
	repo := cmd.String("repo")
	if repo != "" {
		parts := strings.SplitN(repo, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid repo format %q; expected owner/repo", repo)
		}
		return parts[0], parts[1], nil
	}

	login, err := config.ResolveLogin(cmd.String("login"), "")
	if err == nil && login.URL != "" {
		owner, name, err := git.DetectRepo(login.URL)
		if err == nil && owner != "" {
			return owner, name, nil
		}
	}

	return "", "", fmt.Errorf("no repository specified; use --repo owner/repo or run from a git repo with a Gitea remote")
}
