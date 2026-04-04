package board

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/halfwhey/extea/internal/client"
	"github.com/halfwhey/extea/internal/parser"
	"github.com/urfave/cli/v3"
)

// CmdColumns manages project board columns.
var CmdColumns = cli.Command{
	Name:     "columns",
	Aliases:  []string{"column", "col"},
	Usage:    "Manage project board columns",
	Category: "BOARD",
	Commands: []*cli.Command{
		&cmdColumnsList,
		&cmdColumnsCreate,
		&cmdColumnsEdit,
		&cmdColumnsDelete,
		&cmdColumnsDefault,
		&cmdColumnsMove,
	},
	Flags: boardFlags,
}

// --- list ---

var cmdColumnsList = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List columns in a project",
	Action:  runColumnsList,
	Flags: append([]cli.Flag{
		&cli.IntFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project ID (required)"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "simple", Usage: "Output format (simple, json)"},
	}, boardFlags...),
}

func runColumnsList(_ context.Context, cmd *cli.Command) error {
	pid := cmd.Int("project")
	if pid == 0 {
		return fmt.Errorf("--project is required")
	}

	c, err := getClient(cmd)
	if err != nil {
		return err
	}
	prefix, err := getRepoPath(cmd)
	if err != nil {
		return err
	}

	resp, err := c.Get(fmt.Sprintf("%s/projects/%d", prefix, pid))
	if err != nil {
		return err
	}

	board, err := parser.ParseBoardState(resp)
	if err != nil {
		return err
	}

	if cmd.String("output") == "json" {
		return printJSON(board.Columns)
	}

	if len(board.Columns) == 0 {
		fmt.Println("No columns.")
		return nil
	}

	for _, col := range board.Columns {
		line := fmt.Sprintf("%-4d  %-20s", col.ID, col.Title)
		if col.IsDefault {
			line += "  (default)"
		}
		if col.Color != "" {
			line += fmt.Sprintf("  %s", col.Color)
		}
		line += fmt.Sprintf("  %d issues", len(col.Issues))
		fmt.Println(line)
	}
	return nil
}

// --- create ---

var cmdColumnsCreate = cli.Command{
	Name:   "create",
	Usage:  "Create a column in a project",
	Action: runColumnsCreate,
	Flags: append([]cli.Flag{
		&cli.IntFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project ID (required)"},
		&cli.StringFlag{Name: "title", Aliases: []string{"t"}, Usage: "Column title (required)"},
		&cli.StringFlag{Name: "color", Usage: "Hex color (e.g. #009800)"},
	}, boardFlags...),
}

