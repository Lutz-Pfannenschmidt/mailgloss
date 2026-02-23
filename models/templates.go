package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"mailgloss/storage"
	"mailgloss/ui"
)

// TemplatesView represents the different views in templates
type TemplatesView int

const (
	TemplatesViewList TemplatesView = iota
	TemplatesViewAdd
	TemplatesViewEdit
	TemplatesViewDetail
)

// TemplatesModel represents the templates tab
type TemplatesModel struct {
	templates   *storage.Templates
	currentView TemplatesView

	// List view
	templateList []storage.Template
	selectedIdx  int

	// Detail view
	detailTemplate *storage.Template

	// Add/Edit view
	isEditing  bool
	editingID  string
	inputs     []textinput.Model
	bodyArea   textarea.Model
	FocusIndex int

	// Status
	saved     bool
	saveError string
	width     int
	height    int
}

const (
	templateName = iota
	templateSubject
	templateBody
	templateDescription
	templateTags
	templateSaveButton
	templateCancelButton
)

// NewTemplatesModel creates a new templates model
func NewTemplatesModel(templates *storage.Templates) TemplatesModel {
	return TemplatesModel{
		templates:    templates,
		currentView:  TemplatesViewList,
		templateList: templates.GetAll(),
		selectedIdx:  0,
	}
}

