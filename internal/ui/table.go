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
// Cells are padded with fmt.Sprintf to guarantee exact column widths — this
// avoids the lipgloss Width+PaddingRight interaction that wraps content when
// (content_length + padding) > Width.
func (t *Table) Render() string {
	var sb strings.Builder

	headerStyle := lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true)
	cellStyle := lipgloss.NewStyle().Foreground(ColorValue)
	dimStyle := lipgloss.NewStyle().Foreground(ColorMeta)

	// pad returns s left-aligned within exactly width chars, truncating if needed.
	pad := func(s string, width int) string {
		if len(s) >= width {
			return s[:width]
		}
		return s + strings.Repeat(" ", width-len(s))
	}

	// Header row.
	var headers []string
	for _, col := range t.Columns {
		headers = append(headers, headerStyle.Render(pad(col.Title, col.Width)))
	}
	sb.WriteString(strings.Join(headers, " "))
	sb.WriteString("\n")

	// Divider.
	var divParts []string
	for _, col := range t.Columns {
		divParts = append(divParts, dimStyle.Render(pad(strings.Repeat("-", col.Width), col.Width)))
	}
	sb.WriteString(strings.Join(divParts, " "))
	sb.WriteString("\n")

	// Data rows.
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
		sb.WriteString(strings.Join(cells, " "))
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
