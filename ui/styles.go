package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Color palette
	Primary   = lipgloss.Color("205") // Pink
	Secondary = lipgloss.Color("99")  // Purple
	Success   = lipgloss.Color("42")  // Green
	Error     = lipgloss.Color("196") // Red
	Warning   = lipgloss.Color("214") // Orange
	Info      = lipgloss.Color("86")  // Cyan
	Muted     = lipgloss.Color("241") // Gray
	Highlight = lipgloss.Color("212") // Light pink

	// Tab styles
	ActiveTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	InactiveTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┴",
		BottomRight: "┴",
	}

	ActiveTabStyle = lipgloss.NewStyle().
			Border(ActiveTabBorder, true).
			BorderForeground(Primary).
			Padding(0, 1).
			Bold(true).
			Foreground(Primary)

	InactiveTabStyle = lipgloss.NewStyle().
				Border(InactiveTabBorder, true).
				BorderForeground(Muted).
				Padding(0, 1).
				Foreground(Muted)

	TabGap = lipgloss.NewStyle().
		Padding(0, 1)

	// Window styles
	WindowStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			Padding(0, 0, 1, 0)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Italic(true).
			Padding(0, 0, 1, 0)

	// Form styles
	LabelStyle = lipgloss.NewStyle().
			Foreground(Info).
			Bold(true).
			Width(15)

	// Display label (no fixed width, prevents wrapping)
	DisplayLabelStyle = lipgloss.NewStyle().
				Foreground(Info).
				Bold(true)

	InputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	FocusedInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(Primary).
				Padding(0, 1)

	PlaceholderStyle = lipgloss.NewStyle().
				Foreground(Muted).
				Italic(true)

	// Button styles
	ButtonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(Primary).
			Padding(0, 3).
			MarginTop(1).
			Bold(true)

	ButtonFocusedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Background(Highlight).
				Padding(0, 3).
				MarginTop(1).
				Bold(true)

	// Message styles
	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true).
			Padding(1, 2)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true).
			Padding(1, 2)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Info).
			Padding(1, 2)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true).
			Padding(1, 2)

	// List styles
	ListItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	SelectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(Primary).
				Bold(true)

	ListTitleStyle = lipgloss.NewStyle().
			Foreground(Info).
			Bold(true).
			Padding(0, 0, 0, 2)

	// Help styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(1, 0, 0, 0)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(Info)

	// Status styles
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(Muted).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(Muted).
			Padding(0, 1)

	StatusInfoStyle = lipgloss.NewStyle().
			Foreground(Info)

	// Divider
	DividerStyle = lipgloss.NewStyle().
			Foreground(Muted)
)

// RenderTabs renders the tab bar
func RenderTabs(tabs []string, activeTab int) string {
	var renderedTabs []string

	for i, tab := range tabs {
		var style lipgloss.Style
		if i == activeTab {
			style = ActiveTabStyle
		} else {
			style = InactiveTabStyle
		}
		renderedTabs = append(renderedTabs, style.Render(tab))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

// RenderFormField renders a form field with label and input
func RenderFormField(label, value string, focused bool) string {
	labelPart := LabelStyle.Render(label + ":")

	var valuePart string
	if focused {
		valuePart = FocusedInputStyle.Render(value)
	} else {
		valuePart = InputStyle.Render(value)
	}

	return labelPart + "\n" + valuePart
}

// RenderHelp renders help text
func RenderHelp(keys ...string) string {
	var parts []string
	for i := 0; i < len(keys); i += 2 {
		if i+1 < len(keys) {
			key := HelpKeyStyle.Render(keys[i])
			desc := keys[i+1]
			parts = append(parts, key+" "+desc)
		}
	}
	return HelpStyle.Render(strings.Join(parts, " • "))
}

// Divider returns a horizontal divider
func Divider(width int) string {
	if width <= 0 {
		width = 80
	}
	divider := ""
	for i := 0; i < width; i++ {
		divider += "─"
	}
	return DividerStyle.Render(divider)
}

// CalculateInputWidth calculates responsive input width based on terminal width
func CalculateInputWidth(termWidth int) int {
	// Default width
	return 60
}

// CalculateTextAreaWidth calculates responsive textarea width
func CalculateTextAreaWidth(termWidth int) int {
	// Default width
	return 64
}

// LoadingSpinner returns a spinner style
var SpinnerStyle = lipgloss.NewStyle().
	Foreground(Primary).
	Bold(true)
