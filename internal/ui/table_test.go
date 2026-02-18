package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// KeyValueBlock
// ---------------------------------------------------------------------------

func TestKeyValueBlockContainsTitleAndPairs(t *testing.T) {
	result := KeyValueBlock("My Title", [][2]string{
		{"Name", "Alice"},
		{"Balance", "1.5 ETH"},
	})
	assert.Contains(t, result, "My Title")
	assert.Contains(t, result, "Name")
	assert.Contains(t, result, "Alice")
	assert.Contains(t, result, "Balance")
	assert.Contains(t, result, "1.5 ETH")
}

func TestKeyValueBlockEmptyTitle(t *testing.T) {
	result := KeyValueBlock("", [][2]string{
		{"Key", "Value"},
	})
	assert.Contains(t, result, "Key")
	assert.Contains(t, result, "Value")
}

func TestKeyValueBlockNoPairs(t *testing.T) {
	result := KeyValueBlock("Empty Block", [][2]string{})
	assert.Contains(t, result, "Empty Block")
	assert.NotEmpty(t, result)
}

func TestKeyValueBlockSinglePair(t *testing.T) {
	result := KeyValueBlock("Single", [][2]string{
		{"OnlyKey", "OnlyVal"},
	})
	assert.Contains(t, result, "Single")
	assert.Contains(t, result, "OnlyKey")
	assert.Contains(t, result, "OnlyVal")
}

func TestKeyValueBlockMultiplePairsPreservesOrder(t *testing.T) {
	result := KeyValueBlock("Config", [][2]string{
		{"First", "AAA"},
		{"Second", "BBB"},
		{"Third", "CCC"},
	})
	idxFirst := strings.Index(result, "First")
	idxSecond := strings.Index(result, "Second")
	idxThird := strings.Index(result, "Third")
	require.Greater(t, idxFirst, -1)
	require.Greater(t, idxSecond, -1)
	require.Greater(t, idxThird, -1)
	assert.Less(t, idxFirst, idxSecond, "First should appear before Second")
	assert.Less(t, idxSecond, idxThird, "Second should appear before Third")
}

func TestKeyValueBlockHasBorder(t *testing.T) {
	result := KeyValueBlock("Bordered", [][2]string{
		{"Key", "Val"},
	})
	// lipgloss RoundedBorder uses ╭ and ╰ for corners.
	assert.Contains(t, result, "╭", "should have top-left rounded border")
	assert.Contains(t, result, "╰", "should have bottom-left rounded border")
}

// ---------------------------------------------------------------------------
// Table
// ---------------------------------------------------------------------------

func TestNewTableCreatesEmptyTable(t *testing.T) {
	cols := []Column{
		{Title: "Name", Width: 10},
		{Title: "Value", Width: 20},
	}
	tbl := NewTable(cols)
	assert.Len(t, tbl.Columns, 2)
	assert.Empty(t, tbl.Rows)
	assert.Equal(t, -1, tbl.SelIdx)
}

func TestTableAddRow(t *testing.T) {
	tbl := NewTable([]Column{{Title: "A", Width: 5}})
	tbl.AddRow(Row{"hello"})
	tbl.AddRow(Row{"world"})
	assert.Len(t, tbl.Rows, 2)
}

func TestTableRenderContainsHeaders(t *testing.T) {
	tbl := NewTable([]Column{
		{Title: "Name", Width: 10},
		{Title: "Balance", Width: 12},
	})
	result := tbl.Render()
	assert.Contains(t, result, "Name")
	assert.Contains(t, result, "Balance")
}

func TestTableRenderContainsRowData(t *testing.T) {
	tbl := NewTable([]Column{
		{Title: "Chain", Width: 10},
		{Title: "Status", Width: 10},
	})
	tbl.AddRow(Row{"ethereum", "healthy"})
	tbl.AddRow(Row{"base", "down"})

	result := tbl.Render()
	assert.Contains(t, result, "ethereum")
	assert.Contains(t, result, "healthy")
	assert.Contains(t, result, "base")
	assert.Contains(t, result, "down")
}

func TestTableRenderHasDivider(t *testing.T) {
	tbl := NewTable([]Column{{Title: "Col", Width: 8}})
	result := tbl.Render()
	assert.Contains(t, result, "--------", "should have a divider line")
}

func TestTableRenderEmptyRows(t *testing.T) {
	tbl := NewTable([]Column{
		{Title: "Header", Width: 10},
	})
	result := tbl.Render()
	assert.Contains(t, result, "Header")
	assert.NotEmpty(t, result)
}

func TestTableRenderRowShorterThanColumns(t *testing.T) {
	tbl := NewTable([]Column{
		{Title: "A", Width: 5},
		{Title: "B", Width: 5},
		{Title: "C", Width: 5},
	})
	tbl.AddRow(Row{"only1"})
	// Should not panic — missing cells render as empty.
	result := tbl.Render()
	assert.Contains(t, result, "only1")
}

func TestTableRenderPreservesRowOrder(t *testing.T) {
	tbl := NewTable([]Column{{Title: "Item", Width: 10}})
	tbl.AddRow(Row{"first"})
	tbl.AddRow(Row{"second"})
	tbl.AddRow(Row{"third"})

	result := tbl.Render()
	idxFirst := strings.Index(result, "first")
	idxSecond := strings.Index(result, "second")
	idxThird := strings.Index(result, "third")
	assert.Less(t, idxFirst, idxSecond)
	assert.Less(t, idxSecond, idxThird)
}

func TestTableRenderSelectedRow(t *testing.T) {
	tbl := NewTable([]Column{{Title: "Name", Width: 10}})
	tbl.AddRow(Row{"row0"})
	tbl.AddRow(Row{"row1"})
	tbl.SelIdx = 1

	result := tbl.Render()
	assert.Contains(t, result, "row0")
	assert.Contains(t, result, "row1")
}

func TestTableMultipleColumns(t *testing.T) {
	tbl := NewTable([]Column{
		{Title: "Hash", Width: 14},
		{Title: "From", Width: 14},
		{Title: "Value", Width: 12},
	})
	tbl.AddRow(Row{"0xabc", "0xdef", "1.5 ETH"})
	result := tbl.Render()
	assert.Contains(t, result, "Hash")
	assert.Contains(t, result, "From")
	assert.Contains(t, result, "Value")
	assert.Contains(t, result, "0xabc")
	assert.Contains(t, result, "0xdef")
	assert.Contains(t, result, "1.5 ETH")
}

// ---------------------------------------------------------------------------
// Banner
// ---------------------------------------------------------------------------

func TestBannerContainsBranding(t *testing.T) {
	result := Banner()
	assert.Contains(t, result, "Web3 Power CLI", "banner should contain product tagline")
	assert.Contains(t, result, "1.0.0", "banner should contain version")
	assert.Contains(t, result, "26 chains", "banner should mention chain count")
}

func TestBannerNonEmpty(t *testing.T) {
	assert.NotEmpty(t, Banner())
}
