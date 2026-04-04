package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	CSRFToken  string
}

// New creates an authenticated client by performing a fresh login.
// Gitea rotates session secrets on each request, so we login fresh
// each invocation rather than persisting session cookies.
func New(baseURL, username, password string) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("no Gitea URL; configure tea or set GITEA_URL")
	}
	if username == "" || password == "" {
		return nil, fmt.Errorf("missing credentials; set GITEA_USERNAME and GITEA_PASSWORD")
	}
	return Login(baseURL, username, password)
}

// Login authenticates against Gitea and returns a ready-to-use client.
func Login(baseURL, username, password string) (*Client, error) {
	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{Jar: jar}

	// GET login page to seed CSRF cookie
	resp, err := httpClient.Get(baseURL + "/user/login")
	if err != nil {
		return nil, fmt.Errorf("failed to reach login page: %w", err)
	}
	resp.Body.Close()

	u, _ := url.Parse(baseURL)
	var csrfToken string
	for _, c := range jar.Cookies(u) {
		if c.Name == "_csrf" {
			csrfToken = c.Value
			break
		}
	}
	if csrfToken == "" {
		return nil, fmt.Errorf("no CSRF token found on login page")
	}

	// POST login credentials
	form := url.Values{
		"_csrf":     {csrfToken},
		"user_name": {username},
		"password":  {password},
		"remember":  {"on"},
	}
	resp, err = httpClient.PostForm(baseURL+"/user/login", form)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	resp.Body.Close()

	// Verify we got a session
	var hasSession bool
	for _, c := range jar.Cookies(u) {
		if c.Name == "gitea_incredible" && c.Value != "" {
			hasSession = true
		}
		if c.Name == "_csrf" {
			csrfToken = c.Value
		}
	}
	if !hasSession {
		return nil, fmt.Errorf("login failed (check credentials)")
	}

	// The client follows redirects and keeps the cookie jar alive,
	// so session rotation is handled transparently within one invocation.
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: httpClient,
		CSRFToken:  csrfToken,
	}, nil
}

// refreshCSRF fetches a page and updates the CSRF token from the response cookies.
func (c *Client) refreshCSRF() error {
	u, _ := url.Parse(c.BaseURL)
	for _, cookie := range c.HTTPClient.Jar.Cookies(u) {
		if cookie.Name == "_csrf" {
			c.CSRFToken = cookie.Value
			return nil
		}
	}
	return nil
}

// Get performs a GET request and returns the response.
func (c *Client) Get(path string) (*http.Response, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + path)
	if err != nil {
		return nil, err
	}
	c.refreshCSRF()
	return resp, nil
}

// PostForm sends a form-encoded POST with CSRF token in the body.
func (c *Client) PostForm(path string, data url.Values) (*http.Response, error) {
	data.Set("_csrf", c.CSRFToken)
	resp, err := c.HTTPClient.PostForm(c.BaseURL+path, data)
	if err != nil {
		return nil, err
	}
	c.refreshCSRF()
	return resp, nil
}

// DoCSRF performs a request with the CSRF header.
func (c *Client) DoCSRF(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Csrf-Token", c.CSRFToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	c.refreshCSRF()
	return resp, nil
}

// PostJSON sends a JSON POST with CSRF header.
func (c *Client) PostJSON(path string, payload any) (*http.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return c.DoCSRF("POST", path, strings.NewReader(string(data)), "application/json")
}

// PostCSRF sends a POST with just the CSRF header (no body).
func (c *Client) PostCSRF(path string) (*http.Response, error) {
	return c.DoCSRF("POST", path, nil, "")
}

// PutForm sends a form-encoded PUT with CSRF header.
func (c *Client) PutForm(path string, data url.Values) (*http.Response, error) {
	return c.DoCSRF("PUT", path, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
}

// Delete sends a DELETE with CSRF header.
func (c *Client) Delete(path string) (*http.Response, error) {
	return c.DoCSRF("DELETE", path, nil, "")
}

// GetIssueInternalID fetches the internal database ID for an issue number via the API.
func (c *Client) GetIssueInternalID(owner, repo string, issueNumber int) (int64, error) {
	resp, err := c.Get(fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d", owner, repo, issueNumber))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("failed to get issue #%d: HTTP %d", issueNumber, resp.StatusCode)
	}

	var result struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return result.ID, nil
}

// CheckResponse reads a response and checks for success.
func CheckResponse(resp *http.Response) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		body, err := io.ReadAll(resp.Body)
		if err != nil || len(body) == 0 {
			return nil
		}
		var result map[string]any
		if err := json.Unmarshal(body, &result); err != nil {
			return nil // HTML response, that's fine
		}
		if errMsg, ok := result["errorMessage"]; ok {
			return fmt.Errorf("%v", errMsg)
		}
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
