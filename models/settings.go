package models

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"mailgloss/config"
	"mailgloss/ui"
)

// SettingsView represents the different views in settings
type SettingsView int

const (
	SettingsViewList SettingsView = iota
	SettingsViewApp
	SettingsViewAdd
	SettingsViewEdit
)

// SettingsModel represents the settings tab
type SettingsModel struct {
	config      *config.Config
	currentView SettingsView

	// List view
	providerList []string
	selectedIdx  int
	appSelected  bool

	// Add/Edit view
	isEditing       bool
	editingName     string // Name of provider being edited
	inputs          []textinput.Model
	FocusIndex      int
	providerTypes   []config.Provider
	providerTypeIdx int

	// App-level settings form (separate from provider inputs)
	appInputs     []textinput.Model
	appFocusIndex int

	// Status
	saved     bool
	saveError string
	width     int
	height    int
}

const (
	settingsProviderType = iota // Only used in add mode
	settingsName
	settingsFromAddress
	settingsFromName
	settingsDateFormat
	// Mailgun fields
	settingsMailgunAPIKey
	settingsMailgunDomain
	settingsMailgunURL
	// SMTP fields
	settingsSMTPHost
	settingsSMTPPort
	settingsSMTPUsername
	settingsSMTPPassword
	// SendGrid fields
	settingsSendGridAPIKey
	// Postmark fields
	settingsPostmarkAPIKey
	// SparkPost fields
	settingsSparkPostAPIKey
	settingsSparkPostURL
	// Postal fields
	settingsPostalURL
	settingsPostalAPIKey
	// Actions
	settingsSaveButton
	settingsCancelButton
)

// NewSettingsModel creates a new settings model
func NewSettingsModel(cfg *config.Config) SettingsModel {
	m := SettingsModel{
		config:       cfg,
		currentView:  SettingsViewList,
		providerList: nil,
		selectedIdx:  0,
		appSelected:  true,
		providerTypes: []config.Provider{
			config.ProviderMailgun,
			config.ProviderSMTP,
			config.ProviderSendGrid,
			config.ProviderPostmark,
			config.ProviderSparkPost,
			config.ProviderPostal,
		},
	}
	m.refreshProviderList()
	return m
}

// refreshProviderList reloads providerList
func (m *SettingsModel) refreshProviderList() {
	providers := m.config.ListProviders()
	m.providerList = make([]string, 0, len(providers))
	m.providerList = append(m.providerList, providers...)
}

