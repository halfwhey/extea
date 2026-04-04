package git

import (
	"net/url"
	"os/exec"
	"regexp"
	"strings"
)

// DetectRepo attempts to determine owner/repo from the current directory's
// git remotes by matching against the given Gitea host URL.
// Priority: origin > upstream > first match.
func DetectRepo(giteaURL string) (owner, repo string, err error) {
	out, err := exec.Command("git", "remote", "-v").Output()
	if err != nil {
		return "", "", err
	}

	giteaHost := extractHost(giteaURL)
	if giteaHost == "" {
		return "", "", nil
	}

	type remote struct {
		name  string
		owner string
		repo  string
	}
	var remotes []remote

	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// Only process fetch URLs (avoid duplicates from push)
		if len(fields) >= 3 && fields[2] != "(fetch)" {
			continue
		}

		o, r := parseRemoteURL(fields[1], giteaHost)
		if o != "" && r != "" {
			remotes = append(remotes, remote{name: fields[0], owner: o, repo: r})
		}
	}

	if len(remotes) == 0 {
		return "", "", nil
	}

	// Priority: origin > upstream > first
	for _, preferred := range []string{"origin", "upstream"} {
		for _, r := range remotes {
			if r.name == preferred {
				return r.owner, r.repo, nil
			}
		}
	}
	return remotes[0].owner, remotes[0].repo, nil
}

// parseRemoteURL extracts owner/repo from a remote URL if it matches the given host.
func parseRemoteURL(rawURL, giteaHost string) (owner, repo string) {
	// SSH shorthand: git@host:owner/repo.git
	sshRe := regexp.MustCompile(`^[\w.-]+@([\w.-]+):(.+)/(.+?)(?:\.git)?$`)
	if m := sshRe.FindStringSubmatch(rawURL); len(m) == 4 {
		if strings.EqualFold(m[1], giteaHost) {
			return m[2], m[3]
		}
		return "", ""
	}

	// HTTP(S) or SSH URL: parse normally
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", ""
	}

	host := u.Hostname()
	if !strings.EqualFold(host, giteaHost) {
		return "", ""
	}

	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	return parts[0], parts[1]
}

func extractHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
