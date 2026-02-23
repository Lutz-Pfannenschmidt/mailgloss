package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"mailgloss/storage"
	"mailgloss/ui"
)

// ContactsView represents the different views in contacts
type ContactsView int

const (
	ContactsViewList ContactsView = iota
	ContactsViewAdd
	ContactsViewEdit
	ContactsViewDetail
)

// ContactsModel represents the contacts tab
type ContactsModel struct {
	contacts    *storage.Contacts
	currentView ContactsView

	// List view
	contactList []storage.Contact
	selectedIdx int

	// Detail view
	detailContact *storage.Contact

	// Add/Edit view
	isEditing  bool
	editingID  string
	inputs     []textinput.Model
	FocusIndex int

	// Status
	saved     bool
	saveError string
	width     int
	height    int
}

const (
	contactName = iota
	contactEmail
	contactNotes
	contactTags
	contactSaveButton
	contactCancelButton
)

// NewContactsModel creates a new contacts model
func NewContactsModel(contacts *storage.Contacts) ContactsModel {
	return ContactsModel{
		contacts:    contacts,
		currentView: ContactsViewList,
		contactList: contacts.GetAll(),
		selectedIdx: 0,
	}
}

// Init initializes the contacts model
func (m ContactsModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the contacts model
func (m ContactsModel) Update(msg tea.Msg) (ContactsModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.currentView {
		case ContactsViewList:
			return m.updateListView(msg)
		case ContactsViewDetail:
			return m.updateDetailView(msg)
		case ContactsViewAdd, ContactsViewEdit:
			var cmd tea.Cmd
			m, cmd = m.updateFormView(msg)
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case ContactSavedMsg:
		m.saved = true
		m.saveError = ""
		m.currentView = ContactsViewList
		m.contactList = m.contacts.GetAll()

	case ContactErrorMsg:
		m.saved = false
		m.saveError = msg.Error
	}

	// Update focused input if in form view
	if m.currentView == ContactsViewAdd || m.currentView == ContactsViewEdit {
		if m.FocusIndex >= contactName && m.FocusIndex < contactSaveButton {
			var cmd tea.Cmd
			m.inputs[m.FocusIndex], cmd = m.inputs[m.FocusIndex].Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// updateListView handles updates for the contact list view
func (m ContactsModel) updateListView(msg tea.KeyMsg) (ContactsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case "down", "j":
		if m.selectedIdx < len(m.contactList)-1 {
			m.selectedIdx++
		}
	case "a", "n":
		// Add new contact
		m.currentView = ContactsViewAdd
		m.isEditing = false
		m.editingID = ""
		m.initializeForm(nil)
	case "enter":
		// View contact details
		if len(m.contactList) > 0 {
			m.detailContact = &m.contactList[m.selectedIdx]
			m.currentView = ContactsViewDetail
		}
	case "e":
		// Edit selected contact
		if len(m.contactList) > 0 {
			contact := m.contactList[m.selectedIdx]
			m.currentView = ContactsViewEdit
			m.isEditing = true
			m.editingID = contact.ID
			m.initializeForm(&contact)
		}
	case "d", "x":
		// Delete selected contact
		if len(m.contactList) > 0 {
			contact := m.contactList[m.selectedIdx]
			if err := m.contacts.Delete(contact.ID); err != nil {
				m.saveError = fmt.Sprintf("Failed to delete contact: %v", err)
			} else {
				m.contactList = m.contacts.GetAll()
				if m.selectedIdx >= len(m.contactList) && m.selectedIdx > 0 {
					m.selectedIdx--
				}
			}
		}
	}
	return m, nil
}

// updateDetailView handles updates for the detail view
func (m ContactsModel) updateDetailView(msg tea.KeyMsg) (ContactsModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.currentView = ContactsViewList
		m.detailContact = nil
	case "e":
		// Edit this contact
		if m.detailContact != nil {
			contact := m.contacts.Get(m.detailContact.ID)
			if contact != nil {
				m.currentView = ContactsViewEdit
				m.isEditing = true
				m.editingID = contact.ID
				m.initializeForm(contact)
			}
		}
	case "d", "x":
		// Delete this contact
		if m.detailContact != nil {
			if err := m.contacts.Delete(m.detailContact.ID); err != nil {
				m.saveError = fmt.Sprintf("Failed to delete contact: %v", err)
			} else {
				m.currentView = ContactsViewList
				m.detailContact = nil
				m.contactList = m.contacts.GetAll()
				if m.selectedIdx >= len(m.contactList) && m.selectedIdx > 0 {
					m.selectedIdx--
				}
			}
		}
	}
	return m, nil
}

// updateFormView handles updates for the add/edit form view
func (m ContactsModel) updateFormView(msg tea.KeyMsg) (ContactsModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case "esc":
		m.currentView = ContactsViewList
		m.saved = false
		m.saveError = ""
		return m, nil

	case "tab", "shift+tab", "up", "down":
		m.saved = false
		m.saveError = ""

		s := msg.String()
		if s == "up" || s == "shift+tab" {
			m.FocusIndex--
		} else {
			m.FocusIndex++
		}

		if m.FocusIndex > contactCancelButton {
			m.FocusIndex = contactName
		} else if m.FocusIndex < contactName {
			m.FocusIndex = contactCancelButton
		}

		// Update focus
		for i := range m.inputs {
			m.inputs[i].Blur()
		}

		if m.FocusIndex >= contactName && m.FocusIndex < contactSaveButton {
			cmds = append(cmds, m.inputs[m.FocusIndex].Focus())
		}

		return m, tea.Batch(cmds...)

	case "enter":
		if m.FocusIndex == contactSaveButton {
			return m, m.saveContact()
		} else if m.FocusIndex == contactCancelButton {
			m.currentView = ContactsViewList
			m.saved = false
			m.saveError = ""
		}
	}

	return m, tea.Batch(cmds...)
}

// initializeForm sets up the form inputs for add/edit
func (m *ContactsModel) initializeForm(contact *storage.Contact) {
	m.inputs = make([]textinput.Model, contactSaveButton)

	m.inputs[contactName] = createInput("John Doe", 200, 60)
	m.inputs[contactEmail] = createInput("john@example.com", 200, 60)
	m.inputs[contactNotes] = createInput("Optional notes", 500, 60)
	m.inputs[contactTags] = createInput("work, client (comma-separated)", 500, 60)

	if contact != nil {
		m.inputs[contactName].SetValue(contact.Name)
		m.inputs[contactEmail].SetValue(contact.Email)
		m.inputs[contactNotes].SetValue(contact.Notes)
		if len(contact.Tags) > 0 {
			m.inputs[contactTags].SetValue(strings.Join(contact.Tags, ", "))
		}
	}

	m.FocusIndex = contactName
	m.inputs[contactName].Focus()
}

// saveContact saves the contact
func (m *ContactsModel) saveContact() tea.Cmd {
	return func() tea.Msg {
		contact := storage.Contact{
			Name:  m.inputs[contactName].Value(),
			Email: m.inputs[contactEmail].Value(),
			Notes: m.inputs[contactNotes].Value(),
		}

		// Parse tags
		tagsStr := m.inputs[contactTags].Value()
		if tagsStr != "" {
			tags := strings.Split(tagsStr, ",")
			for i, tag := range tags {
				tags[i] = strings.TrimSpace(tag)
			}
			contact.Tags = tags
		}

		// Validation
		if contact.Name == "" {
			return ContactErrorMsg{Error: "Name is required"}
		}
		if contact.Email == "" {
			return ContactErrorMsg{Error: "Email is required"}
		}

		var err error
		if m.isEditing {
			contact.ID = m.editingID
			err = m.contacts.Update(m.editingID, contact)
		} else {
			err = m.contacts.Add(contact)
		}

		if err != nil {
			return ContactErrorMsg{Error: err.Error()}
		}

		return ContactSavedMsg{}
	}
}

// View renders the contacts model
func (m ContactsModel) View() string {
	switch m.currentView {
	case ContactsViewList:
		return m.renderListView()
	case ContactsViewDetail:
		return m.renderDetailView()
	case ContactsViewAdd:
		return m.renderFormView("Add Contact")
	case ContactsViewEdit:
		return m.renderFormView("Edit Contact")
	}
	return ""
}

// renderListView renders the contact list
func (m ContactsModel) renderListView() string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Contacts"))
	b.WriteString("\n\n")

	if len(m.contactList) == 0 {
		b.WriteString(ui.ErrorStyle.Render("No contacts saved."))
		b.WriteString("\n\n")
		b.WriteString("Press 'a' or 'n' to add a new contact.\n")
	} else {
		b.WriteString(ui.SubtitleStyle.Render(fmt.Sprintf("Contacts (%d)", len(m.contactList))))
		b.WriteString("\n\n")

		for i, contact := range m.contactList {
			prefix := "  "
			style := ui.DisplayLabelStyle
			if i == m.selectedIdx {
				prefix = "▸ "
				style = style.Foreground(ui.Primary)
			}

			display := fmt.Sprintf("%s <%s>", contact.Name, contact.Email)
			if len(contact.Tags) > 0 {
				display += ui.LabelStyle.Render(fmt.Sprintf(" [%s]", strings.Join(contact.Tags, ", ")))
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

// renderDetailView renders the contact detail view
func (m ContactsModel) renderDetailView() string {
	if m.detailContact == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Contact Details"))
	b.WriteString("\n\n")

	contact := m.detailContact

	b.WriteString(ui.LabelStyle.Render("Name:"))
	b.WriteString("\n")
	b.WriteString(ui.DisplayLabelStyle.Render(contact.Name))
	b.WriteString("\n\n")

	b.WriteString(ui.LabelStyle.Render("Email:"))
	b.WriteString("\n")
	b.WriteString(ui.DisplayLabelStyle.Render(contact.Email))
	b.WriteString("\n\n")

	if contact.Notes != "" {
		b.WriteString(ui.LabelStyle.Render("Notes:"))
		b.WriteString("\n")
		b.WriteString(ui.DisplayLabelStyle.Render(contact.Notes))
		b.WriteString("\n\n")
	}

	if len(contact.Tags) > 0 {
		b.WriteString(ui.LabelStyle.Render("Tags:"))
		b.WriteString("\n")
		b.WriteString(ui.DisplayLabelStyle.Render(strings.Join(contact.Tags, ", ")))
		b.WriteString("\n\n")
	}

	b.WriteString(ui.LabelStyle.Render("Created:"))
	b.WriteString("\n")
	b.WriteString(ui.DisplayLabelStyle.Render(contact.CreatedAt.Format("2006-01-02 15:04:05")))
	b.WriteString("\n\n")

	b.WriteString(ui.LabelStyle.Render("Updated:"))
	b.WriteString("\n")
	b.WriteString(ui.DisplayLabelStyle.Render(contact.UpdatedAt.Format("2006-01-02 15:04:05")))
	b.WriteString("\n\n")

	b.WriteString(ui.RenderHelp(
		"e", "edit",
		"d/x", "delete",
		"Esc/q", "back",
	))

	return b.String()
}

// renderFormView renders the add/edit form
func (m ContactsModel) renderFormView(title string) string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render(title))
	b.WriteString("\n\n")

	m.renderField(&b, "Name", contactName)
	m.renderField(&b, "Email", contactEmail)
	m.renderField(&b, "Notes", contactNotes)
	m.renderField(&b, "Tags", contactTags)

	b.WriteString("\n")

	// Save button
	saveText := "[ Save ]"
	if m.FocusIndex == contactSaveButton {
		b.WriteString(ui.ButtonFocusedStyle.Render(saveText))
	} else {
		b.WriteString(ui.ButtonStyle.Render(saveText))
	}

	b.WriteString("  ")

	// Cancel button
	cancelText := "[ Cancel ]"
	if m.FocusIndex == contactCancelButton {
		b.WriteString(ui.ButtonFocusedStyle.Render(cancelText))
	} else {
		b.WriteString(ui.ButtonStyle.Render(cancelText))
	}

	b.WriteString("\n")

	// Status messages
	if m.saved {
		b.WriteString("\n")
		b.WriteString(ui.SuccessStyle.Render("✓ Contact saved successfully!"))
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

func (m *ContactsModel) renderField(b *strings.Builder, label string, fieldIndex int) {
	if fieldIndex < 0 || fieldIndex >= len(m.inputs) {
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
		b.WriteString(ui.FocusedInputStyle.Render(m.inputs[fieldIndex].View()))
	} else {
		b.WriteString(m.inputs[fieldIndex].View())
	}
	b.WriteString("\n")
}

// ContactSavedMsg signals contact was saved successfully
type ContactSavedMsg struct{}

// ContactErrorMsg signals a contact error
type ContactErrorMsg struct {
	Error string
}