// Init initializes the settings model
func (m SettingsModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the settings model
func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.currentView {
		case SettingsViewList:
			return m.updateListView(msg)
		case SettingsViewAdd, SettingsViewEdit:
			var cmd tea.Cmd
			m, cmd = m.updateFormView(msg)
			cmds = append(cmds, cmd)
			// Don't return here - let input update below happen
		case SettingsViewApp:
			var cmd tea.Cmd
			m, cmd = m.updateAppView(msg)
			cmds = append(cmds, cmd)
			// Don't return here - let input update below happen
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case ConfigSavedMsg:
		m.saved = true
		m.saveError = ""
		// Return to list view
		m.currentView = SettingsViewList
		m.providerList = m.config.ListProviders()
		// If there are providers, select first provider; otherwise focus App Settings
		if len(m.providerList) > 0 {
			m.appSelected = false
			m.selectedIdx = 0
		} else {
			m.appSelected = true
			m.selectedIdx = 0
		}

	case ConfigErrorMsg:
		m.saved = false
		m.saveError = msg.Error
	}

	// Update focused input if in form view
	if m.currentView == SettingsViewAdd || m.currentView == SettingsViewEdit {
		if m.FocusIndex >= settingsName && m.FocusIndex < settingsSaveButton {
			var cmd tea.Cmd
			m.inputs[m.FocusIndex-1], cmd = m.inputs[m.FocusIndex-1].Update(msg)
			cmds = append(cmds, cmd)
		}
	} else if m.currentView == SettingsViewApp {
		if m.appFocusIndex >= 0 && m.appFocusIndex < len(m.appInputs) {
			var cmd tea.Cmd
			m.appInputs[m.appFocusIndex], cmd = m.appInputs[m.appFocusIndex].Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// updateListView handles updates for the provider list view
func (m SettingsModel) updateListView(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		// Move focus up: from provider list to App Settings, or wrap within providers
		if m.appSelected {
			// already on App Settings - wrap to last provider if any
			if len(m.providerList) > 0 {
				m.appSelected = false
				m.selectedIdx = len(m.providerList) - 1
			}
		} else {
			if m.selectedIdx > 0 {
				m.selectedIdx--
			} else {
				// move focus to App Settings
				m.appSelected = true
			}
		}
	case "down", "j":
		// Move focus down: from App Settings into provider list, or advance within providers
		if m.appSelected {
			if len(m.providerList) > 0 {
				m.appSelected = false
				m.selectedIdx = 0
			}
		} else {
			if m.selectedIdx < len(m.providerList)-1 {
				m.selectedIdx++
			} else {
				// wrap to App Settings
				m.appSelected = true
			}
		}
	case "a", "n":
		// Add new provider
		m.currentView = SettingsViewAdd
		m.isEditing = false
		m.editingName = ""
		m.initializeForm(nil)
	case "e", "enter":
		// Edit selected provider
		// If App Settings selected, open app form
		if m.appSelected {
			m.currentView = SettingsViewApp
			// initialize app inputs
			m.appInputs = make([]textinput.Model, 1)
			m.appInputs[0] = createInput("02.01.2006", 50, 60)
			if m.config != nil && m.config.DateFormat != "" {
				m.appInputs[0].SetValue(m.config.DateFormat)
			}
			m.appFocusIndex = 0
			m.appInputs[0].Focus()
			return m, nil
		}

		if len(m.providerList) > 0 {
			providerName := m.providerList[m.selectedIdx]
			if pc, err := m.config.GetProvider(providerName); err == nil {
				m.currentView = SettingsViewEdit
				m.isEditing = true
				m.editingName = providerName
				m.initializeForm(pc)
			}
		}
	case "d", "x":
		// Delete selected provider
		// Only allow delete when a provider is focused
		if !m.appSelected && len(m.providerList) > 0 {
			providerName := m.providerList[m.selectedIdx]
			if err := m.config.DeleteProvider(providerName); err == nil {
				if err := m.config.Save(); err != nil {
					// Handle save error
					m.saveError = fmt.Sprintf("Failed to save config: %v", err)
				} else {
					m.providerList = m.config.ListProviders()
					if len(m.providerList) == 0 {
						m.selectedIdx = 0
						m.appSelected = true
					} else if m.selectedIdx >= len(m.providerList) && m.selectedIdx > 0 {
						m.selectedIdx--
					}
				}
			}
		}
	case "s":
		// Set as default
		if !m.appSelected && len(m.providerList) > 0 {
			m.config.DefaultProvider = m.providerList[m.selectedIdx]
			if err := m.config.Save(); err != nil {
				m.saveError = fmt.Sprintf("Failed to save config: %v", err)
			}
		}
	}
	return m, nil
}

// updateFormView handles updates for the add/edit form view
func (m SettingsModel) updateFormView(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case "esc":
		// Cancel and return to list
		m.currentView = SettingsViewList
		m.saved = false
		m.saveError = ""
		return m, nil

	case "tab", "shift+tab", "up", "down":
		m.saved = false
		m.saveError = ""

		s := msg.String()
		if s == "up" || s == "shift+tab" {
			m.FocusIndex--
			// Skip fields that aren't part of current provider
			for !m.isValidField(m.FocusIndex) && m.FocusIndex >= settingsProviderType {
				m.FocusIndex--
			}
		} else {
			m.FocusIndex++
			// Skip fields that aren't part of current provider
			for !m.isValidField(m.FocusIndex) && m.FocusIndex <= settingsCancelButton {
				m.FocusIndex++
			}
		}

		// Determine focus range based on mode
		minFocus := settingsProviderType
		if m.isEditing {
			minFocus = settingsName // Skip provider type in edit mode
		}
		maxFocus := settingsCancelButton

		if m.FocusIndex > maxFocus {
			m.FocusIndex = minFocus
			// Make sure we land on a valid field
			for !m.isValidField(m.FocusIndex) && m.FocusIndex <= maxFocus {
				m.FocusIndex++
			}
		} else if m.FocusIndex < minFocus {
			m.FocusIndex = maxFocus
		}

		// Update focus
		for i := range m.inputs {
			m.inputs[i].Blur()
		}

		// Only focus inputs, not the provider type selector
		if m.FocusIndex >= settingsName && m.FocusIndex < settingsSaveButton {
			cmds = append(cmds, m.inputs[m.FocusIndex-1].Focus()) // -1 because inputs array doesn't include settingsProviderType
		}

		return m, tea.Batch(cmds...)

	case "left", "right":
		// Change provider type (only when on the type selector in add mode)
		if !m.isEditing && m.FocusIndex == settingsProviderType {
			if msg.String() == "left" {
				m.providerTypeIdx--
				if m.providerTypeIdx < 0 {
					m.providerTypeIdx = len(m.providerTypes) - 1
				}
			} else {
				m.providerTypeIdx++
				if m.providerTypeIdx >= len(m.providerTypes) {
					m.providerTypeIdx = 0
				}
			}
		}

	case "enter":
		if m.FocusIndex == settingsSaveButton {
			return m, m.saveProviderConfig()
		} else if m.FocusIndex == settingsCancelButton {
			m.currentView = SettingsViewList
			m.saved = false
			m.saveError = ""
		}
	}

	return m, tea.Batch(cmds...)
}

// updateAppView handles the app-level settings form input/navigation
func (m SettingsModel) updateAppView(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case "esc":
		m.currentView = SettingsViewList
		return m, nil

	case "tab", "shift+tab", "up", "down":
		// We have 3 focusable positions in App view: 0=input, 1=Save, 2=Cancel
		s := msg.String()
		if s == "up" || s == "shift+tab" {
			m.appFocusIndex--
		} else {
			m.appFocusIndex++
		}

		// wrap between 0 and 2
		if m.appFocusIndex > 2 {
			m.appFocusIndex = 0
		} else if m.appFocusIndex < 0 {
			m.appFocusIndex = 2
		}

		// Update input focus only when the input field is selected
		for i := range m.appInputs {
			m.appInputs[i].Blur()
		}
		if m.appFocusIndex == 0 && len(m.appInputs) > 0 {
			cmds = append(cmds, m.appInputs[0].Focus())
		}
		return m, tea.Batch(cmds...)

	case "enter":
		// Enter behavior depends on focused control: Save (1), Cancel (2), or Save from input (0)
		if m.appFocusIndex == 2 {
			// Cancel
			m.currentView = SettingsViewList
			return m, nil
		}

		// Save (either input focused or Save button)
		dateFmt := ""
		if len(m.appInputs) > 0 {
			dateFmt = m.appInputs[0].Value()
		}
		if dateFmt != "" {
			m.config.DateFormat = dateFmt
			if err := m.config.Save(); err != nil {
				return m, func() tea.Msg { return ConfigErrorMsg{Error: err.Error()} }
			}
		}
		return m, func() tea.Msg { return ConfigSavedMsg{} }
	}

	return m, nil
}

