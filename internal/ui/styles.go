package ui

import "github.com/charmbracelet/lipgloss"

// Color palette.
var (
	ColorSuccess   = lipgloss.Color("#00D26A") // green  — receive, success
	ColorWarning   = lipgloss.Color("#FFB800") // yellow — send, warning
	ColorError     = lipgloss.Color("#FF4444") // red    — error, danger
	ColorInfo      = lipgloss.Color("#00B4D8") // cyan   — info, addresses, hashes
	ColorAddress   = lipgloss.Color("#00B4D8") // cyan   — addresses, hashes
	ColorValue     = lipgloss.Color("#FFFFFF") // white bold — ETH values
	ColorMeta      = lipgloss.Color("#555555") // dim gray  — timestamps, metadata
	ColorHint      = lipgloss.Color("#888888") // light gray — hints, tips
	ColorBorder    = lipgloss.Color("#1E3A5F") // dark blue — UI chrome
	ColorChain     = lipgloss.Color("#9B5DE5") // purple    — chain names
	ColorHighlight = lipgloss.Color("#F15BB5") // pink      — selected rows
)

// Base styles.
var (
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true)
	StyleError   = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
	StyleInfo    = lipgloss.NewStyle().Foreground(ColorInfo).Bold(true)
	StyleHint    = lipgloss.NewStyle().Foreground(ColorHint).Italic(true)
	StyleAddress = lipgloss.NewStyle().Foreground(ColorAddress)
	StyleValue   = lipgloss.NewStyle().Foreground(ColorValue).Bold(true)
	StyleMeta    = lipgloss.NewStyle().Foreground(ColorMeta)
	StyleChain   = lipgloss.NewStyle().Foreground(ColorChain).Bold(true)

	StyleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	StyleHeader = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true).
			Underline(true)

	StyleSelected = lipgloss.NewStyle().
			Background(ColorHighlight).
			Foreground(lipgloss.Color("#000000")).
			Bold(true)

	StyleTitle = lipgloss.NewStyle().
			Foreground(ColorChain).
			Bold(true).
			MarginBottom(1)

	StyleDim = lipgloss.NewStyle().Foreground(ColorMeta)
)

// Banner returns the w3cli ASCII banner.
func Banner() string {
	art := `
  ██╗    ██╗██████╗  ██████╗██╗     ██╗
  ██║    ██║╚════██╗██╔════╝██║     ██║
  ██║ █╗ ██║ █████╔╝██║     ██║     ██║
  ██║███╗██║ ╚═══██╗██║     ██║     ██║
  ╚███╔███╔╝██████╔╝╚██████╗███████╗██║
   ╚══╝╚══╝ ╚═════╝  ╚═════╝╚══════╝╚═╝`

	tagline := StyleMeta.Render("     The Web3 Power CLI  ⚡  v1.0.0")
	features := StyleMeta.Render("  ✦ 26 chains  ✦ Smart RPC  ✦ Contract Studio")

	return StyleChain.Render(art) + "\n" + tagline + "\n" + features + "\n"
}

// Success formats a success message.
func Success(msg string) string { return StyleSuccess.Render("✓ " + msg) }

// Warn formats a warning message.
func Warn(msg string) string { return StyleWarning.Render("⚠ " + msg) }

// Err formats an error message.
func Err(msg string) string { return StyleError.Render("✗ " + msg) }

// Addr formats an address.
func Addr(a string) string { return StyleAddress.Render(a) }

// Val formats a value.
func Val(v string) string { return StyleValue.Render(v) }

// Meta formats metadata text.
func Meta(m string) string { return StyleMeta.Render(m) }

// Info formats an informational message.
func Info(msg string) string { return StyleInfo.Render("ℹ " + msg) }

// Hint formats a hint/tip message.
func Hint(msg string) string { return StyleHint.Render("💡 " + msg) }

// ChainName formats a chain name.
func ChainName(c string) string { return StyleChain.Render(c) }

// TruncateAddr shortens an address for display: 0x1234…5678.
func TruncateAddr(addr string) string {
	if len(addr) <= 10 {
		return addr
	}
	return addr[:6] + "…" + addr[len(addr)-4:]
}
