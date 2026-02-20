package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ── Types ────────────────────────────────────────────────────────────────────

// StudioParam describes one ABI input parameter with human-friendly hints.
type StudioParam struct {
	Name    string
	Type    string
	Example string // pre-filled hint based on ABI type, e.g. "0xAbCd… (42 chars)"

	// IsTokenAmount marks a uint256 param that represents a token quantity.
	// When true, the user enters a human-readable amount (e.g. "1.5") and
	// collectStudioInputs scales it by 10^Decimals before passing to the ABI encoder.
	IsTokenAmount bool
	Decimals      int // token decimals, used when IsTokenAmount is true
}

// StudioEntry is one navigable item in the contract studio.
type StudioEntry struct {
	Name        string
	Selector    string       // "0xa9059cbb" — empty for events
	Sig         string       // canonical sig, e.g. "transfer(address,uint256)"
	IsWrite     bool
	IsEvent     bool
	Inputs      []StudioParam
	OutputTypes []string // display only (read functions)
	Description string   // human description, shown in the info panel
}

// ── Bubble Tea model ─────────────────────────────────────────────────────────

// StudioModel is the Bubble Tea model for the interactive function navigator.
// It shows read functions, write functions and events in labelled sections,
// lets the user navigate with ↑↓ / j k, and exits with the selected entry
// when Enter is pressed.
type StudioModel struct {
	// Static contract metadata
	ContractName string
	Address      string
	Network      string
	Mode         string // "mainnet" | "testnet"
	Kind         string // e.g. "builtin:w3token"
	FuncCount    int
	EventCount   int

	Entries []StudioEntry

	// internal navigation
	navItems []int // navItems[i] = index into Entries; events are excluded
	cursor   int   // position inside navItems

	// output
	Selected *StudioEntry
	Quitting bool
}

func (m *StudioModel) buildNav() {
	m.navItems = nil
	// order: reads first, then writes; events are displayed but not navigable
	var reads, writes []int
	for i, e := range m.Entries {
		if e.IsEvent {
			continue
		}
		if e.IsWrite {
			writes = append(writes, i)
		} else {
			reads = append(reads, i)
		}
	}
	m.navItems = append(m.navItems, reads...)
	m.navItems = append(m.navItems, writes...)
}

func (m StudioModel) Init() tea.Cmd { return nil }