// initializeForm sets up the form inputs for add/edit
func (m *SettingsModel) initializeForm(pc *config.ProviderConfig) {
	// Create all possible inputs (excluding settingsProviderType which is not an input)
	m.inputs = make([]textinput.Model, settingsSaveButton-1)

	// Name field
	m.inputs[settingsName-1] = createInput("my-provider", 100, 60)
	if pc != nil {
		m.inputs[settingsName-1].SetValue(pc.Name)
		m.inputs[settingsName-1].Blur() // Can't edit name when editing
	}

	// Common fields
	m.inputs[settingsFromAddress-1] = createInput("email@example.com", 200, 60)
	m.inputs[settingsFromName-1] = createInput("Your Name", 200, 60)
	m.inputs[settingsDateFormat-1] = createInput("02.01.2006", 50, 60)

	if pc != nil {
		m.inputs[settingsFromAddress-1].SetValue(pc.FromAddress)
		m.inputs[settingsFromName-1].SetValue(pc.FromName)
		// if config has date format set globally, use that for provider edit mode
		if m.config != nil && m.config.DateFormat != "" {
			m.inputs[settingsDateFormat-1].SetValue(m.config.DateFormat)
		}
	}

	// Mailgun fields
	m.inputs[settingsMailgunAPIKey-1] = createInput("Mailgun API Key", 500, 60)
	m.inputs[settingsMailgunAPIKey-1].EchoMode = textinput.EchoPassword
	m.inputs[settingsMailgunDomain-1] = createInput("your-domain.com", 200, 60)
	m.inputs[settingsMailgunURL-1] = createInput("https://api.mailgun.net", 500, 60)

	// SMTP fields
	m.inputs[settingsSMTPHost-1] = createInput("smtp.gmail.com", 500, 60)
	m.inputs[settingsSMTPPort-1] = createInput("587", 10, 60)
	m.inputs[settingsSMTPUsername-1] = createInput("username", 500, 60)
	m.inputs[settingsSMTPPassword-1] = createInput("password", 500, 60)
	m.inputs[settingsSMTPPassword-1].EchoMode = textinput.EchoPassword

	// SendGrid fields
	m.inputs[settingsSendGridAPIKey-1] = createInput("SendGrid API Key", 500, 60)
	m.inputs[settingsSendGridAPIKey-1].EchoMode = textinput.EchoPassword

	// Postmark fields
	m.inputs[settingsPostmarkAPIKey-1] = createInput("Postmark API Key", 500, 60)
	m.inputs[settingsPostmarkAPIKey-1].EchoMode = textinput.EchoPassword

	// SparkPost fields
	m.inputs[settingsSparkPostAPIKey-1] = createInput("SparkPost API Key", 500, 60)
	m.inputs[settingsSparkPostAPIKey-1].EchoMode = textinput.EchoPassword
	m.inputs[settingsSparkPostURL-1] = createInput("https://api.sparkpost.com", 500, 60)

	// Postal fields
	m.inputs[settingsPostalURL-1] = createInput("https://postal.example.com", 500, 60)
	m.inputs[settingsPostalAPIKey-1] = createInput("Postal API Key", 500, 60)
	m.inputs[settingsPostalAPIKey-1].EchoMode = textinput.EchoPassword

	// If editing, populate provider-specific fields
	if pc != nil {
		m.providerTypeIdx = m.getProviderTypeIndex(pc.Type)

		switch pc.Type {
		case config.ProviderMailgun:
			if pc.Mailgun != nil {
				m.inputs[settingsMailgunAPIKey-1].SetValue(pc.Mailgun.APIKey)
				m.inputs[settingsMailgunDomain-1].SetValue(pc.Mailgun.Domain)
				m.inputs[settingsMailgunURL-1].SetValue(pc.Mailgun.URL)
			}
		case config.ProviderSMTP:
			if pc.SMTP != nil {
				m.inputs[settingsSMTPHost-1].SetValue(pc.SMTP.Host)
				m.inputs[settingsSMTPPort-1].SetValue(fmt.Sprintf("%d", pc.SMTP.Port))
				m.inputs[settingsSMTPUsername-1].SetValue(pc.SMTP.Username)
				m.inputs[settingsSMTPPassword-1].SetValue(pc.SMTP.Password)
			}
		case config.ProviderSendGrid:
			if pc.SendGrid != nil {
				m.inputs[settingsSendGridAPIKey-1].SetValue(pc.SendGrid.APIKey)
			}
		case config.ProviderPostmark:
			if pc.Postmark != nil {
				m.inputs[settingsPostmarkAPIKey-1].SetValue(pc.Postmark.APIKey)
			}
		case config.ProviderSparkPost:
			if pc.SparkPost != nil {
				m.inputs[settingsSparkPostAPIKey-1].SetValue(pc.SparkPost.APIKey)
				m.inputs[settingsSparkPostURL-1].SetValue(pc.SparkPost.URL)
			}
		case config.ProviderPostal:
			if pc.Postal != nil {
				m.inputs[settingsPostalURL-1].SetValue(pc.Postal.URL)
				m.inputs[settingsPostalAPIKey-1].SetValue(pc.Postal.APIKey)
			}
		}
	}

	// Focus first field - provider type in add mode, name in edit mode
	if m.isEditing {
		m.FocusIndex = settingsName
		m.inputs[settingsName-1].Focus()
	} else {
		// In add mode, start at name field (first input)
		m.FocusIndex = settingsName
		m.inputs[settingsName-1].Focus()
	}
}

