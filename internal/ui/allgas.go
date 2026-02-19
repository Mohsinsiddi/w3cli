package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// AGStatus mirrors ABStatus for gas fetching.
type AGStatus int

const (
	AGStatusFetching AGStatus = iota
	AGStatusDone
	AGStatusError
)

// AllGasRow holds gas pricing data for one chain.
type AllGasRow struct {
	ChainName   string
	DisplayName string
	Status      AGStatus
	GasGwei     float64 // primary display price (base fee if EIP-1559, else legacy)
	BaseFeeGwei float64 // raw base fee (0 if legacy)
	IsEIP1559   bool
	Latency     time.Duration
	ErrMsg      string
}

// AllGasResult is sent by each fetch goroutine when it finishes.
type AllGasResult struct {
	ChainName   string
	GasGwei     float64
	BaseFeeGwei float64
	IsEIP1559   bool
	Latency     time.Duration
	Err         error
}

// AllGasResultMsg wraps AllGasResult as a Bubble Tea message.
type AllGasResultMsg AllGasResult

type allGasTickMsg struct{}

// AllGasModel is the Bubble Tea model for the multi-chain gas scanner.
type AllGasModel struct {
	Mode     string
	Rows     []AllGasRow
	RowIndex map[string]int
	Total    int
	Done     int
	Frame    int
	Sorted   bool
	Quitting bool
	FetchFn  func(chainName string) tea.Cmd
}

func (m AllGasModel) Init() tea.Cmd { return agTick() }

func agTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return allGasTickMsg{}
	})
}

func (m AllGasModel) failCount() int {
	n := 0
	for _, row := range m.Rows {
		if row.Status == AGStatusError {
			n++
		}
	}
	return n
}

func (m AllGasModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.Quitting = true
			return m, tea.Quit

		case "r":
			if m.FetchFn == nil {
				return m, nil
			}
			var cmds []tea.Cmd
			for i := range m.Rows {
				if m.Rows[i].Status == AGStatusError {
					m.Rows[i].Status = AGStatusFetching
					m.Rows[i].ErrMsg = ""
					m.Done--
					m.Sorted = false
					cmds = append(cmds, m.FetchFn(m.Rows[i].ChainName))
				}
			}
			if len(cmds) > 0 {
				return m, tea.Batch(cmds...)
			}
			return m, nil
		}

	case allGasTickMsg:
		m.Frame = (m.Frame + 1) % len(abSpinFrames)
		if m.Done >= m.Total && !m.Sorted {
			m.Sorted = true
			// Sort cheapest gas first — most useful ordering.
			sort.SliceStable(m.Rows, func(i, j int) bool {
				gi, gj := m.Rows[i].GasGwei, m.Rows[j].GasGwei
				// Errors sink to bottom.
				if m.Rows[i].Status == AGStatusError {
					return false
				}
				if m.Rows[j].Status == AGStatusError {
					return true
				}
				return gi < gj
			})
		}
		return m, agTick()

	case AllGasResultMsg:
		idx, ok := m.RowIndex[msg.ChainName]
		if !ok {
			return m, nil
		}
		if msg.Err != nil {
			m.Rows[idx].Status = AGStatusError
			m.Rows[idx].ErrMsg = trimErr(msg.Err.Error())
		} else {
			m.Rows[idx].Status = AGStatusDone
			m.Rows[idx].GasGwei = msg.GasGwei
			m.Rows[idx].BaseFeeGwei = msg.BaseFeeGwei
			m.Rows[idx].IsEIP1559 = msg.IsEIP1559
			m.Rows[idx].Latency = msg.Latency
		}
		m.Done++
	}

	return m, nil
}

