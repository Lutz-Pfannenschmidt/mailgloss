package models

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"mailgloss/storage"
	"mailgloss/ui"
)

// VariablePromptModel represents a modal prompt for collecting template variable values
type VariablePromptModel struct {
	template   storage.Template
	variables  []string
	inputs     []textinput.Model
	focusIndex int
	values     map[string]string
	defaults   map[string]string
	width      int
	height     int
	cancelled  bool
}

// NewVariablePrompt creates a new variable prompt for a template
// defaults is a map of default values for variables (e.g. date, from_name)
// Values in defaults will be prefilled in the inputs but can be overridden by the user.
func NewVariablePrompt(template storage.Template, defaults map[string]string) VariablePromptModel {
	// Start with template-defined variables
	variables := make([]string, 0, len(template.Variables)+3)
	seen := make(map[string]bool)
	for _, v := range template.Variables {
		seen[v] = true
		variables = append(variables, v)
	}

	// Add system variables with sensible defaults if not present
	sysVars := []string{"date", "from_name", "from_email"}
	for _, sv := range sysVars {
		if !seen[sv] {
			variables = append(variables, sv)
			seen[sv] = true
		}
	}
	inputs := make([]textinput.Model, len(variables))

	// Create text inputs for each variable
	for i, varName := range variables {
		input := textinput.New()
		input.Placeholder = "Enter value for " + varName
		input.CharLimit = 500
		input.Width = 60

		// Prefill with default if available
		if v, ok := defaults[varName]; ok {
			input.SetValue(v)
		}

		// Focus the first input
		if i == 0 {
			input.Focus()
		}

		inputs[i] = input
	}

	return VariablePromptModel{
		template:   template,
		variables:  variables,
		inputs:     inputs,
		focusIndex: 0,
		values:     make(map[string]string),
		defaults:   defaults,
		cancelled:  false,
	}
}

// Update handles messages for the variable prompt
func (m VariablePromptModel) Update(msg tea.Msg) (VariablePromptModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel and close
			m.cancelled = true
			return m, func() tea.Msg {
				return VariablePromptClosedMsg{Cancelled: true}
			}

		case "tab", "shift+tab", "up", "down":
			// Navigate between inputs
			s := msg.String()
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex >= len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}

			// Update focus
			for i := range m.inputs {
				m.inputs[i].Blur()
			}
			if m.focusIndex < len(m.inputs) {
				cmds = append(cmds, m.inputs[m.focusIndex].Focus())
			}

			return m, tea.Batch(cmds...)

		case "enter":
			// Submit values
			for i, varName := range m.variables {
				m.values[varName] = m.inputs[i].Value()
			}
			return m, func() tea.Msg {
				return VariablePromptSubmittedMsg{
					Template: m.template,
					Values:   m.values,
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Update focused input
	if m.focusIndex >= 0 && m.focusIndex < len(m.inputs) {
		var cmd tea.Cmd
		m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the variable prompt
func (m VariablePromptModel) View() string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Template Variables"))
	b.WriteString("\n\n")

	b.WriteString(ui.SubtitleStyle.Render("Template: " + m.template.Name))
	b.WriteString("\n\n")

	b.WriteString(ui.InfoStyle.Render("Please provide values for the following variables:"))
	b.WriteString("\n\n")

	// Render each variable input
	for i, varName := range m.variables {
		focused := i == m.focusIndex
		labelStyle := ui.LabelStyle
		if focused {
			labelStyle = labelStyle.Foreground(ui.Primary)
		}

		// Indicate if this variable has a default value
		label := varName
		if m.defaults != nil {
			if def, ok := m.defaults[varName]; ok && def != "" {
				label = label + " (default)"
			}
		}
		b.WriteString(labelStyle.Render(label + ":"))
		b.WriteString("\n")

		if focused {
			b.WriteString(ui.FocusedInputStyle.Render(m.inputs[i].View()))
		} else {
			b.WriteString(m.inputs[i].View())
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(ui.RenderHelp(
		"Tab/↑/↓", "navigate",
		"Enter", "submit",
		"Esc", "cancel",
	))

	return b.String()
}

// VariablePromptSubmittedMsg is sent when the user submits variable values
type VariablePromptSubmittedMsg struct {
	Template storage.Template
	Values   map[string]string
}

// VariablePromptClosedMsg is sent when the variable prompt is closed
type VariablePromptClosedMsg struct {
	Cancelled bool
}
