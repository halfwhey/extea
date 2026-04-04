package board

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/halfwhey/extea/internal/client"
	"github.com/halfwhey/extea/internal/parser"
	"github.com/urfave/cli/v3"
)

var boardFlags = []cli.Flag{
	&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "Repository (owner/repo)"},
	&cli.StringFlag{Name: "login", Aliases: []string{"l"}, Usage: "Use a different Gitea login"},
}

// CmdProjects manages project boards.
var CmdProjects = cli.Command{
	Name:     "projects",
	Aliases:  []string{"project", "p"},
	Usage:    "Manage project boards",
	Category: "BOARD",
	Commands: []*cli.Command{
		&cmdProjectsList,
		&cmdProjectsView,
		&cmdProjectsCreate,
		&cmdProjectsEdit,
		&cmdProjectsClose,
		&cmdProjectsOpen,
		&cmdProjectsDelete,
		&cmdProjectsAssign,
		&cmdProjectsUnassign,
		&cmdProjectsMove,
	},
	Flags:  boardFlags,
	Action: runProjectsList,
}

// --- list ---

var cmdProjectsList = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List projects",
	Action:  runProjectsList,
	Flags: append([]cli.Flag{
		&cli.StringFlag{Name: "state", Aliases: []string{"s"}, Value: "open", Usage: "Filter by state (open, closed, all)"},
		&cli.StringFlag{Name: "keyword", Aliases: []string{"k"}, Usage: "Search keyword"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "simple", Usage: "Output format (simple, json)"},
	}, boardFlags...),
}

func runProjectsList(_ context.Context, cmd *cli.Command) error {
	c, err := getClient(cmd)
	if err != nil {
		return err
	}
	prefix, err := getRepoPath(cmd)
	if err != nil {
		return err
	}

	state := cmd.String("state")
	keyword := cmd.String("keyword")

	var allProjects []parser.Project

	states := []string{state}
	if state == "all" {
		states = []string{"open", "closed"}
	}

	for _, s := range states {
		params := url.Values{}
		if s == "closed" {
			params.Set("state", "closed")
		}
		if keyword != "" {
			params.Set("q", keyword)
		}

		path := prefix + "/projects"
		if len(params) > 0 {
			path += "?" + params.Encode()
		}

		resp, err := c.Get(path)
		if err != nil {
			return err
		}

		projects, err := parser.ParseProjectList(resp)
		if err != nil {
			return fmt.Errorf("failed to parse projects: %w", err)
		}

		if s == "closed" {
			for i := range projects {
				projects[i].IsClosed = true
			}
		}
		allProjects = append(allProjects, projects...)
	}

	if cmd.String("output") == "json" {
		return printJSON(allProjects)
	}

	if len(allProjects) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	for _, p := range allProjects {
		line := fmt.Sprintf("#%-4d %s", p.ID, p.Title)
		if p.IsClosed {
			line += " [closed]"
		}
		if p.OpenIssues > 0 || p.ClosedIssues > 0 {
			line += fmt.Sprintf("  (%d open, %d closed)", p.OpenIssues, p.ClosedIssues)
		}
		fmt.Println(line)
	}
	return nil
}

// --- view ---

var cmdProjectsView = cli.Command{
	Name:    "view",
	Aliases: []string{"v"},
	Usage:   "View a project board",
	Action:  runProjectsView,
	Flags: append([]cli.Flag{
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "simple", Usage: "Output format (simple, json)"},
	}, boardFlags...),
}

