package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ABStatus is the fetch state of one chain's balance result.
type ABStatus int

const (
	ABStatusFetching ABStatus = iota
	ABStatusDone
	ABStatusError
)

// AllBalRow holds mainnet and/or testnet state for a single chain.
type AllBalRow struct {
	ChainName   string
	DisplayName string
	Currency    string

	// Mainnet result.
	MainStatus  ABStatus
	MainBalance string
	MainUSD     string
	MainLatency time.Duration
	MainErr     string

	// Testnet result (only populated when Mode == "both").
	TestStatus  ABStatus
	TestBalance string
	TestUSD     string
	TestLatency time.Duration
	TestErr     string
}

func (r AllBalRow) mainUSDFloat() float64 {
	f, _ := strconv.ParseFloat(strings.TrimPrefix(r.MainUSD, "$"), 64)
	return f
}

// AllBalResult is sent by each fetch goroutine when it finishes.
type AllBalResult struct {
	ChainName string
	NetMode   string // "mainnet" or "testnet"
	Balance   string
	USD       string
	Latency   time.Duration
	Err       error
}

// AllBalResultMsg wraps AllBalResult as a Bubble Tea message.
type AllBalResultMsg AllBalResult

type allBalTickMsg struct{}

var abSpinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// AllBalModel is the Bubble Tea model for the multi-chain balance scanner.
type AllBalModel struct {
	Address  string
	Mode     string // "mainnet" | "testnet" | "both"
	Rows     []AllBalRow
	RowIndex map[string]int
	Total    int // total goroutines expected
	Done     int // goroutines that have responded
	Frame    int // spinner frame index
	Sorted   bool
	Quitting bool
}

func (m AllBalModel) Init() tea.Cmd {
	return abTick()
}

func abTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return allBalTickMsg{}
	})
}

func (m AllBalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.Quitting = true
			return m, tea.Quit
		}

	case allBalTickMsg:
		m.Frame = (m.Frame + 1) % len(abSpinFrames)
		// Once all goroutines are done, sort by USD value descending.
		if m.Done >= m.Total && !m.Sorted {
			m.Sorted = true
			sort.SliceStable(m.Rows, func(i, j int) bool {
				// Primary: mainnet USD (if available).
				ui, uj := m.Rows[i].mainUSDFloat(), m.Rows[j].mainUSDFloat()
				if ui != uj {
					return ui > uj
				}
				// Fallback: mainnet raw balance.
				bi, _ := strconv.ParseFloat(m.Rows[i].MainBalance, 64)
				bj, _ := strconv.ParseFloat(m.Rows[j].MainBalance, 64)
				if bi != bj {
					return bi > bj
				}
				// Fallback: testnet raw balance.
				ti, _ := strconv.ParseFloat(m.Rows[i].TestBalance, 64)
				tj, _ := strconv.ParseFloat(m.Rows[j].TestBalance, 64)
				return ti > tj
			})
		}
		return m, abTick()

	case AllBalResultMsg:
		idx, ok := m.RowIndex[msg.ChainName]
		if !ok {
			return m, nil
		}
		if msg.NetMode == "testnet" {
			if msg.Err != nil {
				m.Rows[idx].TestStatus = ABStatusError
				m.Rows[idx].TestErr = trimErr(msg.Err.Error())
			} else {
				m.Rows[idx].TestStatus = ABStatusDone
				m.Rows[idx].TestBalance = msg.Balance
				m.Rows[idx].TestUSD = msg.USD
				m.Rows[idx].TestLatency = msg.Latency
			}
		} else {
			if msg.Err != nil {
				m.Rows[idx].MainStatus = ABStatusError
				m.Rows[idx].MainErr = trimErr(msg.Err.Error())
			} else {
				m.Rows[idx].MainStatus = ABStatusDone
				m.Rows[idx].MainBalance = msg.Balance
				m.Rows[idx].MainUSD = msg.USD
				m.Rows[idx].MainLatency = msg.Latency
			}
		}
		m.Done++
	}

	return m, nil
}

func (m AllBalModel) View() string {
	if m.Quitting {
		return ""
	}

	var sb strings.Builder
	spin := abSpinFrames[m.Frame]

	// ── Title ─────────────────────────────────────────────────────────────
	title := fmt.Sprintf("⚡ All-Chain Balance  ·  %s  ·  mode: %s",
		TruncateAddr(m.Address), m.Mode)
	sb.WriteString(StyleTitle.Render(title) + "\n")

	// ── Progress bar ───────────────────────────────────────────────────────
	var progress string
	if m.Done >= m.Total {
		label := fmt.Sprintf("✓ %d/%d chains done", m.Done, m.Total)
		if m.Sorted {
			label += " · sorted by USD ↓"
		}
		progress = StyleSuccess.Render(label)
	} else {
		progress = StyleInfo.Render(fmt.Sprintf("%s %d/%d fetching…", spin, m.Done, m.Total))
	}
	sb.WriteString(progress + StyleMeta.Render("   press q to quit") + "\n\n")

	// ── Table ──────────────────────────────────────────────────────────────
	if m.Mode == "both" {
		sb.WriteString(m.viewBoth(spin))
	} else {
		sb.WriteString(m.viewSingle(spin))
	}

	return sb.String()
}

