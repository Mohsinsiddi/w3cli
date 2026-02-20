package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// PickerItem is one entry shown in the interactive picker.
type PickerItem struct {
	Label    string // primary text (e.g. wallet name)
	SubLabel string // secondary text shown dimmed (e.g. address)
	Value    string // value returned on selection (may differ from Label)
}

// pickerModel is the Bubble Tea model for the interactive list picker.
type pickerModel struct {
	title    string
	items    []PickerItem
	cursor   int
	selected *PickerItem
	quitting bool
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", " ":
			if len(m.items) > 0 {
				item := m.items[m.cursor]
				m.selected = &item
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m pickerModel) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(StyleTitle.Render("  "+m.title) + "\n\n")

	for i, item := range m.items {
		prefix := "    "
		if i == m.cursor {
			prefix = "  ▸ "
		}

		line := prefix + StyleValue.Render(item.Label)
		if item.SubLabel != "" {
			line += "  " + StyleMeta.Render(item.SubLabel)
		}

		if i == m.cursor {
			sb.WriteString(StyleSelected.Render(line) + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(StyleMeta.Render("  [ ↑↓ / jk ] navigate   [ Enter ] select   [ q ] cancel") + "\n")
	return sb.String()
}

// PickItem runs an interactive list picker and returns the selected item's Value.
// Returns ("", nil) if the user cancels. Returns an error only on TUI failure.
func PickItem(title string, items []PickerItem) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no items to pick from")
	}

	m := pickerModel{title: title, items: items}
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("picker: %w", err)
	}

	fm := final.(pickerModel)
	if fm.quitting || fm.selected == nil {
		return "", nil
	}
	return fm.selected.Value, nil
}