func runProjectsView(_ context.Context, cmd *cli.Command) error {
	id := cmd.Args().First()
	if id == "" {
		return fmt.Errorf("project ID required")
	}

	c, err := getClient(cmd)
	if err != nil {
		return err
	}
	prefix, err := getRepoPath(cmd)
	if err != nil {
		return err
	}

	resp, err := c.Get(prefix + "/projects/" + id)
	if err != nil {
		return err
	}

	board, err := parser.ParseBoardState(resp)
	if err != nil {
		return fmt.Errorf("failed to parse board: %w", err)
	}

	if cmd.String("output") == "json" {
		return printJSON(board)
	}

	title := board.ProjectTitle
	if title == "" {
		title = "Project #" + id
	}
	fmt.Printf("Project: %s\n\n", title)

	if len(board.Columns) == 0 {
		fmt.Println("  (no columns)")
		return nil
	}

	for _, col := range board.Columns {
		header := fmt.Sprintf("[%s]", col.Title)
		if col.IsDefault {
			header += " (default)"
		}
		if col.Color != "" {
			header += fmt.Sprintf(" %s", col.Color)
		}
		header += fmt.Sprintf("  ID:%d  %d issues", col.ID, len(col.Issues))
		fmt.Println(header)

		if len(col.Issues) == 0 {
			fmt.Println("  (empty)")
		}
		for _, issue := range col.Issues {
			num := ""
			if issue.Number > 0 {
				num = fmt.Sprintf("#%-4d ", issue.Number)
			}
			t := issue.Title
			if t == "" {
				t = fmt.Sprintf("(internal ID: %d)", issue.InternalID)
			}
			fmt.Printf("  %s%s\n", num, t)
		}
		fmt.Println()
	}
	return nil
}

// --- create ---

var cmdProjectsCreate = cli.Command{
	Name:    "create",
	Aliases: []string{"c"},
	Usage:   "Create a new project",
	Action:  runProjectsCreate,
	Flags: append([]cli.Flag{
		&cli.StringFlag{Name: "title", Aliases: []string{"t"}, Usage: "Project title (required)"},
		&cli.StringFlag{Name: "description", Aliases: []string{"d"}, Usage: "Project description"},
		&cli.StringFlag{Name: "template", Value: "kanban", Usage: "Template: none, kanban, triage"},
		&cli.StringFlag{Name: "card-type", Value: "text", Usage: "Card type: text, images"},
	}, boardFlags...),
}

func templateType(t string) string {
	switch strings.ToLower(t) {
	case "kanban":
		return "1"
	case "triage":
		return "2"
	default:
		return "0"
	}
}

func cardType(t string) string {
	if strings.ToLower(t) == "images" {
		return "1"
	}
	return "0"
}

func runProjectsCreate(_ context.Context, cmd *cli.Command) error {
	title := cmd.String("title")
	if title == "" {
		return fmt.Errorf("--title is required")
	}

	c, err := getClient(cmd)
	if err != nil {
		return err
	}
	prefix, err := getRepoPath(cmd)
	if err != nil {
		return err
	}

	form := url.Values{
		"title":         {title},
		"content":       {cmd.String("description")},
		"template_type": {templateType(cmd.String("template"))},
		"card_type":     {cardType(cmd.String("card-type"))},
	}

	resp, err := c.PostForm(prefix+"/projects/new", form)
	if err != nil {
		return err
	}
	if err := client.CheckResponse(resp); err != nil {
		return err
	}

	fmt.Printf("Project %q created.\n", title)
	return nil
}

// --- edit ---

var cmdProjectsEdit = cli.Command{
	Name:    "edit",
	Aliases: []string{"e"},
	Usage:   "Edit a project",
	Action:  runProjectsEdit,
	Flags: append([]cli.Flag{
		&cli.StringFlag{Name: "title", Aliases: []string{"t"}, Usage: "New title"},
		&cli.StringFlag{Name: "description", Aliases: []string{"d"}, Usage: "New description"},
		&cli.StringFlag{Name: "card-type", Usage: "Card type: text, images"},
	}, boardFlags...),
}

