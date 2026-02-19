package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// WatchTxMsg is sent when a new matching transaction is found during polling.
type WatchTxMsg struct {
	Hash        string
	Direction   string // "←" incoming or "→" outgoing
	Counterpart string // truncated address of the other party
	ValueStr    string // formatted amount, e.g. "0.5000"
	Currency    string // e.g. "ETH"
	BlockNum    uint64
	TxRow       TxRow // for o/c shortcuts
}

// WatchStatusMsg updates the polling status bar.
type WatchStatusMsg struct {
	BlockNum uint64
	Fetching bool
	ErrMsg   string
}

// WatchModel is the Bubble Tea model for the live transaction stream.
type WatchModel struct {
	Address  string
	Chain    string
	Mode     string
	Rows     []WatchTxMsg
	TxData   []TxRow
	cursor   int
	Status   WatchStatusMsg
	Frame    int
	Quitting bool
	flash    string
}

type watchTickMsg struct{}

func watchSpinTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return watchTickMsg{}
	})
}

func (m WatchModel) Init() tea.Cmd { return watchSpinTick() }

func (m WatchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		m.flash = ""
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.Quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.Rows)-1 {
				m.cursor++
			}

		case "o":
			if m.cursor < len(m.TxData) {
				url := m.TxData[m.cursor].ExplorerURL
				if url != "" {
					openBrowser(url)
					m.flash = "Opening in browser…"
				} else {
					m.flash = "No explorer URL available"
				}
			}

		case "c":
			if m.cursor < len(m.TxData) {
				hash := m.TxData[m.cursor].FullHash
				if hash == "" {
					m.flash = "No hash available"
					break
				}
				if err := copyToClipboard(hash); err == nil {
					m.flash = "Copied: " + hash[:10] + "…"
				} else {
					m.flash = "Copy failed"
				}
			}
		}

	case watchTickMsg:
		m.Frame = (m.Frame + 1) % len(abSpinFrames)
		return m, watchSpinTick()

	case WatchTxMsg:
		// New transactions prepend so latest is at top.
		m.Rows = append([]WatchTxMsg{msg}, m.Rows...)
		m.TxData = append([]TxRow{msg.TxRow}, m.TxData...)
		// Cap at 200 rows.
		if len(m.Rows) > 200 {
			m.Rows = m.Rows[:200]
			m.TxData = m.TxData[:200]
		}

	case WatchStatusMsg:
		m.Status = msg
	}

	return m, nil
}

func (m WatchModel) View() string {
	if m.Quitting {
		return ""
	}

	var sb strings.Builder
	spin := abSpinFrames[m.Frame]

	// ── Title ─────────────────────────────────────────────────────────────
	title := fmt.Sprintf("👁  Live Transactions  ·  %s  ·  %s · %s",
		TruncateAddr(m.Address), m.Chain, m.Mode)
	sb.WriteString(StyleTitle.Render(title) + "\n")

	// ── Status bar ────────────────────────────────────────────────────────
	if m.Status.ErrMsg != "" {
		sb.WriteString(StyleError.Render("✗ "+m.Status.ErrMsg) + "\n\n")
	} else if m.Status.Fetching {
		sb.WriteString(StyleInfo.Render(fmt.Sprintf("%s polling block #%d…", spin, m.Status.BlockNum)) + "\n\n")
	} else if m.Status.BlockNum > 0 {
		sb.WriteString(StyleMeta.Render(fmt.Sprintf("  last checked: block #%d", m.Status.BlockNum)) + "\n\n")
	} else {
		sb.WriteString(StyleMeta.Render("  connecting…") + "\n\n")
	}

	// ── Table ─────────────────────────────────────────────────────────────
	const (
		wHash = 14
		wDir  = 2
		wAddr = 16
		wVal  = 16
		wBlk  = 10
	)
	sep := StyleMeta.Render(strings.Repeat("─", wHash+wDir+wAddr+wVal+wBlk+12))

	// Header
	sb.WriteString(
		padR(StyleDim.Render("HASH"), wHash) + "  " +
			padR(StyleDim.Render("DR"), wDir) + "  " +
			padR(StyleDim.Render("COUNTERPART"), wAddr) + "  " +
			padR(StyleDim.Render("VALUE"), wVal) + "  " +
			StyleDim.Render("BLOCK") + "\n",
	)
	sb.WriteString(sep + "\n")

	if len(m.Rows) == 0 {
		sb.WriteString(StyleMeta.Render("  Waiting for transactions…") + "\n")
	} else {
		for i, row := range m.Rows {
			hashStr := StyleAddress.Render(TruncateAddr(row.Hash))

			var dirStr string
			if row.Direction == "←" {
				dirStr = StyleSuccess.Render("←")
			} else {
				dirStr = StyleWarning.Render("→")
			}

			addrStr := StyleAddress.Render(row.Counterpart)

			var valStr string
			if row.ValueStr != "0.0000" && row.ValueStr != "" {
				valStr = StyleValue.Render(row.ValueStr) + " " + StyleDim.Render(row.Currency)
			} else {
				valStr = StyleDim.Render("0.0000 " + row.Currency)
			}

			blkStr := StyleMeta.Render(fmt.Sprintf("#%d", row.BlockNum))

			line :=
				padR(hashStr, wHash) + "  " +
					padR(dirStr, wDir) + "  " +
					padR(addrStr, wAddr) + "  " +
					padR(valStr, wVal) + "  " +
					blkStr

			if i == m.cursor {
				sb.WriteString(StyleSelected.Render(line) + "\n")
			} else {
				sb.WriteString(line + "\n")
			}
		}
		sb.WriteString(sep + "\n")
		sb.WriteString(StyleMeta.Render(fmt.Sprintf("  %d transaction(s) found", len(m.Rows))) + "\n")
	}

	// ── Controls ─────────────────────────────────────────────────────────
	sb.WriteString("\n")
	if m.flash != "" {
		sb.WriteString(StyleSuccess.Render("  ✓ " + m.flash))
	} else {
		sb.WriteString(watchControls())
	}
	sb.WriteString("\n")

	return sb.String()
}

func watchControls() string {
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