// Init initializes the templates model
func (m TemplatesModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the templates model
func (m TemplatesModel) Update(msg tea.Msg) (TemplatesModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.currentView {
		case TemplatesViewList:
			return m.updateListView(msg)
		case TemplatesViewDetail:
			return m.updateDetailView(msg)
		case TemplatesViewAdd, TemplatesViewEdit:
			var cmd tea.Cmd
			m, cmd = m.updateFormView(msg)
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case TemplateSavedMsg:
		m.saved = true
		m.saveError = ""
		m.currentView = TemplatesViewList
		m.templateList = m.templates.GetAll()

	case TemplateErrorMsg:
		m.saved = false
		m.saveError = msg.Error
	}

	// Update focused input/textarea if in form view
	if m.currentView == TemplatesViewAdd || m.currentView == TemplatesViewEdit {
		if m.FocusIndex == templateBody {
			var cmd tea.Cmd
			m.bodyArea, cmd = m.bodyArea.Update(msg)
			cmds = append(cmds, cmd)
		} else if m.FocusIndex >= templateName && m.FocusIndex < templateBody {
			var cmd tea.Cmd
			m.inputs[m.FocusIndex], cmd = m.inputs[m.FocusIndex].Update(msg)
			cmds = append(cmds, cmd)
		} else if m.FocusIndex > templateBody && m.FocusIndex < templateSaveButton {
			var cmd tea.Cmd
			idx := m.FocusIndex - 1 // Adjust for templateBody being a textarea
			m.inputs[idx], cmd = m.inputs[idx].Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// updateListView handles updates for the template list view
func (m TemplatesModel) updateListView(msg tea.KeyMsg) (TemplatesModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case "down", "j":
		if m.selectedIdx < len(m.templateList)-1 {
			m.selectedIdx++
		}
	case "a", "n":
		// Add new template
		m.currentView = TemplatesViewAdd
		m.isEditing = false
		m.editingID = ""
		m.initializeForm(nil)
	case "enter":
		// View template details
		if len(m.templateList) > 0 {
			m.detailTemplate = &m.templateList[m.selectedIdx]
			m.currentView = TemplatesViewDetail
		}
	case "e":
		// Edit selected template
		if len(m.templateList) > 0 {
			template := m.templateList[m.selectedIdx]
			m.currentView = TemplatesViewEdit
			m.isEditing = true
			m.editingID = template.ID
			m.initializeForm(&template)
		}
	case "d", "x":
		// Delete selected template
		if len(m.templateList) > 0 {
			template := m.templateList[m.selectedIdx]
			if err := m.templates.Delete(template.ID); err != nil {
				m.saveError = fmt.Sprintf("Failed to delete template: %v", err)
			} else {
				m.templateList = m.templates.GetAll()
				if m.selectedIdx >= len(m.templateList) && m.selectedIdx > 0 {
					m.selectedIdx--
				}
			}
		}
	}
	return m, nil
}

// updateDetailView handles updates for the detail view
func (m TemplatesModel) updateDetailView(msg tea.KeyMsg) (TemplatesModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.currentView = TemplatesViewList
		m.detailTemplate = nil
	case "e":
		// Edit this template
		if m.detailTemplate != nil {
			template := m.templates.Get(m.detailTemplate.ID)
			if template != nil {
				m.currentView = TemplatesViewEdit
				m.isEditing = true
				m.editingID = template.ID
				m.initializeForm(template)
			}
		}
	case "d", "x":
		// Delete this template
		if m.detailTemplate != nil {
			if err := m.templates.Delete(m.detailTemplate.ID); err != nil {
				m.saveError = fmt.Sprintf("Failed to delete template: %v", err)
			} else {
				m.currentView = TemplatesViewList
				m.detailTemplate = nil
				m.templateList = m.templates.GetAll()
				if m.selectedIdx >= len(m.templateList) && m.selectedIdx > 0 {
					m.selectedIdx--
				}
			}
		}
	}
	return m, nil
}

// updateFormView handles updates for the add/edit form view
func (m TemplatesModel) updateFormView(msg tea.KeyMsg) (TemplatesModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case "esc":
		// Only cancel if not in body textarea or if textarea is not focused
		if m.FocusIndex != templateBody {
			m.currentView = TemplatesViewList
			m.saved = false
			m.saveError = ""
			return m, nil
		}

	case "tab", "shift+tab":
		m.saved = false
		m.saveError = ""

		s := msg.String()
		if s == "shift+tab" {
			m.FocusIndex--
		} else {
			m.FocusIndex++
		}

		if m.FocusIndex > templateCancelButton {
			m.FocusIndex = templateName
		} else if m.FocusIndex < templateName {
			m.FocusIndex = templateCancelButton
		}

		// Update focus
		for i := range m.inputs {
			m.inputs[i].Blur()
		}
		m.bodyArea.Blur()

		if m.FocusIndex == templateBody {
			m.bodyArea.Focus()
			cmds = append(cmds, textarea.Blink)
		} else if m.FocusIndex >= templateName && m.FocusIndex < templateBody {
			cmds = append(cmds, m.inputs[m.FocusIndex].Focus())
		} else if m.FocusIndex > templateBody && m.FocusIndex < templateSaveButton {
			idx := m.FocusIndex - 1
			cmds = append(cmds, m.inputs[idx].Focus())
		}

		return m, tea.Batch(cmds...)

	case "enter":
		// Only handle enter for buttons, not body textarea
		if m.FocusIndex == templateSaveButton {
			return m, m.saveTemplate()
		} else if m.FocusIndex == templateCancelButton {
			m.currentView = TemplatesViewList
			m.saved = false
			m.saveError = ""
		}
	}

	return m, tea.Batch(cmds...)
}

// initializeForm sets up the form inputs for add/edit
func (m *TemplatesModel) initializeForm(template *storage.Template) {
	// inputs array excludes templateBody (which is a textarea)
	// We need 4 elements: name, subject, description, tags
	m.inputs = make([]textinput.Model, 4)

	m.inputs[templateName] = createInput("Template Name", 200, 60)
	m.inputs[templateSubject] = createInput("Email Subject", 500, 60)
	// templateBody is a textarea, so it's skipped in the inputs array
	// Adjust index for description and tags since they come after body
	m.inputs[templateDescription-1] = createInput("Optional description", 500, 60)
	m.inputs[templateTags-1] = createInput("tag1, tag2 (comma-separated)", 500, 60)

	// Initialize textarea for body
	m.bodyArea = textarea.New()
	m.bodyArea.Placeholder = "Email body... (use {{variable}} syntax, e.g. {{date}}, {{from_name}}, {{from_email}})"
	m.bodyArea.SetWidth(60)
	m.bodyArea.SetHeight(10)
	m.bodyArea.CharLimit = 10000

	if template != nil {
		m.inputs[templateName].SetValue(template.Name)
		m.inputs[templateSubject].SetValue(template.Subject)
		m.bodyArea.SetValue(template.Body)
		m.inputs[templateDescription-1].SetValue(template.Description)
		if len(template.Tags) > 0 {
			m.inputs[templateTags-1].SetValue(strings.Join(template.Tags, ", "))
		}
	}

	m.FocusIndex = templateName
	m.inputs[templateName].Focus()
}

// saveTemplate saves the template
func (m *TemplatesModel) saveTemplate() tea.Cmd {
	return func() tea.Msg {
		template := storage.Template{
			Name:        m.inputs[templateName].Value(),
			Subject:     m.inputs[templateSubject].Value(),
			Body:        m.bodyArea.Value(),
			Description: m.inputs[templateDescription-1].Value(),
		}

		// Parse tags
		tagsStr := m.inputs[templateTags-1].Value()
		if tagsStr != "" {
			tags := strings.Split(tagsStr, ",")
			for i, tag := range tags {
				tags[i] = strings.TrimSpace(tag)
			}
			template.Tags = tags
		}

		// Validation
		if template.Name == "" {
			return TemplateErrorMsg{Error: "Name is required"}
		}
		if template.Subject == "" {
			return TemplateErrorMsg{Error: "Subject is required"}
		}
		if template.Body == "" {
			return TemplateErrorMsg{Error: "Body is required"}
		}

		var err error
		if m.isEditing {
			template.ID = m.editingID
			err = m.templates.Update(m.editingID, template)
		} else {
			err = m.templates.Add(template)
		}

		if err != nil {
			return TemplateErrorMsg{Error: err.Error()}
		}

		return TemplateSavedMsg{}
	}
}

// View renders the templates model
func (m TemplatesModel) View() string {
	switch m.currentView {
	case TemplatesViewList:
		return m.renderListView()
	case TemplatesViewDetail:
		return m.renderDetailView()
	case TemplatesViewAdd:
		return m.renderFormView("Add Template")
	case TemplatesViewEdit:
		return m.renderFormView("Edit Template")
	}
	return ""
}

// renderListView renders the template list
func (m TemplatesModel) renderListView() string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Templates"))
	b.WriteString("\n\n")

	if len(m.templateList) == 0 {
		b.WriteString(ui.ErrorStyle.Render("No templates saved."))
		b.WriteString("\n\n")
		b.WriteString("Press 'a' or 'n' to add a new template.\n")
	} else {
		b.WriteString(ui.SubtitleStyle.Render(fmt.Sprintf("Templates (%d)", len(m.templateList))))
		b.WriteString("\n\n")

		for i, template := range m.templateList {
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
			if len(template.Tags) > 0 {
				display += ui.LabelStyle.Render(fmt.Sprintf(" [%s]", strings.Join(template.Tags, ", ")))
			}

			b.WriteString(style.Render(prefix + display))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(ui.RenderHelp(
		"↑/↓", "navigate",
		"Enter", "view",
		"a/n", "add",
		"e", "edit",
		"d/x", "delete",
	))

	return b.String()
}

// renderDetailView renders the template detail view
func (m TemplatesModel) renderDetailView() string {
	if m.detailTemplate == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Template Details"))
	b.WriteString("\n\n")

	template := m.detailTemplate

	b.WriteString(ui.LabelStyle.Render("Name:"))
	b.WriteString("\n")
	b.WriteString(ui.DisplayLabelStyle.Render(template.Name))
	b.WriteString("\n\n")

	b.WriteString(ui.LabelStyle.Render("Subject:"))
	b.WriteString("\n")
	b.WriteString(ui.DisplayLabelStyle.Render(template.Subject))
	b.WriteString("\n\n")

	b.WriteString(ui.LabelStyle.Render("Body:"))
	b.WriteString("\n")
	b.WriteString(ui.DisplayLabelStyle.Render(template.Body))
	b.WriteString("\n\n")

	if template.Description != "" {
		b.WriteString(ui.LabelStyle.Render("Description:"))
		b.WriteString("\n")
		b.WriteString(ui.DisplayLabelStyle.Render(template.Description))
		b.WriteString("\n\n")
	}

	if len(template.Variables) > 0 {
		b.WriteString(ui.LabelStyle.Render("Variables:"))
		b.WriteString("\n")
		b.WriteString(ui.DisplayLabelStyle.Render(strings.Join(template.Variables, ", ")))
		b.WriteString("\n\n")
	}

	if len(template.Tags) > 0 {
		b.WriteString(ui.LabelStyle.Render("Tags:"))
		b.WriteString("\n")
		b.WriteString(ui.DisplayLabelStyle.Render(strings.Join(template.Tags, ", ")))
		b.WriteString("\n\n")
	}

	b.WriteString(ui.RenderHelp(
		"e", "edit",
		"d/x", "delete",
		"Esc/q", "back",
	))

	return b.String()
}

// renderFormView renders the add/edit form
func (m TemplatesModel) renderFormView(title string) string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render(title))
	b.WriteString("\n\n")

	m.renderInputField(&b, "Name", templateName)
	m.renderInputField(&b, "Subject", templateSubject)

	// Render body textarea
	focused := m.FocusIndex == templateBody
	labelStyle := ui.LabelStyle
	if focused {
		labelStyle = labelStyle.Foreground(ui.Primary)
	}
	b.WriteString(labelStyle.Render("Body:"))
	b.WriteString("\n")
	if focused {
		b.WriteString(ui.FocusedInputStyle.Render(m.bodyArea.View()))
	} else {
		b.WriteString(m.bodyArea.View())
	}
	b.WriteString("\n")

	m.renderInputField(&b, "Description", templateDescription)
	m.renderInputField(&b, "Tags", templateTags)

	b.WriteString("\n")

	// Save button
	saveText := "[ Save ]"
	if m.FocusIndex == templateSaveButton {
		b.WriteString(ui.ButtonFocusedStyle.Render(saveText))
	} else {
		b.WriteString(ui.ButtonStyle.Render(saveText))
	}

	b.WriteString("  ")

	// Cancel button
	cancelText := "[ Cancel ]"
	if m.FocusIndex == templateCancelButton {
		b.WriteString(ui.ButtonFocusedStyle.Render(cancelText))
	} else {
		b.WriteString(ui.ButtonStyle.Render(cancelText))
	}

	b.WriteString("\n")

	// Status messages
	if m.saved {
		b.WriteString("\n")
		b.WriteString(ui.SuccessStyle.Render("✓ Template saved successfully!"))
	} else if m.saveError != "" {
		b.WriteString("\n")
		b.WriteString(ui.ErrorStyle.Render("✗ Error: " + m.saveError))
	}

	b.WriteString("\n\n")
	b.WriteString(ui.RenderHelp(
		"Tab", "next field",
		"Enter", "save",
		"Esc", "cancel",
	))

	return b.String()
}

func (m *TemplatesModel) renderInputField(b *strings.Builder, label string, fieldIndex int) {
	// Adjust index for inputs array (which excludes templateBody)
	idx := fieldIndex
	if fieldIndex > templateBody {
		idx = fieldIndex - 1
	}

	if idx < 0 || idx >= len(m.inputs) {
		return
	}

	focused := m.FocusIndex == fieldIndex
	labelStyle := ui.LabelStyle
	if focused {
		labelStyle = labelStyle.Foreground(ui.Primary)
	}

	b.WriteString(labelStyle.Render(label + ":"))
	b.WriteString("\n")

	if focused {
		b.WriteString(ui.FocusedInputStyle.Render(m.inputs[idx].View()))
	} else {
		b.WriteString(m.inputs[idx].View())
	}
	b.WriteString("\n")
}

// TemplateSavedMsg signals template was saved successfully
type TemplateSavedMsg struct{}

// TemplateErrorMsg signals a template error
type TemplateErrorMsg struct {
	Error string
}
