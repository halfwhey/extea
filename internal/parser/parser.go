package parser

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Project struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	IsClosed     bool   `json:"is_closed"`
	OpenIssues   int    `json:"open_issues"`
	ClosedIssues int    `json:"closed_issues"`
}

type Column struct {
	ID        int     `json:"id"`
	Title     string  `json:"title"`
	Color     string  `json:"color,omitempty"`
	Sorting   int     `json:"sorting"`
	IsDefault bool    `json:"is_default"`
	Issues    []Issue `json:"issues"`
}

type Issue struct {
	InternalID int    `json:"internal_id"`
	Number     int    `json:"number"`
	Title      string `json:"title"`
}

type BoardState struct {
	ProjectTitle string   `json:"project_title"`
	Columns      []Column `json:"columns"`
}

// ParseProjectList extracts projects from the project listing page HTML.
func ParseProjectList(resp *http.Response) ([]Project, error) {
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var projects []Project
	projectIDRe := regexp.MustCompile(`/projects/(\d+)$`)

	doc.Find("a.project-board-title").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		matches := projectIDRe.FindStringSubmatch(href)
		if len(matches) < 2 {
			return
		}
		id, _ := strconv.Atoi(matches[1])
		title := strings.TrimSpace(s.Text())
		projects = append(projects, Project{
			ID:    id,
			Title: title,
		})
	})

	// Fallback: try generic link patterns if the above found nothing
	if len(projects) == 0 {
		doc.Find("a[href*='/projects/']").Each(func(_ int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			matches := projectIDRe.FindStringSubmatch(href)
			if len(matches) < 2 {
				return
			}
			id, _ := strconv.Atoi(matches[1])
			title := strings.TrimSpace(s.Text())
			if title == "" || id == 0 {
				return
			}
			// Deduplicate
			for _, p := range projects {
				if p.ID == id {
					return
				}
			}
			projects = append(projects, Project{
				ID:    id,
				Title: title,
			})
		})
	}

	// Parse issue counts from progress bars or summary text
	doc.Find(".project-board-item, .flex-item").Each(func(i int, s *goquery.Selection) {
		if i >= len(projects) {
			return
		}
		// Try to find open/closed counts in the item
		text := s.Text()
		openRe := regexp.MustCompile(`(\d+)\s+open`)
		closedRe := regexp.MustCompile(`(\d+)\s+closed`)
		if m := openRe.FindStringSubmatch(text); len(m) > 1 {
			projects[i].OpenIssues, _ = strconv.Atoi(m[1])
		}
		if m := closedRe.FindStringSubmatch(text); len(m) > 1 {
			projects[i].ClosedIssues, _ = strconv.Atoi(m[1])
		}
	})

	return projects, nil
}

// ParseBoardState extracts columns and issues from a project board page.
func ParseBoardState(resp *http.Response) (*BoardState, error) {
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	board := &BoardState{}

	// Project title from page header
	board.ProjectTitle = strings.TrimSpace(doc.Find(".project-header h2, .project-title").First().Text())
	if board.ProjectTitle == "" {
		board.ProjectTitle = strings.TrimSpace(doc.Find("h2").First().Text())
	}

	issueNumberRe := regexp.MustCompile(`/issues/(\d+)`)

	doc.Find(".project-column").Each(func(_ int, col *goquery.Selection) {
		column := Column{}

		// Column ID
		if idStr, exists := col.Attr("data-id"); exists {
			column.ID, _ = strconv.Atoi(idStr)
		}

		// Sorting order
		if sortStr, exists := col.Attr("data-sorting"); exists {
			column.Sorting, _ = strconv.Atoi(sortStr)
		}

		// Column title from edit button data attribute
		col.Find("[data-modal-project-column-title-input]").Each(func(_ int, btn *goquery.Selection) {
			if title, exists := btn.Attr("data-modal-project-column-title-input"); exists {
				column.Title = title
			}
		})
		// Fallback: try the column header text
		if column.Title == "" {
			column.Title = strings.TrimSpace(col.Find(".project-column-title-text, .column-title").First().Text())
		}

		// Column color
		col.Find("[data-modal-project-column-color-input]").Each(func(_ int, btn *goquery.Selection) {
			if color, exists := btn.Attr("data-modal-project-column-color-input"); exists {
				column.Color = color
			}
		})

		// Default column detection
		col.Find("[data-tooltip-content]").Each(func(_ int, el *goquery.Selection) {
			if tip, exists := el.Attr("data-tooltip-content"); exists {
				if strings.Contains(tip, "New issues added to this project") {
					column.IsDefault = true
				}
			}
		})

		// Issue cards
		col.Find(".issue-card").Each(func(_ int, card *goquery.Selection) {
			issue := Issue{}

			if idStr, exists := card.Attr("data-issue"); exists {
				issue.InternalID, _ = strconv.Atoi(idStr)
			}

			// Extract issue number from link
			card.Find("a[href*='/issues/']").Each(func(_ int, link *goquery.Selection) {
				href, _ := link.Attr("href")
				if matches := issueNumberRe.FindStringSubmatch(href); len(matches) > 1 {
					issue.Number, _ = strconv.Atoi(matches[1])
				}
			})

			// Extract title - try the link text or card text
			card.Find(".project-card-title, a.issue-title").Each(func(_ int, el *goquery.Selection) {
				issue.Title = strings.TrimSpace(el.Text())
			})
			if issue.Title == "" {
				// Broader fallback
				card.Find("a[href*='/issues/']").Each(func(_ int, el *goquery.Selection) {
					text := strings.TrimSpace(el.Text())
					if text != "" && text != fmt.Sprintf("#%d", issue.Number) {
						issue.Title = text
					}
				})
			}

			if issue.InternalID > 0 {
				column.Issues = append(column.Issues, issue)
			}
		})

		if column.ID > 0 {
			board.Columns = append(board.Columns, column)
		}
	})

	return board, nil
}
