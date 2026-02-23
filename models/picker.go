package models

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"mailgloss/storage"
	"mailgloss/ui"
)

// PickerType represents the type of picker
type PickerType int

const (
	PickerTypeContact PickerType = iota
	PickerTypeTemplate
)

// PickerModel represents a modal picker for contacts or templates
type PickerModel struct {
	pickerType  PickerType
	contacts    []storage.Contact
	templates   []storage.Template
	selectedIdx int
	width       int
	height      int
	targetField int // Which field to populate (for contacts: to, cc, bcc)
}

// NewContactPicker creates a new contact picker
func NewContactPicker(contacts *storage.Contacts, targetField int) PickerModel {
	return PickerModel{
		pickerType:  PickerTypeContact,
		contacts:    contacts.GetAll(),
		selectedIdx: 0,
		targetField: targetField,
	}
}

// NewTemplatePicker creates a new template picker
func NewTemplatePicker(templates *storage.Templates) PickerModel {
	return PickerModel{
		pickerType:  PickerTypeTemplate,
		templates:   templates.GetAll(),
		selectedIdx: 0,
	}
}

// Update handles messages for the picker model
func (m PickerModel) Update(msg tea.Msg) (PickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		case "down", "j":
			maxIdx := 0
			if m.pickerType == PickerTypeContact {
				maxIdx = len(m.contacts) - 1
			} else {
				maxIdx = len(m.templates) - 1
			}
			if m.selectedIdx < maxIdx {
				m.selectedIdx++
			}
		case "enter":
			// Return selected item
			if m.pickerType == PickerTypeContact && len(m.contacts) > 0 {
				return m, func() tea.Msg {
					return ContactSelectedMsg{
						Contact:     m.contacts[m.selectedIdx],
						TargetField: m.targetField,
					}
				}
			} else if m.pickerType == PickerTypeTemplate && len(m.templates) > 0 {
				return m, func() tea.Msg {
					return TemplateSelectedMsg{
						Template: m.templates[m.selectedIdx],
					}
				}
			}
		case "esc", "q":
			return m, func() tea.Msg {
				return PickerClosedMsg{}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// View renders the picker
func (m PickerModel) View() string {
	var b strings.Builder

	if m.pickerType == PickerTypeContact {
		b.WriteString(ui.TitleStyle.Render("Select Contact"))
		b.WriteString("\n\n")

		if len(m.contacts) == 0 {
			b.WriteString(ui.ErrorStyle.Render("No contacts available."))
			b.WriteString("\n\n")
			b.WriteString("Add contacts in the Contacts tab first.\n")
		} else {
			for i, contact := range m.contacts {
				prefix := "  "
				style := ui.DisplayLabelStyle
				if i == m.selectedIdx {
					prefix = "▸ "
					style = style.Foreground(ui.Primary)
				}

				display := fmt.Sprintf("%s <%s>", contact.Name, contact.Email)
				b.WriteString(style.Render(prefix + display))
				b.WriteString("\n")
			}
		}
	} else {
		b.WriteString(ui.TitleStyle.Render("Select Template"))
		b.WriteString("\n\n")

		if len(m.templates) == 0 {
			b.WriteString(ui.ErrorStyle.Render("No templates available."))
			b.WriteString("\n\n")
			b.WriteString("Add templates in the Templates tab first.\n")
		} else {
			for i, template := range m.templates {
				prefix := "  "
				style := ui.DisplayLabelStyle
				if i == m.selectedIdx {
					prefix = "▸ "
					style = style.Foreground(ui.Primary)
				}

				display := template.Name
				if template.Description != "" {
					display += ui.LabelStyle.Render(" - " + template.Description)
				}
				b.WriteString(style.Render(prefix + display))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(ui.RenderHelp(
		"↑/↓", "navigate",
		"Enter", "select",
		"Esc/q", "cancel",
	))

	return b.String()
}

// ContactSelectedMsg is sent when a contact is selected
type ContactSelectedMsg struct {
	Contact     storage.Contact
	TargetField int
}

// TemplateSelectedMsg is sent when a template is selected
type TemplateSelectedMsg struct {
	Template storage.Template
}

// PickerClosedMsg is sent when the picker is closed
type PickerClosedMsg struct{}