func runProjectsEdit(_ context.Context, cmd *cli.Command) error {
	id := cmd.Args().First()
	if id == "" {
		return fmt.Errorf("project ID required")
	}

	c, err := getClient(cmd)
	if err != nil {
		return err
	}
	prefix, err := getRepoPath(cmd)
	if err != nil {
		return err
	}

	form := url.Values{}
	if t := cmd.String("title"); t != "" {
		form.Set("title", t)
	}
	if d := cmd.String("description"); d != "" {
		form.Set("content", d)
	}
	if ct := cmd.String("card-type"); ct != "" {
		form.Set("card_type", cardType(ct))
	}

	resp, err := c.PostForm(prefix+"/projects/"+id+"/edit", form)
	if err != nil {
		return err
	}
	return client.CheckResponse(resp)
}

// --- close/open/delete ---

var cmdProjectsClose = cli.Command{
	Name:   "close",
	Usage:  "Close a project",
	Flags:  boardFlags,
	Action: func(_ context.Context, cmd *cli.Command) error { return projectAction(cmd, "close", "closed") },
}

var cmdProjectsOpen = cli.Command{
	Name:   "open",
	Usage:  "Reopen a project",
	Flags:  boardFlags,
	Action: func(_ context.Context, cmd *cli.Command) error { return projectAction(cmd, "open", "reopened") },
}

var cmdProjectsDelete = cli.Command{
	Name:    "delete",
	Aliases: []string{"rm"},
	Usage:   "Delete a project",
	Flags:   boardFlags,
	Action: func(_ context.Context, cmd *cli.Command) error {
		id := cmd.Args().First()
		if id == "" {
			return fmt.Errorf("project ID required")
		}
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		prefix, err := getRepoPath(cmd)
		if err != nil {
			return err
		}
		resp, err := c.PostCSRF(prefix + "/projects/" + id + "/delete?id=" + id)
		if err != nil {
			return err
		}
		if err := client.CheckResponse(resp); err != nil {
			return err
		}
		fmt.Printf("Project #%s deleted.\n", id)
		return nil
	},
}

func projectAction(cmd *cli.Command, action, past string) error {
	id := cmd.Args().First()
	if id == "" {
		return fmt.Errorf("project ID required")
	}
	c, err := getClient(cmd)
	if err != nil {
		return err
	}
	prefix, err := getRepoPath(cmd)
	if err != nil {
		return err
	}
	resp, err := c.PostCSRF(prefix + "/projects/" + id + "/" + action)
	if err != nil {
		return err
	}
	if err := client.CheckResponse(resp); err != nil {
		return err
	}
	fmt.Printf("Project #%s %s.\n", id, past)
	return nil
}

// --- assign/unassign/move (issue-board operations) ---

var cmdProjectsAssign = cli.Command{
	Name:    "assign",
	Aliases: []string{"a"},
	Usage:   "Assign issues to a project (e.g. extea projects assign 5 --issue 1)",
	Action:  runProjectsAssign,
	Flags: append([]cli.Flag{
		&cli.IntSliceFlag{Name: "issue", Aliases: []string{"i"}, Usage: "Issue number(s) to assign (repeatable)"},
	}, boardFlags...),
}