func (m StudioModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.Quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.navItems)-1 {
				m.cursor++
			}
		case "enter", " ":
			if len(m.navItems) > 0 {
				e := m.Entries[m.navItems[m.cursor]]
				m.Selected = &e
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m StudioModel) View() string {
	if m.Quitting {
		return ""
	}

	var sb strings.Builder

	const sepWidth = 72

	// ── Title ─────────────────────────────────────────────────────────────
	title := fmt.Sprintf("  Contract Studio  ·  %s  ·  %s (%s)",
		m.ContractName, m.Network, m.Mode)
	sb.WriteString(StyleTitle.Render(title) + "\n\n")

	// ── Metadata ──────────────────────────────────────────────────────────
	sb.WriteString(fmt.Sprintf("  %-10s %s\n",
		StyleMeta.Render("Address"),
		StyleAddress.Render(m.Address)))
	if m.Kind != "" {
		sb.WriteString(fmt.Sprintf("  %-10s %s\n",
			StyleMeta.Render("Kind"),
			StyleValue.Render(m.Kind)))
	}
	sb.WriteString(fmt.Sprintf("  %-10s %s · %s\n",
		StyleMeta.Render("ABI"),
		StyleInfo.Render(fmt.Sprintf("%d functions", m.FuncCount)),
		StyleMeta.Render(fmt.Sprintf("%d events", m.EventCount))))
	sb.WriteString("\n")

	// Build reverse-lookup: entry index → nav cursor position
	navPos := make(map[int]int, len(m.navItems))
	for pos, entIdx := range m.navItems {
		navPos[entIdx] = pos
	}

	// Split entries into three buckets
	var readIdxs, writeIdxs []int
	var eventEntries []StudioEntry
	for i, e := range m.Entries {
		if e.IsEvent {
			eventEntries = append(eventEntries, e)
		} else if e.IsWrite {
			writeIdxs = append(writeIdxs, i)
		} else {
			readIdxs = append(readIdxs, i)
		}
	}

	ruler := StyleMeta.Render(strings.Repeat("─", sepWidth))

	// ── Read section ──────────────────────────────────────────────────────
	if len(readIdxs) > 0 {
		hdr := fmt.Sprintf("  ── Read (%d) ", len(readIdxs))
		fill := sepWidth - len(hdr) - 2
		if fill < 0 {
			fill = 0
		}
		sb.WriteString(StyleHeader.Render(hdr) + StyleMeta.Render(strings.Repeat("─", fill)) + "\n")

		for _, idx := range readIdxs {
			e := m.Entries[idx]
			pos := navPos[idx]
			selected := pos == m.cursor

			outStr := ""
			if len(e.OutputTypes) > 0 {
				outStr = StyleMeta.Render("  →  " + strings.Join(e.OutputTypes, ", "))
			}

			prefix := "    "
			if selected {
				prefix = "  ▸ "
			}
			line := fmt.Sprintf("%s%s  %s(%s)%s",
				prefix,
				StyleMeta.Render(e.Selector),
				StyleValue.Render(e.Name),
				StyleMeta.Render(studioParamSig(e.Inputs)),
				outStr,
			)
			if selected {
				sb.WriteString(StyleSelected.Render(line) + "\n")
			} else {
				sb.WriteString(line + "\n")
			}
		}
		sb.WriteString("\n")
	}

	// ── Write section ─────────────────────────────────────────────────────
	if len(writeIdxs) > 0 {
		hdr := fmt.Sprintf("  ── Write (%d) ", len(writeIdxs))
		fill := sepWidth - len(hdr) - 2
		if fill < 0 {
			fill = 0
		}
		sb.WriteString(StyleHeader.Render(hdr) + StyleMeta.Render(strings.Repeat("─", fill)) + "\n")

		for _, idx := range writeIdxs {
			e := m.Entries[idx]
			pos := navPos[idx]
			selected := pos == m.cursor

			prefix := "    "
			if selected {
				prefix = "  ▸ "
			}
			line := fmt.Sprintf("%s%s  %s(%s)",
				prefix,
				StyleMeta.Render(e.Selector),
				StyleWarning.Render(e.Name),
				StyleMeta.Render(studioParamSig(e.Inputs)),
			)
			if selected {
				sb.WriteString(StyleSelected.Render(line) + "\n")
			} else {
				sb.WriteString(line + "\n")
			}
		}
		sb.WriteString("\n")
	}

	// ── Events section ────────────────────────────────────────────────────
	if len(eventEntries) > 0 {
		hdr := fmt.Sprintf("  ── Events (%d) ", len(eventEntries))
		fill := sepWidth - len(hdr) - 2
		if fill < 0 {
			fill = 0
		}
		sb.WriteString(StyleHeader.Render(hdr) + StyleMeta.Render(strings.Repeat("─", fill)) + "\n")
		for _, e := range eventEntries {
			sb.WriteString(fmt.Sprintf("    %s(%s)\n",
				StyleInfo.Render(e.Name),
				StyleMeta.Render(studioParamSig(e.Inputs))))
		}
		sb.WriteString("\n")
	}

	// ── Description panel ─────────────────────────────────────────────────
	sb.WriteString(ruler + "\n")
	if len(m.navItems) > 0 {
		cur := m.Entries[m.navItems[m.cursor]]
		desc := cur.Description
		if desc == "" {
			desc = cur.Sig
		}
		sb.WriteString(StyleMeta.Render("  "+desc) + "\n")
	}
	sb.WriteString(ruler + "\n\n")

	// ── Controls ──────────────────────────────────────────────────────────
	sb.WriteString(
		StyleMeta.Render("  [ ↑↓ / jk ]") + " navigate   " +
			StyleInfo.Render("[ Enter ]") + " select & call   " +
			StyleMeta.Render("[ q ]") + " quit\n")

	return sb.String()
}

// RunStudio launches the interactive function navigator with altscreen and
// returns the selected entry, or nil if the user quit.
func RunStudio(m StudioModel) (*StudioEntry, error) {
	m.buildNav()
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("studio: %w", err)
	}
	fm := final.(StudioModel)
	if fm.Quitting || fm.Selected == nil {
		return nil, nil
	}
	return fm.Selected, nil
}

// studioParamSig formats params as "type name, type name".
func studioParamSig(params []StudioParam) string {
	parts := make([]string, len(params))
	for i, p := range params {
		if p.Name != "" {
			parts[i] = p.Type + " " + p.Name
		} else {
			parts[i] = p.Type
		}
	}
	return strings.Join(parts, ", ")
}