func (m *SettingsModel) getProviderTypeIndex(pt config.Provider) int {
	for i, p := range m.providerTypes {
		if p == pt {
			return i
		}
	}
	return 0
}

// isValidField checks if a field index is valid for the current provider type
func (m *SettingsModel) isValidField(fieldIndex int) bool {
	// Provider type selector (add mode only)
	if fieldIndex == settingsProviderType {
		return !m.isEditing
	}

	// Common fields
	if fieldIndex >= settingsName && fieldIndex <= settingsFromName {
		return true
	}

	// Buttons
	if fieldIndex == settingsSaveButton || fieldIndex == settingsCancelButton {
		return true
	}

	// Provider-specific fields
	currentProvider := m.providerTypes[m.providerTypeIdx]
	switch currentProvider {
	case config.ProviderMailgun:
		return fieldIndex >= settingsMailgunAPIKey && fieldIndex <= settingsMailgunURL
	case config.ProviderSMTP:
		return fieldIndex >= settingsSMTPHost && fieldIndex <= settingsSMTPPassword
	case config.ProviderSendGrid:
		return fieldIndex == settingsSendGridAPIKey
	case config.ProviderPostmark:
		return fieldIndex == settingsPostmarkAPIKey
	case config.ProviderSparkPost:
		return fieldIndex >= settingsSparkPostAPIKey && fieldIndex <= settingsSparkPostURL
	case config.ProviderPostal:
		return fieldIndex >= settingsPostalURL && fieldIndex <= settingsPostalAPIKey
	}

	return false
}