// viewSingle renders the table for a single network mode (mainnet or testnet).
func (m AllBalModel) viewSingle(spin string) string {
	const (
		wChain = 18
		wBal   = 24
		wUSD   = 12
		wLat   = 10
	)
	sep := StyleMeta.Render(strings.Repeat("─", wChain+wBal+wUSD+wLat+10))

	var sb strings.Builder

	// Header
	sb.WriteString(
		padR(StyleDim.Render("CHAIN"), wChain) + "  " +
			padR(StyleDim.Render("BALANCE"), wBal) + "  " +
			padR(StyleDim.Render("USD"), wUSD) + "  " +
			padR(StyleDim.Render("LATENCY"), wLat) + "  " +
			StyleDim.Render("STATUS") + "\n",
	)
	sb.WriteString(sep + "\n")

	var totalUSD float64

	for _, row := range m.Rows {
		balStr, usdStr, latStr, statStr := renderSingleRow(
			row.MainStatus, row.MainBalance, row.MainUSD,
			row.Currency, row.MainLatency, row.MainErr, spin,
		)

		if row.MainStatus == ABStatusDone {
			var u float64
			fmt.Sscanf(strings.TrimPrefix(row.MainUSD, "$"), "%f", &u)
			totalUSD += u
		}

		sb.WriteString(
			padR(ChainName(row.DisplayName), wChain) + "  " +
				padR(balStr, wBal) + "  " +
				padR(usdStr, wUSD) + "  " +
				padR(latStr, wLat) + "  " +
				statStr + "\n",
		)
	}

	sb.WriteString(sep + "\n")

	// Count chains with a non-zero balance (regardless of whether USD resolved).
	var nonZeroChains int
	for _, row := range m.Rows {
		if row.MainStatus == ABStatusDone {
			if bal, _ := strconv.ParseFloat(row.MainBalance, 64); bal > 0 {
				nonZeroChains++
			}
		}
	}

	// Footer: show USD total if available, otherwise a chain-count summary.
	switch {
	case totalUSD > 0:
		sb.WriteString(
			padR(StyleMeta.Render("TOTAL"), wChain) + "  " +
				padR("", wBal) + "  " +
				StyleSuccess.Render(fmt.Sprintf("$%.2f", totalUSD)) + "\n",
		)
	case nonZeroChains > 0:
		sb.WriteString(StyleInfo.Render(
			fmt.Sprintf("  Balances found on %d chain(s) · USD prices unavailable (rate limit)", nonZeroChains),
		) + "\n")
	case m.Done >= m.Total:
		sb.WriteString(StyleMeta.Render("  No balances found for this address on any chain.") + "\n")
	}

	return sb.String()
}

