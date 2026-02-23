package models

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"mailgloss/config"
	"mailgloss/mailer"
	"mailgloss/storage"
	"mailgloss/ui"
)

// Tab represents a tab in the app
type Tab int

const (
	TabCompose Tab = iota
	TabHistory
	TabContacts
	TabTemplates
	TabSettings
)

// AppModel is the main application model
type AppModel struct {
	activeTab      Tab
	composeModel   ComposeModel
	historyModel   HistoryModel
	contactsModel  ContactsModel
	templatesModel TemplatesModel
	settingsModel  SettingsModel
	config         *config.Config
	history        *storage.History
	contacts       *storage.Contacts
	templates      *storage.Templates
	width          int
	height         int
	statusMsg      string
	errorMsg       string
	quitting       bool
}

// NewAppModel creates a new app model
func NewAppModel() (*AppModel, error) {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Get config directory
	configPath, err := config.GetConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}
	configDir := filepath.Dir(configPath)

	// Load history with configured limits
	limits := cfg.GetLimits()
	hist, err := storage.LoadWithMaxEntries(limits.MaxHistoryEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to load history: %w", err)
	}

	// Load contacts
	contacts, err := storage.NewContacts(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load contacts: %w", err)
	}

	// Load templates
	templates, err := storage.NewTemplates(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Create models
	composeModel := NewComposeModel(cfg, contacts, templates)
	historyModel := NewHistoryModel(hist)
	contactsModel := NewContactsModel(contacts)
	templatesModel := NewTemplatesModel(templates)
	settingsModel := NewSettingsModel(cfg)

	// If no providers configured, start on settings tab
	activeTab := TabCompose
	if len(cfg.Providers) == 0 {
		activeTab = TabSettings
	}

	return &AppModel{
		activeTab:      activeTab,
		composeModel:   composeModel,
		historyModel:   historyModel,
		contactsModel:  contactsModel,
		templatesModel: templatesModel,
		settingsModel:  settingsModel,
		config:         cfg,
		history:        hist,
		contacts:       contacts,
		templates:      templates,
	}, nil
}

