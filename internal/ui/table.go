package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Column defines a table column.
type Column struct {
	Title string
	Width int
}

// Row is a slice of cell values.
type Row []string

// Table renders a lipgloss-styled table.
type Table struct {
	Columns []Column
	Rows    []Row
	SelIdx  int // selected row index (-1 = none)
}

// NewTable creates a new table.
func NewTable(cols []Column) *Table {
	return &Table{Columns: cols, SelIdx: -1}
}

// AddRow appends a row.
func (t *Table) AddRow(r Row) {
	t.Rows = append(t.Rows, r)
}

// Render returns the full table as a string.
// Cells are padded with lipgloss.Width() so ANSI color codes are not counted
// as visible characters.
func (t *Table) Render() string {
	var sb strings.Builder

	// pad returns s left-aligned within exactly width visible chars.
	pad := func(s string, width int) string {
		visible := lipgloss.Width(s)
		if visible >= width {
			return s
		}
		return s + strings.Repeat(" ", width-visible)
	}

	// Header row — dim gray, matching allbal column headers.
	var headers []string
	for _, col := range t.Columns {
		headers = append(headers, StyleDim.Render(pad(col.Title, col.Width)))
	}
	sb.WriteString(strings.Join(headers, "  "))
	sb.WriteString("\n")

	// Divider — box-drawing character, matching allbal separator style.
	totalW := 0
	for i, col := range t.Columns {
		totalW += col.Width
		if i < len(t.Columns)-1 {
			totalW += 2 // two-space column gap
		}
	}
	sb.WriteString(StyleMeta.Render(strings.Repeat("─", totalW)))
	sb.WriteString("\n")

	// Data rows.
	cellStyle := lipgloss.NewStyle().Foreground(ColorValue)
	for i, row := range t.Rows {
		var cells []string
		for j, col := range t.Columns {
			val := ""
			if j < len(row) {
				val = row[j]
			}
			if i == t.SelIdx {
				cells = append(cells, StyleSelected.Render(pad(val, col.Width)))
			} else {
				cells = append(cells, cellStyle.Render(pad(val, col.Width)))
			}
		}
		sb.WriteString(strings.Join(cells, "  "))
		sb.WriteString("\n")
	}

	return sb.String()
}

// KeyValueBlock renders a set of key-value pairs in a bordered box.
func KeyValueBlock(title string, pairs [][2]string) string {
	var sb strings.Builder
	if title != "" {
		sb.WriteString(StyleTitle.Render(title))
		sb.WriteString("\n")
	}
	for _, p := range pairs {
		key := StyleMeta.Render(fmt.Sprintf("%-20s", p[0]+":"))
		val := StyleValue.Render(p[1])
		sb.WriteString("  " + key + " " + val + "\n")
	}
	return StyleBorder.Render(sb.String())
}