func runProjectsAssign(_ context.Context, cmd *cli.Command) error {
	projectID := cmd.Args().First()
	if projectID == "" {
		return fmt.Errorf("project ID required")
	}
	issues := cmd.IntSlice("issue")
	if len(issues) == 0 {
		return fmt.Errorf("--issue is required")
	}

	c, err := getClient(cmd)
	if err != nil {
		return err
	}
	owner, repo, err := getRepo(cmd)
	if err != nil {
		return err
	}
	prefix := fmt.Sprintf("/%s/%s", owner, repo)

	for _, num := range issues {
		internalID, err := c.GetIssueInternalID(owner, repo, int(num))
		if err != nil {
			return fmt.Errorf("issue #%d: %w", num, err)
		}

		form := url.Values{"id": {projectID}}
		path := fmt.Sprintf("%s/issues/projects?issue_ids=%d", prefix, internalID)
		resp, err := c.DoCSRF("POST", path, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
		if err != nil {
			return fmt.Errorf("issue #%d: %w", num, err)
		}
		if err := client.CheckResponse(resp); err != nil {
			return fmt.Errorf("issue #%d: %w", num, err)
		}
		fmt.Printf("Assigned #%d to project #%s.\n", num, projectID)
	}
	return nil
}

var cmdProjectsUnassign = cli.Command{
	Name:    "unassign",
	Aliases: []string{"ua"},
	Usage:   "Remove issues from their project",
	Action:  runProjectsUnassign,
	Flags: append([]cli.Flag{
		&cli.IntSliceFlag{Name: "issue", Aliases: []string{"i"}, Usage: "Issue number(s) to unassign (repeatable)"},
	}, boardFlags...),
}

func runProjectsUnassign(_ context.Context, cmd *cli.Command) error {
	issues := cmd.IntSlice("issue")
	if len(issues) == 0 {
		return fmt.Errorf("--issue is required")
	}

	c, err := getClient(cmd)
	if err != nil {
		return err
	}
	owner, repo, err := getRepo(cmd)
	if err != nil {
		return err
	}
	prefix := fmt.Sprintf("/%s/%s", owner, repo)

	for _, num := range issues {
		internalID, err := c.GetIssueInternalID(owner, repo, int(num))
		if err != nil {
			return fmt.Errorf("issue #%d: %w", num, err)
		}

		form := url.Values{"id": {"0"}}
		path := fmt.Sprintf("%s/issues/projects?issue_ids=%d", prefix, internalID)
		resp, err := c.DoCSRF("POST", path, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
		if err != nil {
			return fmt.Errorf("issue #%d: %w", num, err)
		}
		if err := client.CheckResponse(resp); err != nil {
			return fmt.Errorf("issue #%d: %w", num, err)
		}
		fmt.Printf("Unassigned #%d from project.\n", num)
	}
	return nil
}

var cmdProjectsMove = cli.Command{
	Name:    "move",
	Aliases: []string{"m"},
	Usage:   "Move issues to a column (e.g. extea projects move 5 --column 3 --issue 1)",
	Action:  runProjectsMove,
	Flags: append([]cli.Flag{
		&cli.IntFlag{Name: "column", Aliases: []string{"c"}, Usage: "Target column ID (required)"},
		&cli.IntSliceFlag{Name: "issue", Aliases: []string{"i"}, Usage: "Issue number(s) to move (repeatable)"},
	}, boardFlags...),
}

func runProjectsMove(_ context.Context, cmd *cli.Command) error {
	projectID := cmd.Args().First()
	if projectID == "" {
		return fmt.Errorf("project ID required")
	}
	columnID := cmd.Int("column")
	if columnID == 0 {
		return fmt.Errorf("--column is required")
	}
	issues := cmd.IntSlice("issue")
	if len(issues) == 0 {
		return fmt.Errorf("--issue is required")
	}

	c, err := getClient(cmd)
	if err != nil {
		return err
	}
	owner, repo, err := getRepo(cmd)
	if err != nil {
		return err
	}
	prefix := fmt.Sprintf("/%s/%s", owner, repo)

	type issueSorting struct {
		IssueID int64 `json:"issueID"`
		Sorting int   `json:"sorting"`
	}
	payload := make([]issueSorting, len(issues))
	for i, num := range issues {
		internalID, err := c.GetIssueInternalID(owner, repo, int(num))
		if err != nil {
			return fmt.Errorf("issue #%d: %w", num, err)
		}
		payload[i] = issueSorting{IssueID: internalID, Sorting: i}
	}

	path := fmt.Sprintf("%s/projects/%s/%d/move", prefix, projectID, columnID)
	resp, err := c.PostJSON(path, map[string]any{"issues": payload})
	if err != nil {
		return err
	}
	if err := client.CheckResponse(resp); err != nil {
		return err
	}

	nums := make([]string, len(issues))
	for i, n := range issues {
		nums[i] = fmt.Sprintf("#%d", n)
	}
	fmt.Printf("Moved %s to column #%d.\n", strings.Join(nums, ", "), columnID)
	return nil
}

func printJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
