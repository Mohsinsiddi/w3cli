package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// BalanceEntry holds one chain's balance for the dashboard.
type BalanceEntry struct {
	Chain   string
	Address string
	Balance string
	Symbol  string
	USD     string
}

// dashboardModel is the Bubble Tea model for the live balance dashboard.
type dashboardModel struct {
	entries   []BalanceEntry
	lastUpdate time.Time
	interval   time.Duration
	quitting   bool
	fetcher    func() ([]BalanceEntry, error)
	err        string
}

type tickMsg time.Time
type balanceFetchedMsg []BalanceEntry
type balanceErrorMsg string

// NewDashboard creates a Bubble Tea program for the live balance dashboard.
func NewDashboard(interval time.Duration, fetcher func() ([]BalanceEntry, error)) *tea.Program {
	m := dashboardModel{
		interval: interval,
		fetcher:  fetcher,
	}
	return tea.NewProgram(m)
}

func (m dashboardModel) Init() tea.Cmd {
	return tea.Batch(m.fetchCmd(), tick(m.interval))
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

	case tickMsg:
		return m, tea.Batch(m.fetchCmd(), tick(m.interval))

	case balanceFetchedMsg:
		m.entries = []BalanceEntry(msg)
		m.lastUpdate = time.Now()
		m.err = ""

	case balanceErrorMsg:
		m.err = string(msg)
	}

	return m, nil
}

func (m dashboardModel) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(StyleTitle.Render("⚡ Live Balance Dashboard") + "\n")
	sb.WriteString(StyleMeta.Render(fmt.Sprintf("Updated: %s · q to quit\n\n", m.lastUpdate.Format("15:04:05"))))

	if m.err != "" {
		sb.WriteString(Err(m.err) + "\n")
	}

	if len(m.entries) == 0 {
		sb.WriteString(StyleMeta.Render("Loading...") + "\n")
	} else {
		t := NewTable([]Column{
			{Title: "Chain", Width: 16},
			{Title: "Address", Width: 14},
			{Title: "Balance", Width: 22},
			{Title: "USD", Width: 12},
		})
		for _, e := range m.entries {
			t.AddRow(Row{
				ChainName(e.Chain),
				TruncateAddr(e.Address),
				Val(e.Balance) + " " + StyleMeta.Render(e.Symbol),
				StyleSuccess.Render(e.USD),
			})
		}
		sb.WriteString(t.Render())
	}

	return sb.String()
}

func (m dashboardModel) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		entries, err := m.fetcher()
		if err != nil {
			return balanceErrorMsg(err.Error())
		}
		return balanceFetchedMsg(entries)
	}
}

func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
