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

	// FetchFn is called when retrying a failed chain. It must return a tea.Cmd
	// that sends an AllBalResultMsg back to the program.
	FetchFn func(chainName, netMode string) tea.Cmd
}

func (m AllBalModel) Init() tea.Cmd {
	return abTick()
}

func abTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return allBalTickMsg{}
	})
}

// failCount returns the number of failed fetch slots for the current mode.
func (m AllBalModel) failCount() int {
	n := 0
	for _, row := range m.Rows {
		if m.Mode == "mainnet" || m.Mode == "both" {
			if row.MainStatus == ABStatusError {
				n++
			}
		}
		if m.Mode == "testnet" || m.Mode == "both" {
			if row.TestStatus == ABStatusError {
				n++
			}
		}
	}
	return n
}

func (m AllBalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.Quitting = true
			return m, tea.Quit

		case "o":
			OpenURL("https://debank.com/profile/" + m.Address)
			return m, nil

		case "r":
			if m.FetchFn == nil {
				return m, nil
			}
			var cmds []tea.Cmd
			for i := range m.Rows {
				if m.Mode == "mainnet" || m.Mode == "both" {
					if m.Rows[i].MainStatus == ABStatusError {
						m.Rows[i].MainStatus = ABStatusFetching
						m.Rows[i].MainErr = ""
						m.Done--
						m.Sorted = false
						cmds = append(cmds, m.FetchFn(m.Rows[i].ChainName, "mainnet"))
					}
				}
				if m.Mode == "testnet" || m.Mode == "both" {
					if m.Rows[i].TestStatus == ABStatusError {
						m.Rows[i].TestStatus = ABStatusFetching
						m.Rows[i].TestErr = ""
						m.Done--
						m.Sorted = false
						cmds = append(cmds, m.FetchFn(m.Rows[i].ChainName, "testnet"))
					}
				}
			}
			if len(cmds) > 0 {
				return m, tea.Batch(cmds...)
			}
			return m, nil
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

	// ── Progress line ───────────────────────────────────────────────────────
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
	sb.WriteString(progress + "\n\n")

	// ── Table ──────────────────────────────────────────────────────────────
	if m.Mode == "both" {
		sb.WriteString(m.viewBoth(spin))
	} else {
		sb.WriteString(m.viewSingle(spin))
	}

	// ── Controls (below table) ─────────────────────────────────────────────
	sb.WriteString("\n")
	sb.WriteString(abControls(m.failCount(), m.FetchFn != nil))

	return sb.String()
}

// viewSingle renders the table for a single network mode (mainnet or testnet).
func (m AllBalModel) viewSingle(spin string) string {
	const (
		wChain = 16
		wBal   = 18 // "9999.0000 METIS" = 15 chars — 18 gives breathing room
		wUSD   = 10 // "$12345.67" = 9 chars
		wLat   = 8  // "9.999s" = 6 chars
	)
	// Separator spans all columns + the 4 two-space gaps + a ~6-char status col.
	sep := StyleMeta.Render(strings.Repeat("─", wChain+wBal+wUSD+wLat+14))

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
		wChain = 16
		wBal   = 18
		wUSD   = 10
	)
	// Chain + 2*(Bal + "  " + USD) + "  " + ST(2) = 16 + 2*(18+2+10) + 2 + 2 = 82
	sep := StyleMeta.Render(strings.Repeat("─", wChain+2*(wBal+wUSD+2)+6))

	var sb strings.Builder

	// Header
	sb.WriteString(
		padR(StyleDim.Render("CHAIN"), wChain) + "  " +
			padR(StyleDim.Render("MAINNET BAL"), wBal) + "  " +
			padR(StyleDim.Render("USD"), wUSD) + "  " +
			padR(StyleDim.Render("TESTNET BAL"), wBal) + "  " +
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
			combined = mainStat
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
		balFmt := strconv.FormatFloat(bal, 'f', 4, 64)
		if bal > 0 {
			balStr = StyleValue.Render(balFmt) + " " + StyleDim.Render(currency)
			usdStr = StyleSuccess.Render(usd)
			statStr = StyleSuccess.Render("✓")
		} else {
			balStr = StyleDim.Render("0.0000 " + currency)
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

// abControls renders the consistent bottom control bar.
// Format:  [ r ] retry N failed   [ o ] open in browser   [ q ] quit
func abControls(failed int, canRetry bool) string {
	sep := StyleMeta.Render("   ")

	var sb strings.Builder
	if failed > 0 && canRetry {
		sb.WriteString(StyleWarning.Render("[ r ]"))
		sb.WriteString(" retry ")
		sb.WriteString(StyleError.Render(fmt.Sprintf("%d failed", failed)))
		sb.WriteString(sep)
	}
	sb.WriteString(StyleInfo.Render("[ o ]"))
	sb.WriteString(StyleMeta.Render(" open in browser"))
	sb.WriteString(sep)
	sb.WriteString(StyleMeta.Render("[ q ]"))
	sb.WriteString(StyleMeta.Render(" quit"))
	sb.WriteString("\n")
	return sb.String()
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