func runColumnsCreate(_ context.Context, cmd *cli.Command) error {
	pid := cmd.Int("project")
	if pid == 0 {
		return fmt.Errorf("--project is required")
	}
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

	form := url.Values{"title": {title}}
	if color := cmd.String("color"); color != "" {
		form.Set("color", color)
	}

	path := fmt.Sprintf("%s/projects/%d/columns/new", prefix, pid)
	resp, err := c.DoCSRF("POST", path, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if err := client.CheckResponse(resp); err != nil {
		return err
	}

	fmt.Printf("Column %q created in project #%d.\n", title, pid)
	return nil
}

// --- edit ---

var cmdColumnsEdit = cli.Command{
	Name:    "edit",
	Aliases: []string{"e"},
	Usage:   "Edit a column (rename/recolor)",
	Action:  runColumnsEdit,
	Flags: append([]cli.Flag{
		&cli.IntFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project ID (required)"},
		&cli.IntFlag{Name: "column", Aliases: []string{"c"}, Usage: "Column ID (required)"},
		&cli.StringFlag{Name: "title", Aliases: []string{"t"}, Usage: "New title"},
		&cli.StringFlag{Name: "color", Usage: "New hex color"},
	}, boardFlags...),
}

func runColumnsEdit(_ context.Context, cmd *cli.Command) error {
	pid := cmd.Int("project")
	cid := cmd.Int("column")
	if pid == 0 || cid == 0 {
		return fmt.Errorf("--project and --column are required")
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
	if color := cmd.String("color"); color != "" {
		form.Set("color", color)
	}

	path := fmt.Sprintf("%s/projects/%d/%d", prefix, pid, cid)
	resp, err := c.PutForm(path, form)
	if err != nil {
		return err
	}
	return client.CheckResponse(resp)
}

// --- delete ---

var cmdColumnsDelete = cli.Command{
	Name:    "delete",
	Aliases: []string{"rm"},
	Usage:   "Delete a column (issues move to default)",
	Action:  runColumnsDelete,
	Flags: append([]cli.Flag{
		&cli.IntFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project ID (required)"},
		&cli.IntFlag{Name: "column", Aliases: []string{"c"}, Usage: "Column ID (required)"},
	}, boardFlags...),
}

func runColumnsDelete(_ context.Context, cmd *cli.Command) error {
	pid := cmd.Int("project")
	cid := cmd.Int("column")
	if pid == 0 || cid == 0 {
		return fmt.Errorf("--project and --column are required")
	}

	c, err := getClient(cmd)
	if err != nil {
		return err
	}
	prefix, err := getRepoPath(cmd)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s/projects/%d/%d", prefix, pid, cid)
	resp, err := c.Delete(path)
	if err != nil {
		return err
	}
	if err := client.CheckResponse(resp); err != nil {
		return err
	}

	fmt.Printf("Column #%d deleted. Issues moved to default column.\n", cid)
	return nil
}

// --- default ---

var cmdColumnsDefault = cli.Command{
	Name:   "default",
	Usage:  "Set a column as the default",
	Action: runColumnsDefault,
	Flags: append([]cli.Flag{
		&cli.IntFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project ID (required)"},
		&cli.IntFlag{Name: "column", Aliases: []string{"c"}, Usage: "Column ID (required)"},
	}, boardFlags...),
}

func runColumnsDefault(_ context.Context, cmd *cli.Command) error {
	pid := cmd.Int("project")
	cid := cmd.Int("column")
	if pid == 0 || cid == 0 {
		return fmt.Errorf("--project and --column are required")
	}

	c, err := getClient(cmd)
	if err != nil {
		return err
	}
	prefix, err := getRepoPath(cmd)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s/projects/%d/%d/default", prefix, pid, cid)
	resp, err := c.PostCSRF(path)
	if err != nil {
		return err
	}
	return client.CheckResponse(resp)
}

// --- move (reorder) ---

var cmdColumnsMove = cli.Command{
	Name:   "move",
	Usage:  "Reorder columns (e.g. --order 5,6,7)",
	Action: runColumnsMove,
	Flags: append([]cli.Flag{
		&cli.IntFlag{Name: "project", Aliases: []string{"p"}, Usage: "Project ID (required)"},
		&cli.StringFlag{Name: "order", Usage: "Column IDs in desired order, comma-separated"},
	}, boardFlags...),
}

func runColumnsMove(_ context.Context, cmd *cli.Command) error {
	pid := cmd.Int("project")
	if pid == 0 {
		return fmt.Errorf("--project is required")
	}
	order := cmd.String("order")
	if order == "" {
		return fmt.Errorf("--order is required")
	}

	c, err := getClient(cmd)
	if err != nil {
		return err
	}
	prefix, err := getRepoPath(cmd)
	if err != nil {
		return err
	}

	parts := strings.Split(order, ",")
	type colSort struct {
		ColumnID int `json:"columnID"`
		Sorting  int `json:"sorting"`
	}
	columns := make([]colSort, len(parts))
	for i, p := range parts {
		id, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return fmt.Errorf("invalid column ID %q", p)
		}
		columns[i] = colSort{ColumnID: id, Sorting: i}
	}

	path := fmt.Sprintf("%s/projects/%d/move", prefix, pid)
	resp, err := c.PostJSON(path, map[string]any{"columns": columns})
	if err != nil {
		return err
	}
	if err := client.CheckResponse(resp); err != nil {
		return err
	}

	fmt.Printf("Columns reordered in project #%d.\n", pid)
	return nil
}