func (m AllGasModel) View() string {
	if m.Quitting {
		return ""
	}

	var sb strings.Builder
	spin := abSpinFrames[m.Frame]

	// ── Title ────────────────────────────────────────────────────────────
	sb.WriteString(StyleTitle.Render(fmt.Sprintf("⛽ All-Chain Gas Prices  ·  mode: %s", m.Mode)) + "\n")

	// ── Progress ─────────────────────────────────────────────────────────
	var progress string
	if m.Done >= m.Total {
		label := fmt.Sprintf("✓ %d/%d chains done", m.Done, m.Total)
		if m.Sorted {
			label += " · sorted by gas ↑ (cheapest first)"
		}
		progress = StyleSuccess.Render(label)
	} else {
		progress = StyleInfo.Render(fmt.Sprintf("%s %d/%d fetching…", spin, m.Done, m.Total))
	}
	sb.WriteString(progress + "\n\n")

	// ── Table ─────────────────────────────────────────────────────────────
	const (
		wChain = 16
		wGas   = 14 // "999999.99" + " Gwei" fits in 14
		wBase  = 14
		wType  = 8 // "EIP-1559"
		wLat   = 8
	)
	sep := StyleMeta.Render(strings.Repeat("─", wChain+wGas+wBase+wType+wLat+14))

	sb.WriteString(
		padR(StyleDim.Render("CHAIN"), wChain) + "  " +
			padR(StyleDim.Render("GAS (GWEI)"), wGas) + "  " +
			padR(StyleDim.Render("BASE FEE"), wBase) + "  " +
			padR(StyleDim.Render("TYPE"), wType) + "  " +
			padR(StyleDim.Render("LATENCY"), wLat) + "  " +
			StyleDim.Render("STATUS") + "\n",
	)
	sb.WriteString(sep + "\n")

	for _, row := range m.Rows {
		gasStr, baseStr, typeStr, latStr, statStr := renderGasRow(row, spin)
		sb.WriteString(
			padR(ChainName(row.DisplayName), wChain) + "  " +
				padR(gasStr, wGas) + "  " +
				padR(baseStr, wBase) + "  " +
				padR(typeStr, wType) + "  " +
				padR(latStr, wLat) + "  " +
				statStr + "\n",
		)
	}

	sb.WriteString(sep + "\n\n")

	// ── Controls ─────────────────────────────────────────────────────────
	sb.WriteString(abControls(m.failCount(), m.FetchFn != nil))
	return sb.String()
}

func renderGasRow(row AllGasRow, spin string) (gasStr, baseStr, typeStr, latStr, statStr string) {
	switch row.Status {
	case AGStatusFetching:
		return StyleMeta.Render(spin + " fetching…"),
			StyleMeta.Render("—"),
			StyleMeta.Render("—"),
			StyleMeta.Render("—"),
			StyleMeta.Render("⏳")

	case AGStatusDone:
		gasStr = StyleValue.Render(formatGwei(row.GasGwei))
		if row.IsEIP1559 {
			baseStr = StyleInfo.Render(formatGwei(row.BaseFeeGwei))
			typeStr = StyleSuccess.Render("EIP-1559")
		} else {
			baseStr = StyleMeta.Render("—")
			typeStr = StyleMeta.Render("legacy")
		}
		latStr = StyleMeta.Render(row.Latency.Truncate(time.Millisecond).String())
		statStr = StyleSuccess.Render("✓")
		return

	case AGStatusError:
		short := row.ErrMsg
		if len(short) > 22 {
			short = short[:22] + "…"
		}
		return StyleError.Render("✗ " + short),
			StyleMeta.Render("—"),
			StyleMeta.Render("—"),
			StyleMeta.Render("—"),
			StyleError.Render("✗")
	}
	return "", "", "", "", ""
}

// formatGwei formats a Gwei value with appropriate decimal precision.
func formatGwei(g float64) string {
	switch {
	case g == 0:
		return "0"
	case g < 0.001:
		return fmt.Sprintf("%.6f", g)
	case g < 1:
		return fmt.Sprintf("%.4f", g)
	case g < 100:
		return fmt.Sprintf("%.2f", g)
	default:
		return fmt.Sprintf("%.0f", g)
	}
}
