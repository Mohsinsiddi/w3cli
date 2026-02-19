package ui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// TxRow holds per-transaction data needed for interactivity.
type TxRow struct {
	FullHash    string // full 0x... hash (for copy)
	ExplorerURL string // e.g. https://eth.blockscout.com/tx/0x...
}

// txListModel is the bubbletea model for the interactive tx table.
type txListModel struct {
	title  string
	table  *Table
	txData []TxRow // parallel to table.Rows
	cursor int
	flash  string // brief feedback shown in hint bar
}

func (m txListModel) Init() tea.Cmd { return nil }

func (m txListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.flash = ""
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.table.Rows)-1 {
				m.cursor++
			}

		case "o":
			if m.cursor < len(m.txData) {
				url := m.txData[m.cursor].ExplorerURL
				if url != "" {
					openBrowser(url)
					m.flash = "Opening in browser…"
				} else {
					m.flash = "No explorer URL available"
				}
			}

		case "c":
			if m.cursor < len(m.txData) {
				hash := m.txData[m.cursor].FullHash
				if hash == "" {
					m.flash = "No hash available"
					break
				}
				if err := copyToClipboard(hash); err == nil {
					m.flash = "Copied: " + hash[:10] + "…"
				} else {
					m.flash = "Copy failed: " + err.Error()
				}
			}
		}
	}
	return m, nil
}

func (m txListModel) View() string {
	m.table.SelIdx = m.cursor

	var sb strings.Builder

	// Title — same style as allbal.
	sb.WriteString(m.title)
	sb.WriteString("\n\n")

	// Table.
	sb.WriteString(m.table.Render())

	// Controls bar — same bracket format as allbal.
	sb.WriteString("\n")
	if m.flash != "" {
		sb.WriteString(StyleSuccess.Render("  ✓ " + m.flash))
	} else {
		sb.WriteString(txControls())
	}
	sb.WriteString("\n")

	return sb.String()
}

// txControls renders the consistent bottom control bar for the tx table.
func txControls() string {
	sep := StyleMeta.Render("   ")
	var sb strings.Builder
	sb.WriteString(StyleMeta.Render("[ ↑↓ ]"))
	sb.WriteString(StyleMeta.Render(" navigate"))
	sb.WriteString(sep)
	sb.WriteString(StyleInfo.Render("[ o ]"))
	sb.WriteString(StyleMeta.Render(" open in browser"))
	sb.WriteString(sep)
	sb.WriteString(StyleWarning.Render("[ c ]"))
	sb.WriteString(StyleMeta.Render(" copy hash"))
	sb.WriteString(sep)
	sb.WriteString(StyleMeta.Render("[ q ]"))
	sb.WriteString(StyleMeta.Render(" quit"))
	return sb.String()
}

// RunTxList starts the interactive transaction list. Blocks until the user
// presses q/ESC. Uses the alt screen so the terminal is restored on exit.
func RunTxList(title string, table *Table, txData []TxRow) error {
	m := txListModel{
		title:  title,
		table:  table,
		txData: txData,
	}
	p := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// openBrowser opens url in the OS default browser.
func openBrowser(url string) {
	var name string
	switch runtime.GOOS {
	case "darwin":
		name = "open"
	case "windows":
		name = "cmd"
	default:
		name = "xdg-open"
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command(name, "/c", "start", url)
	} else {
		cmd = exec.Command(name, url)
	}
	_ = cmd.Start()
}

// copyToClipboard writes text to the system clipboard.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("clip")
	default:
		// Try wl-copy (Wayland), fall back to xclip.
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("clipboard: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("clipboard: %w", err)
	}
	_, _ = io.WriteString(stdin, text)
	stdin.Close()
	return cmd.Wait()
}