// View renders the settings model
func (m SettingsModel) View() string {
	switch m.currentView {
	case SettingsViewList:
		return m.renderListView()
	case SettingsViewApp:
		return m.renderAppView()
	case SettingsViewAdd:
		return m.renderFormView("Add Provider")
	case SettingsViewEdit:
		return m.renderFormView("Edit Provider: " + m.editingName)
	}
	return ""
}

// renderAppView renders the app-level settings form
func (m SettingsModel) renderAppView() string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Settings - App Settings"))
	b.WriteString("\n\n")

	labelStyle := ui.LabelStyle
	if m.appFocusIndex == 0 {
		labelStyle = labelStyle.Foreground(ui.Primary)
	}
	b.WriteString(labelStyle.Render("Date Format (Go layout, e.g. 02.01.2006):"))
	b.WriteString("\n")

	if len(m.appInputs) > 0 {
		if m.appFocusIndex == 0 {
			b.WriteString(ui.FocusedInputStyle.Render(m.appInputs[0].View()))
		} else {
			b.WriteString(m.appInputs[0].View())
		}
	}
	b.WriteString("\n\n")

	saveText := "[ Save ]"
	if m.appFocusIndex == 1 {
		b.WriteString(ui.ButtonFocusedStyle.Render(saveText))
	} else {
		b.WriteString(ui.ButtonStyle.Render(saveText))
	}
	b.WriteString("  ")
	cancelText := "[ Cancel ]"
	if m.appFocusIndex == 2 {
		b.WriteString(ui.ButtonFocusedStyle.Render(cancelText))
	} else {
		b.WriteString(ui.ButtonStyle.Render(cancelText))
	}

	b.WriteString("\n\n")
	b.WriteString(ui.RenderHelp(
		"Tab", "next field",
		"Enter", "save",
		"Esc", "cancel",
	))

	return b.String()
}

