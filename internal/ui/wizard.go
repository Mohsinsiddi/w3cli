package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WizardResult holds answers collected by the setup wizard.
type WizardResult struct {
	DefaultNetwork  string
	NetworkMode     string
	RPCAlgorithm    string
	WalletAddress   string
	WalletName      string
}

// --- Bubble Tea model ---

type wizardStep int

const (
	stepNetwork wizardStep = iota
	stepMode
	stepAlgorithm
	stepWallet
	stepDone
)

type wizardModel struct {
	step     wizardStep
	result   WizardResult
	cursor   int
	choices  []string
	input    string
	inputMode bool
}

var networks = []string{
	"ethereum", "base", "polygon", "arbitrum", "optimism",
	"bnb", "avalanche", "fantom", "linea", "zksync",
}

var modes = []string{"mainnet", "testnet"}
var algorithms = []string{"fastest", "round-robin", "failover"}

func initialWizard() wizardModel {
	return wizardModel{
		step:    stepNetwork,
		choices: networks,
	}
}

func (m wizardModel) Init() tea.Cmd { return nil }

func (m wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if !m.inputMode && m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if !m.inputMode && m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter":
			if m.inputMode {
				m.applyInput()
				m.inputMode = false
				m.cursor = 0
				m.advance()
			} else {
				m.applyChoice()
				m.cursor = 0
				m.advance()
			}

		case "backspace":
			if m.inputMode && len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}

		default:
			if m.inputMode {
				m.input += msg.String()
			}
		}
	}

	if m.step == stepDone {
		return m, tea.Quit
	}
	return m, nil
}

func (m *wizardModel) advance() {
	m.step++
	switch m.step {
	case stepMode:
		m.choices = modes
	case stepAlgorithm:
		m.choices = algorithms
	case stepWallet:
		m.choices = nil
		m.inputMode = true
		m.input = ""
	}
}

func (m *wizardModel) applyChoice() {
	switch m.step {
	case stepNetwork:
		if m.cursor < len(m.choices) {
			m.result.DefaultNetwork = m.choices[m.cursor]
		}
	case stepMode:
		if m.cursor < len(m.choices) {
			m.result.NetworkMode = m.choices[m.cursor]
		}
	case stepAlgorithm:
		if m.cursor < len(m.choices) {
			m.result.RPCAlgorithm = m.choices[m.cursor]
		}
	}
}

func (m *wizardModel) applyInput() {
	switch m.step {
	case stepWallet:
		// Sanitize: strip whitespace and accidental brackets from paste.
		addr := strings.TrimSpace(m.input)
		addr = strings.Trim(addr, "[]")
		if addr != "" {
			m.result.WalletAddress = addr
			m.result.WalletName = "default"
		}
		m.step = stepDone - 1 // will be incremented in advance()
	}
}

func (m wizardModel) View() string {
	var s string

	switch m.step {
	case stepNetwork:
		s = renderMenu("Select default network:", m.choices, m.cursor)
	case stepMode:
		s = renderMenu("Select network mode:", m.choices, m.cursor)
	case stepAlgorithm:
		s = renderMenu("Select RPC algorithm:", m.choices, m.cursor)
	case stepWallet:
		s = StyleTitle.Render("Add a watch-only wallet (optional)") + "\n\n"
		s += StyleMeta.Render("Enter wallet address (or press Enter to skip):") + "\n"
		s += "> " + StyleAddress.Render(m.input) + "█\n"
	case stepDone:
		s = Success("Setup complete!") + "\n"
	}

	return StyleBorder.Render(s) + "\n"
}

func renderMenu(title string, items []string, cursor int) string {
	s := StyleTitle.Render(title) + "\n\n"
	for i, item := range items {
		icon := "  "
		style := lipgloss.NewStyle().Foreground(ColorValue)
		if i == cursor {
			icon = "▸ "
			style = StyleSelected
		}
		s += icon + style.Render(item) + "\n"
	}
	s += "\n" + StyleMeta.Render("↑/↓ navigate · Enter select · q quit")
	return s
}

// RunWizard launches the interactive setup wizard and returns the result.
func RunWizard() (*WizardResult, error) {
	m := initialWizard()
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("wizard error: %w", err)
	}
	result := final.(wizardModel).result
	return &result, nil
}