// Init initializes the app model
func (m AppModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the app model
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Only handle global shortcuts if we're not in an active input/textarea
		isTyping := false
		switch m.activeTab {
		case TabCompose:
			// Check if we're in an input field or textarea (not on provider selector or send button)
			isTyping = m.composeModel.FocusIndex > providerSelector && m.composeModel.FocusIndex < sendButton
		case TabContacts:
			// Check if we're in the add/edit view
			isTyping = (m.contactsModel.currentView == ContactsViewAdd || m.contactsModel.currentView == ContactsViewEdit) &&
				m.contactsModel.FocusIndex >= contactName && m.contactsModel.FocusIndex < contactSaveButton
		case TabTemplates:
			// Check if we're in the add/edit view
			isTyping = (m.templatesModel.currentView == TemplatesViewAdd || m.templatesModel.currentView == TemplatesViewEdit) &&
				m.templatesModel.FocusIndex >= templateName && m.templatesModel.FocusIndex < templateSaveButton
		case TabSettings:
			// Check if we're in an input field (not in list view or on buttons)
			isTyping = m.settingsModel.currentView != SettingsViewList &&
				m.settingsModel.FocusIndex >= settingsName &&
				m.settingsModel.FocusIndex < settingsSaveButton
		}

		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "q":
			if !isTyping {
				m.quitting = true
				return m, tea.Quit
			}

		case "1":
			if !isTyping {
				m.activeTab = TabCompose
				m.statusMsg = ""
				m.errorMsg = ""
				return m, nil
			}

		case "2":
			if !isTyping {
				m.activeTab = TabHistory
				m.statusMsg = ""
				m.errorMsg = ""
				return m, nil
			}

		case "3":
			if !isTyping {
				m.activeTab = TabContacts
				m.statusMsg = ""
				m.errorMsg = ""
				return m, nil
			}

		case "4":
			if !isTyping {
				m.activeTab = TabTemplates
				m.statusMsg = ""
				m.errorMsg = ""
				return m, nil
			}

		case "5":
			if !isTyping {
				m.activeTab = TabSettings
				m.statusMsg = ""
				m.errorMsg = ""
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case SendEmailMsg:
		// Handle email sending
		m.statusMsg = ""
		m.errorMsg = ""

		// Get provider config
		if msg.ProviderName == "" {
			m.errorMsg = "Please select a provider"
			m.composeModel.isSending = false
			return m, nil
		}

		providerConfig, err := m.config.GetProvider(msg.ProviderName)
		if err != nil {
			m.errorMsg = fmt.Sprintf("Provider error: %v", err)
			m.composeModel.isSending = false
			return m, nil
		}

		// If a custom From address is provided, create a modified config with that address
		if msg.Data.From != "" {
			// Clone the provider config to avoid modifying the original
			modifiedConfig := *providerConfig
			modifiedConfig.FromAddress = msg.Data.From
			// Override FromName if provided, otherwise keep config default
			if msg.Data.FromName != "" {
				modifiedConfig.FromName = msg.Data.FromName
			}
			providerConfig = &modifiedConfig
		}

		// Create mailer for this provider
		limits := m.config.GetLimits()
		ml, err := mailer.NewWithLimits(providerConfig, limits.MaxAttachmentSizeMB)
		if err != nil {
			m.errorMsg = fmt.Sprintf("Failed to initialize mailer: %v", err)
			m.composeModel.isSending = false
			return m, nil
		}

		// Send email
		emailData := mailer.EmailData{
			From:        msg.Data.From,
			FromName:    msg.Data.FromName,
			To:          msg.Data.To,
			CC:          msg.Data.CC,
			BCC:         msg.Data.BCC,
			Subject:     msg.Data.Subject,
			Body:        msg.Data.Body,
			Attachments: msg.Data.Attachments,
		}

		err = ml.Send(emailData)

		// Save to history
		historyEntry := storage.SentEmail{
			From:         msg.Data.From,
			To:           msg.Data.To,
			CC:           msg.Data.CC,
			BCC:          msg.Data.BCC,
			Subject:      msg.Data.Subject,
			Body:         msg.Data.Body,
			Attachments:  msg.Data.Attachments,
			Provider:     ml.GetProviderType(),
			ProviderName: msg.ProviderName,
		}

		if err != nil {
			m.errorMsg = fmt.Sprintf("Failed to send email: %v", err)
			historyEntry.Status = "failed"
			historyEntry.Error = err.Error()
			m.composeModel.isSending = false
		} else {
			m.statusMsg = "Email sent successfully!"
			historyEntry.Status = "success"
			m.composeModel.Clear()
			m.composeModel.isSending = false
		}

		// Save to history regardless of success/failure
		m.history.Add(historyEntry)

		// Refresh history view
		return m, func() tea.Msg {
			return RefreshHistoryMsg{}
		}

	case EmailValidationErrorMsg:
		// Handle email validation errors
		m.statusMsg = ""
		m.errorMsg = fmt.Sprintf("Validation error: %s", msg.Error)
		m.composeModel.isSending = false
		return m, nil

	case ConfigSavedMsg:
		// Reload config and update compose model with new provider list
		if cfg, err := config.Load(); err == nil {
			m.config = cfg
			m.composeModel.UpdateProviders(cfg)
		}

	case RefreshHistoryMsg:
		// Pass to history model
		m.historyModel, cmd = m.historyModel.Update(msg)
		return m, cmd
	}

	// Update active tab
	switch m.activeTab {
	case TabCompose:
		m.composeModel, cmd = m.composeModel.Update(msg)
		cmds = append(cmds, cmd)

	case TabHistory:
		m.historyModel, cmd = m.historyModel.Update(msg)
		cmds = append(cmds, cmd)

	case TabContacts:
		m.contactsModel, cmd = m.contactsModel.Update(msg)
		cmds = append(cmds, cmd)

	case TabTemplates:
		m.templatesModel, cmd = m.templatesModel.Update(msg)
		cmds = append(cmds, cmd)

	case TabSettings:
		m.settingsModel, cmd = m.settingsModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the app model
func (m AppModel) View() string {
	if m.quitting {
		return ""
	}

	// Render tabs with icons
	tabs := []string{"✉ Compose", "📜 History", "👤 Contacts", "📝 Templates", "⚙ Settings"}
	tabBar := ui.RenderTabs(tabs, int(m.activeTab))

	// Render active tab content
	var content string
	switch m.activeTab {
	case TabCompose:
		content = m.composeModel.View()
	case TabHistory:
		content = m.historyModel.View()
	case TabContacts:
		content = m.contactsModel.View()
	case TabTemplates:
		content = m.templatesModel.View()
	case TabSettings:
		content = m.settingsModel.View()
	}

	// Render status messages
	var status string
	if m.statusMsg != "" {
		status = "\n" + ui.SuccessStyle.Render(m.statusMsg)
	} else if m.errorMsg != "" {
		status = "\n" + ui.ErrorStyle.Render(m.errorMsg)
	}

	// Render footer
	footer := ui.HelpStyle.Render("Press q or Ctrl+C to quit | Numbers 1-5 switch tabs (when not typing)")

	return tabBar + "\n\n" + content + status + "\n\n" + footer
}