// viewBoth renders the dual-column mainnet + testnet table.
func (m AllBalModel) viewBoth(spin string) string {
	const (
		wChain = 18
		wBal   = 20
		wUSD   = 10
	)
	totalW := wChain + (wBal+2+wUSD)*2 + 18
	sep := StyleMeta.Render(strings.Repeat("─", totalW))

	var sb strings.Builder

	// Header
	sb.WriteString(
		padR(StyleDim.Render("CHAIN"), wChain) + "  " +
			padR(StyleDim.Render("MAINNET"), wBal) + "  " +
			padR(StyleDim.Render("USD"), wUSD) + "  " +
			padR(StyleDim.Render("TESTNET"), wBal) + "  " +
			padR(StyleDim.Render("USD"), wUSD) + "  " +
			StyleDim.Render("ST") + "\n",
	)
	sb.WriteString(sep + "\n")

	var totalMain, totalTest float64

	for _, row := range m.Rows {
		mainBal, mainUSD, _, mainStat := renderSingleRow(
			row.MainStatus, row.MainBalance, row.MainUSD,
			row.Currency, row.MainLatency, row.MainErr, spin,
		)
		testBal, testUSD, _, testStat := renderSingleRow(
			row.TestStatus, row.TestBalance, row.TestUSD,
			row.Currency, row.TestLatency, row.TestErr, spin,
		)

		// Combined status icon
		combined := StyleMeta.Render("…")
		switch {
		case row.MainStatus == ABStatusDone && row.TestStatus == ABStatusDone:
			combined = StyleSuccess.Render("✓✓")
		case row.MainStatus == ABStatusError && row.TestStatus == ABStatusError:
			combined = StyleError.Render("✗✗")
		case row.MainStatus == ABStatusDone || row.TestStatus == ABStatusDone:
			combined = StyleWarning.Render("½")
		case row.MainStatus == ABStatusError || row.TestStatus == ABStatusError:
			combined = StyleWarning.Render("!")
		case row.MainStatus == ABStatusFetching:
			combined = StyleMeta.Render(mainStat)
		}
		_ = testStat

		if row.MainStatus == ABStatusDone {
			var u float64
			fmt.Sscanf(strings.TrimPrefix(row.MainUSD, "$"), "%f", &u)
			totalMain += u
		}
		if row.TestStatus == ABStatusDone {
			var u float64
			fmt.Sscanf(strings.TrimPrefix(row.TestUSD, "$"), "%f", &u)
			totalTest += u
		}

		sb.WriteString(
			padR(ChainName(row.DisplayName), wChain) + "  " +
				padR(mainBal, wBal) + "  " +
				padR(mainUSD, wUSD) + "  " +
				padR(testBal, wBal) + "  " +
				padR(testUSD, wUSD) + "  " +
				combined + "\n",
		)
	}

	sb.WriteString(sep + "\n")

	// Count non-zero chains for each mode.
	var nonZeroMain, nonZeroTest int
	for _, row := range m.Rows {
		if row.MainStatus == ABStatusDone {
			if b, _ := strconv.ParseFloat(row.MainBalance, 64); b > 0 {
				nonZeroMain++
			}
		}
		if row.TestStatus == ABStatusDone {
			if b, _ := strconv.ParseFloat(row.TestBalance, 64); b > 0 {
				nonZeroTest++
			}
		}
	}

	switch {
	case totalMain > 0 || totalTest > 0:
		sb.WriteString(
			padR(StyleMeta.Render("TOTAL USD"), wChain) + "  " +
				padR("", wBal) + "  " +
				padR(StyleSuccess.Render(fmt.Sprintf("$%.2f", totalMain)), wUSD) + "  " +
				padR("", wBal) + "  " +
				StyleSuccess.Render(fmt.Sprintf("$%.2f", totalTest)) + "\n",
		)
	case nonZeroMain > 0 || nonZeroTest > 0:
		sb.WriteString(StyleInfo.Render(
			fmt.Sprintf("  Mainnet: %d chain(s) with balance  ·  Testnet: %d chain(s) with balance  ·  USD unavailable (rate limit)",
				nonZeroMain, nonZeroTest),
		) + "\n")
	case m.Done >= m.Total:
		sb.WriteString(StyleMeta.Render("  No balances found on any chain.") + "\n")
	}

	return sb.String()
}

// renderSingleRow returns (balStr, usdStr, latStr, statStr) for one result cell.
func renderSingleRow(status ABStatus, balance, usd, currency string, latency time.Duration, errMsg, spin string) (balStr, usdStr, latStr, statStr string) {
	switch status {
	case ABStatusFetching:
		return StyleMeta.Render(spin + " fetching…"),
			StyleMeta.Render("—"),
			StyleMeta.Render("—"),
			StyleMeta.Render("⏳")

	case ABStatusDone:
		bal, _ := strconv.ParseFloat(balance, 64)
		if bal > 0 {
			balStr = StyleValue.Render(balance) + " " + StyleDim.Render(currency)
			usdStr = StyleSuccess.Render(usd)
			statStr = StyleSuccess.Render("✓")
		} else {
			balStr = StyleDim.Render("0.000000 " + currency)
			usdStr = StyleDim.Render("$0.00")
			statStr = StyleDim.Render("·")
		}
		latStr = StyleMeta.Render(latency.Truncate(time.Millisecond).String())
		return

	case ABStatusError:
		short := errMsg
		if len(short) > 22 {
			short = short[:22] + "…"
		}
		return StyleError.Render("✗ " + short),
			StyleMeta.Render("—"),
			StyleMeta.Render("—"),
			StyleError.Render("✗")
	}
	return "", "", "", ""
}

// padR pads s to visible width n (ANSI-safe using lipgloss.Width).
func padR(s string, n int) string {
	w := lipgloss.Width(s)
	if w >= n {
		return s
	}
	return s + strings.Repeat(" ", n-w)
}

func trimErr(s string) string {
	// Strip common noisy prefixes from RPC error messages.
	for _, prefix := range []string{
		"Post \"", "dial tcp", "connection refused",
		"no RPCs configured", "context deadline",
	} {
		if strings.Contains(s, prefix) {
			if idx := strings.Index(s, prefix); idx >= 0 {
				s = s[idx:]
			}
			break
		}
	}
	if len(s) > 30 {
		return s[:30] + "…"
	}
	return s
}