// renderListView renders the provider list
func (m SettingsModel) renderListView() string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Settings"))
	b.WriteString("\n\n")

	// App Settings control (separate from provider list)
	appLabel := "[ App Settings ]"
	if m.appSelected {
		b.WriteString(ui.ButtonFocusedStyle.Render(appLabel))
	} else {
		b.WriteString(ui.ButtonStyle.Render(appLabel))
	}
	b.WriteString("\n\n")

	if len(m.providerList) == 0 {
		b.WriteString(ui.ErrorStyle.Render("No providers configured."))
		b.WriteString("\n\n")
		b.WriteString("Press 'a' or 'n' to add a new provider.\n")
	} else {
		b.WriteString(ui.SubtitleStyle.Render(fmt.Sprintf("Providers (%d)", len(m.providerList))))
		b.WriteString("\n\n")

		for i, name := range m.providerList {
			prefix := "  "
			style := ui.DisplayLabelStyle
			if !m.appSelected && i == m.selectedIdx {
				prefix = "▸ "
				style = style.Foreground(ui.Primary)
			}

			display := name
			if name == m.config.DefaultProvider {
				display += ui.SuccessStyle.Render(" (default)")
			}

			b.WriteString(style.Render(prefix + display))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(ui.RenderHelp(
		"↑/↓", "navigate",
		"a/n", "add",
		"e/Enter", "edit",
		"d/x", "delete",
		"s", "set default",
	))

	return b.String()
}

// renderFormView renders the add/edit form
func (m SettingsModel) renderFormView(title string) string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render(title))
	b.WriteString("\n\n")

	// Provider type selector (only in add mode)
	if !m.isEditing {
		focused := m.FocusIndex == settingsProviderType
		label := ui.LabelStyle
		if focused {
			label = label.Foreground(ui.Primary)
		}
		b.WriteString(label.Render("Provider Type:"))
		b.WriteString("\n")

		typeDisplay := string(m.providerTypes[m.providerTypeIdx])
		if focused {
			typeDisplay = "< " + typeDisplay + " >"
			b.WriteString(ui.FocusedInputStyle.Render(typeDisplay))
		} else {
			b.WriteString(typeDisplay)
		}
		b.WriteString("\n\n")
	}

	// Name field
	m.renderField(&b, "Name", settingsName, !m.isEditing)

	// Common fields
	m.renderField(&b, "From Address", settingsFromAddress, true)
	m.renderField(&b, "From Name", settingsFromName, true)
	m.renderField(&b, "Date Format", settingsDateFormat, true)
	b.WriteString("\n")

	// Provider-specific fields
	currentProvider := m.providerTypes[m.providerTypeIdx]

	switch currentProvider {
	case config.ProviderMailgun:
		b.WriteString(ui.SubtitleStyle.Render("Mailgun Configuration"))
		b.WriteString("\n")
		m.renderField(&b, "API Key", settingsMailgunAPIKey, true)
		m.renderField(&b, "Domain", settingsMailgunDomain, true)
		m.renderField(&b, "API URL", settingsMailgunURL, true)

	case config.ProviderSMTP:
		b.WriteString(ui.SubtitleStyle.Render("SMTP Configuration"))
		b.WriteString("\n")
		m.renderField(&b, "Host", settingsSMTPHost, true)
		m.renderField(&b, "Port", settingsSMTPPort, true)
		m.renderField(&b, "Username", settingsSMTPUsername, true)
		m.renderField(&b, "Password", settingsSMTPPassword, true)

	case config.ProviderSendGrid:
		b.WriteString(ui.SubtitleStyle.Render("SendGrid Configuration"))
		b.WriteString("\n")
		m.renderField(&b, "API Key", settingsSendGridAPIKey, true)

	case config.ProviderPostmark:
		b.WriteString(ui.SubtitleStyle.Render("Postmark Configuration"))
		b.WriteString("\n")
		m.renderField(&b, "API Key", settingsPostmarkAPIKey, true)

	case config.ProviderSparkPost:
		b.WriteString(ui.SubtitleStyle.Render("SparkPost Configuration"))
		b.WriteString("\n")
		m.renderField(&b, "API Key", settingsSparkPostAPIKey, true)
		m.renderField(&b, "API URL", settingsSparkPostURL, true)

	case config.ProviderPostal:
		b.WriteString(ui.SubtitleStyle.Render("Postal Configuration"))
		b.WriteString("\n")
		m.renderField(&b, "URL", settingsPostalURL, true)
		m.renderField(&b, "API Key", settingsPostalAPIKey, true)
	}

	b.WriteString("\n")

	// Save button
	saveText := "[ Save ]"
	if m.FocusIndex == settingsSaveButton {
		b.WriteString(ui.ButtonFocusedStyle.Render(saveText))
	} else {
		b.WriteString(ui.ButtonStyle.Render(saveText))
	}

	b.WriteString("  ")

	// Cancel button
	cancelText := "[ Cancel ]"
	if m.FocusIndex == settingsCancelButton {
		b.WriteString(ui.ButtonFocusedStyle.Render(cancelText))
	} else {
		b.WriteString(ui.ButtonStyle.Render(cancelText))
	}

	b.WriteString("\n")

	// Status messages
	if m.saved {
		b.WriteString("\n")
		b.WriteString(ui.SuccessStyle.Render("✓ Provider saved successfully!"))
	} else if m.saveError != "" {
		b.WriteString("\n")
		b.WriteString(ui.ErrorStyle.Render("✗ Error: " + m.saveError))
	}

	b.WriteString("\n\n")
	b.WriteString(ui.RenderHelp(
		"Tab", "next field",
		"←/→", "change type",
		"Enter", "save",
		"Esc", "cancel",
	))

	return b.String()
}

