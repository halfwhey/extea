package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Login struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	User     string `json:"user"`
	Password string `json:"-"`
}

type teaConfig struct {
	Logins []teaLogin `yaml:"logins"`
}

type teaLogin struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	User     string `yaml:"user"`
	Default  bool   `yaml:"default"`
	Password string `yaml:"password,omitempty"`
}

// LoadTeaLogins reads logins from the tea CLI config file.
func LoadTeaLogins() ([]Login, error) {
	path, err := teaConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg teaConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse tea config: %w", err)
	}

	logins := make([]Login, len(cfg.Logins))
	for i, tl := range cfg.Logins {
		logins[i] = Login{Name: tl.Name, URL: tl.URL, User: tl.User, Password: tl.Password}
	}
	return logins, nil
}

// ResolveLogin finds the login to use based on the override name, tea config, or env vars.
func ResolveLogin(nameOverride, urlOverride string) (*Login, error) {
	path, err := teaConfigPath()
	if err == nil {
		if _, err := os.Stat(path); err == nil {
			return resolveFromTea(path, nameOverride, urlOverride)
		}
	}
	return resolveFromEnv(urlOverride)
}

func resolveFromTea(path, nameOverride, urlOverride string) (*Login, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg teaConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse tea config: %w", err)
	}

	if len(cfg.Logins) == 0 {
		return resolveFromEnv(urlOverride)
	}

	var login *teaLogin

	if nameOverride != "" {
		for i := range cfg.Logins {
			if cfg.Logins[i].Name == nameOverride {
				login = &cfg.Logins[i]
				break
			}
		}
		if login == nil {
			return nil, fmt.Errorf("tea login %q not found", nameOverride)
		}
	} else {
		if envUser := os.Getenv("GITEA_USERNAME"); envUser != "" {
			for i := range cfg.Logins {
				if cfg.Logins[i].User == envUser {
					login = &cfg.Logins[i]
					break
				}
			}
		}
		if login == nil {
			for i := range cfg.Logins {
				if cfg.Logins[i].Default {
					login = &cfg.Logins[i]
					break
				}
			}
		}
		if login == nil && len(cfg.Logins) == 1 {
			login = &cfg.Logins[0]
		}
		if login == nil {
			return nil, fmt.Errorf("multiple tea logins found and none is default; use --login to pick one, or set GITEA_USERNAME")
		}
	}

	result := &Login{Name: login.Name, URL: login.URL, User: login.User, Password: login.Password}

	if u := os.Getenv("GITEA_USERNAME"); u != "" {
		result.User = u
	}
	if urlOverride != "" {
		result.URL = urlOverride
	}

	return result, nil
}

func resolveFromEnv(urlOverride string) (*Login, error) {
	url := os.Getenv("GITEA_URL")
	if urlOverride != "" {
		url = urlOverride
	}
	user := os.Getenv("GITEA_USERNAME")

	if url == "" || user == "" {
		return nil, fmt.Errorf("no tea config found; set GITEA_URL, GITEA_USERNAME, and GITEA_PASSWORD env vars")
	}

	return &Login{Name: "env", URL: url, User: user}, nil
}

// Password returns the password to use for board auth.
// Priority: GITEA_PASSWORD env var → config password → error.
func Password(configPassword string) (string, error) {
	if pw := os.Getenv("GITEA_PASSWORD"); pw != "" {
		return pw, nil
	}
	if configPassword != "" {
		return configPassword, nil
	}
	return "", fmt.Errorf("no password configured; set GITEA_PASSWORD or run 'extea login add' with board access enabled")
}

// SetLoginPassword adds or updates the password field for a login in tea's config.
// Uses yaml.Node to preserve all existing fields and comments.
func SetLoginPassword(loginName, password string) error {
	path, err := teaConfigPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return fmt.Errorf("unexpected config format")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("unexpected config format")
	}

	// Find the "logins" key
	var loginsNode *yaml.Node
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "logins" {
			loginsNode = root.Content[i+1]
			break
		}
	}
	if loginsNode == nil || loginsNode.Kind != yaml.SequenceNode {
		return fmt.Errorf("no logins found in config")
	}

	// Find the login entry by name
	for _, entry := range loginsNode.Content {
		if entry.Kind != yaml.MappingNode {
			continue
		}

		var isTarget bool
		for i := 0; i < len(entry.Content)-1; i += 2 {
			if entry.Content[i].Value == "name" && entry.Content[i+1].Value == loginName {
				isTarget = true
				break
			}
		}
		if !isTarget {
			continue
		}

		// Check if password field already exists
		for i := 0; i < len(entry.Content)-1; i += 2 {
			if entry.Content[i].Value == "password" {
				entry.Content[i+1].Value = password
				return writeConfig(path, &doc)
			}
		}

		// Add password field
		entry.Content = append(entry.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "password"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: password},
		)
		return writeConfig(path, &doc)
	}

	return fmt.Errorf("login %q not found in config", loginName)
}

func writeConfig(path string, doc *yaml.Node) error {
	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0600)
}

func teaConfigPath() (string, error) {
	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cfgDir = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgDir, "tea", "config.yml"), nil
}