func (m *SettingsModel) renderField(b *strings.Builder, label string, fieldIndex int, enabled bool) {
	actualIndex := fieldIndex - 1 // Adjust for settingsProviderType offset
	if actualIndex < 0 || actualIndex >= len(m.inputs) {
		return
	}

	focused := m.FocusIndex == fieldIndex && enabled
	labelStyle := ui.LabelStyle
	if focused {
		labelStyle = labelStyle.Foreground(ui.Primary)
	}

	b.WriteString(labelStyle.Render(label + ":"))
	b.WriteString("\n")

	if !enabled {
		b.WriteString(ui.LabelStyle.Render(m.inputs[actualIndex].Value()))
	} else if focused {
		b.WriteString(ui.FocusedInputStyle.Render(m.inputs[actualIndex].View()))
	} else {
		b.WriteString(m.inputs[actualIndex].View())
	}
	b.WriteString("\n")
}

func (m *SettingsModel) saveProviderConfig() tea.Cmd {
	return func() tea.Msg {
		// Build provider config from inputs (subtract 1 for settingsProviderType offset)
		pc := &config.ProviderConfig{
			Name:        m.inputs[settingsName-1].Value(),
			Type:        m.providerTypes[m.providerTypeIdx],
			FromAddress: m.inputs[settingsFromAddress-1].Value(),
			FromName:    m.inputs[settingsFromName-1].Value(),
		}

		// If editing, delete the old entry first (in case name changed)
		if m.isEditing && m.editingName != pc.Name {
			m.config.DeleteProvider(m.editingName)
		}

		// Set provider-specific config
		switch pc.Type {
		case config.ProviderMailgun:
			pc.Mailgun = &config.MailgunConfig{
				APIKey: m.inputs[settingsMailgunAPIKey-1].Value(),
				Domain: m.inputs[settingsMailgunDomain-1].Value(),
				URL:    m.inputs[settingsMailgunURL-1].Value(),
			}

		case config.ProviderSMTP:
			port := 587
			portStr := m.inputs[settingsSMTPPort-1].Value()
			if portStr != "" {
				parsedPort, err := strconv.Atoi(portStr)
				if err != nil {
					return ConfigErrorMsg{Error: fmt.Sprintf("invalid port number: %s", portStr)}
				}
				if parsedPort < 1 || parsedPort > 65535 {
					return ConfigErrorMsg{Error: fmt.Sprintf("port must be between 1 and 65535, got %d", parsedPort)}
				}
				port = parsedPort
			}
			pc.SMTP = &config.SMTPConfig{
				Host:     m.inputs[settingsSMTPHost-1].Value(),
				Port:     port,
				Username: m.inputs[settingsSMTPUsername-1].Value(),
				Password: m.inputs[settingsSMTPPassword-1].Value(),
			}

		case config.ProviderSendGrid:
			pc.SendGrid = &config.SendGridConfig{
				APIKey: m.inputs[settingsSendGridAPIKey-1].Value(),
			}

		case config.ProviderPostmark:
			pc.Postmark = &config.PostmarkConfig{
				APIKey: m.inputs[settingsPostmarkAPIKey-1].Value(),
			}

		case config.ProviderSparkPost:
			pc.SparkPost = &config.SparkPostConfig{
				APIKey: m.inputs[settingsSparkPostAPIKey-1].Value(),
				URL:    m.inputs[settingsSparkPostURL-1].Value(),
			}

		case config.ProviderPostal:
			pc.Postal = &config.PostalConfig{
				URL:    m.inputs[settingsPostalURL-1].Value(),
				APIKey: m.inputs[settingsPostalAPIKey-1].Value(),
			}
		}

		// Add to config
		if err := m.config.AddProvider(pc); err != nil {
			return ConfigErrorMsg{Error: err.Error()}
		}

		// Save config
		// Save any app-level settings: date format
		dateFmt := m.inputs[settingsDateFormat-1].Value()
		if dateFmt != "" {
			m.config.DateFormat = dateFmt
		}

		if err := m.config.Save(); err != nil {
			return ConfigErrorMsg{Error: err.Error()}
		}

		return ConfigSavedMsg{}
	}
}

func createInput(placeholder string, charLimit, width int) textinput.Model {
	input := textinput.New()
	input.Placeholder = placeholder
	input.CharLimit = charLimit
	input.Width = width
	return input
}

// ConfigSavedMsg signals config was saved successfully
type ConfigSavedMsg struct{}

// ConfigErrorMsg signals a config error
type ConfigErrorMsg struct {
	Error string
}
