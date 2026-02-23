package models

import (
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"mailgloss/config"
	"mailgloss/storage"
	"mailgloss/ui"
)

// ComposeModel represents the compose email tab
type ComposeModel struct {
	inputs           []textinput.Model
	textarea         textarea.Model
	FocusIndex       int
	attachments      []string
	width            int
	height           int
	providers        []string // List of available provider names
	selectedProvider string   // Currently selected provider
	providerIdx      int      // Index in providers list
	config           *config.Config
	fileSelector     *FileSelectModel     // File selector for attachments
	showFileSelector bool                 // Whether to show file selector
	picker           *PickerModel         // Contact/template picker
	showPicker       bool                 // Whether to show picker
	variablePrompt   *VariablePromptModel // Variable prompt for templates
	showVarPrompt    bool                 // Whether to show variable prompt
	contacts         *storage.Contacts
	templates        *storage.Templates
	spinner          spinner.Model // Loading spinner
	isSending        bool          // Whether email is being sent
}

const (
	providerSelector = iota
	fromInput
	toInput
	ccInput
	bccInput
	subjectInput
	attachmentInput
	bodyInput
	sendButton
)

// NewComposeModel creates a new compose model
func NewComposeModel(cfg *config.Config, contacts *storage.Contacts, templates *storage.Templates) ComposeModel {
	providers := cfg.ListProviders()
	selectedProvider := cfg.DefaultProvider
	providerIdx := 0

	// Find index of default provider
	for i, p := range providers {
		if p == selectedProvider {
			providerIdx = i
			break
		}
	}

	// Get limits from config
	limits := cfg.GetLimits()

	// Create text inputs
	inputs := make([]textinput.Model, 6)

	// From field
	inputs[fromInput-1] = textinput.New()
	inputs[fromInput-1].Placeholder = "Name <email@example.com> or user@ (optional)"
	inputs[fromInput-1].CharLimit = 200
	inputs[fromInput-1].Width = 60

	// To field
	inputs[toInput-1] = textinput.New()
	inputs[toInput-1].Placeholder = "recipient@example.com (Ctrl+P for contacts)"
	inputs[toInput-1].CharLimit = limits.MaxEmailsPerField
	inputs[toInput-1].Width = 60

	// CC field
	inputs[ccInput-1] = textinput.New()
	inputs[ccInput-1].Placeholder = "cc@example.com (optional, Ctrl+P for contacts)"
	inputs[ccInput-1].CharLimit = limits.MaxEmailsPerField
	inputs[ccInput-1].Width = 60

	// BCC field
	inputs[bccInput-1] = textinput.New()
	inputs[bccInput-1].Placeholder = "bcc@example.com (optional, Ctrl+P for contacts)"
	inputs[bccInput-1].CharLimit = limits.MaxEmailsPerField
	inputs[bccInput-1].Width = 60

	// Subject field
	inputs[subjectInput-1] = textinput.New()
	inputs[subjectInput-1].Placeholder = "Email subject (Ctrl+T for templates)"
	inputs[subjectInput-1].CharLimit = 200
	inputs[subjectInput-1].Width = 60

	// Attachment field
	inputs[attachmentInput-1] = textinput.New()
	inputs[attachmentInput-1].Placeholder = "/path/to/file.pdf (Enter to add, Ctrl+F for browser)"
	inputs[attachmentInput-1].CharLimit = 500
	inputs[attachmentInput-1].Width = 60

	// Create textarea for body
	ta := textarea.New()
	ta.Placeholder = "Email body... (Ctrl+T for templates)"
	ta.SetWidth(64)
	ta.SetHeight(8)
	ta.CharLimit = limits.MaxBodyLength

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = ui.SpinnerStyle

	return ComposeModel{
		inputs:           inputs,
		textarea:         ta,
		FocusIndex:       providerSelector,
		attachments:      []string{},
		providers:        providers,
		selectedProvider: selectedProvider,
		providerIdx:      providerIdx,
		config:           cfg,
		fileSelector:     nil,
		showFileSelector: false,
		picker:           nil,
		showPicker:       false,
		variablePrompt:   nil,
		showVarPrompt:    false,
		contacts:         contacts,
		templates:        templates,
		spinner:          s,
		isSending:        false,
	}
}

// Init initializes the compose model
func (m ComposeModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

// Update handles messages for the compose model
func (m ComposeModel) Update(msg tea.Msg) (ComposeModel, tea.Cmd) {
	var cmds []tea.Cmd

	// Update spinner
	var spinnerCmd tea.Cmd
	m.spinner, spinnerCmd = m.spinner.Update(msg)
	cmds = append(cmds, spinnerCmd)

	// If variable prompt is open, route messages to it
	if m.showVarPrompt && m.variablePrompt != nil {
		switch msg := msg.(type) {
		case VariablePromptSubmittedMsg:
			// Variables submitted, render template with values
			// Merge defaults (if any) with submitted values so submitted values override defaults
			finalVars := make(map[string]string)
			// If prompt existed, use its defaults
			if m.variablePrompt != nil && m.variablePrompt.defaults != nil {
				for k, v := range m.variablePrompt.defaults {
					finalVars[k] = v
				}
			}
			for k, v := range msg.Values {
				finalVars[k] = v
			}

			subject, body := storage.RenderTemplate(msg.Template, finalVars)

			m.inputs[subjectInput-1].SetValue(subject)
			m.textarea.SetValue(body)

			m.showVarPrompt = false
			m.variablePrompt = nil
			return m, nil

		case VariablePromptClosedMsg:
			// User cancelled variable prompt
			m.showVarPrompt = false
			m.variablePrompt = nil
			return m, nil
		}

		// Update variable prompt
		var cmd tea.Cmd
		*m.variablePrompt, cmd = m.variablePrompt.Update(msg)
		return m, cmd
	}

	// If picker is open, route messages to it
	if m.showPicker && m.picker != nil {
		switch msg := msg.(type) {
		case ContactSelectedMsg:
			// Contact was selected, add to appropriate field
			contact := msg.Contact
			emailStr := fmt.Sprintf("%s <%s>", contact.Name, contact.Email)

			currentValue := m.inputs[msg.TargetField-1].Value()
			if currentValue != "" {
				emailStr = currentValue + ", " + emailStr
			}
			m.inputs[msg.TargetField-1].SetValue(emailStr)

			m.showPicker = false
			m.picker = nil
			return m, nil

		case TemplateSelectedMsg:
			// Template was selected
			template := msg.Template

			// If template has variables, show variable prompt
			if len(template.Variables) > 0 {
				// Build defaults map for system variables
				defaults := make(map[string]string)
				// date default from config
				dateFormat := "02.01.2006"
				if m.config != nil && m.config.DateFormat != "" {
					dateFormat = m.config.DateFormat
				}
				defaults["date"] = time.Now().Format(dateFormat)

				// from_name/from_email defaults from provider config or compose From field
				if m.selectedProvider != "" && m.config != nil {
					if pc, err := m.config.GetProvider(m.selectedProvider); err == nil {
						if pc.FromName != "" {
							defaults["from_name"] = pc.FromName
						}
						if pc.FromAddress != "" {
							defaults["from_email"] = pc.FromAddress
						}
					}
				}

				// Also consider the From input override (Name <email>)
				fromInput := strings.TrimSpace(m.inputs[fromInput-1].Value())
				if fromInput != "" {
					// Parse as name and email if possible
					if addr, err := mail.ParseAddress(fromInput); err == nil {
						if addr.Name != "" {
							defaults["from_name"] = addr.Name
						}
						if addr.Address != "" {
							defaults["from_email"] = addr.Address
						}
					}
				}

				varPrompt := NewVariablePrompt(template, defaults)
				m.variablePrompt = &varPrompt
				m.showVarPrompt = true
				m.showPicker = false
				m.picker = nil
				return m, nil
			}

			// No variables, just insert template as-is
			subject, body := storage.RenderTemplate(template, map[string]string{})

			m.inputs[subjectInput-1].SetValue(subject)
			m.textarea.SetValue(body)

			m.showPicker = false
			m.picker = nil
			return m, nil

		case PickerClosedMsg:
			m.showPicker = false
			m.picker = nil
			return m, nil
		}

		// Update picker
		var cmd tea.Cmd
		*m.picker, cmd = m.picker.Update(msg)
		return m, cmd
	}

	// If file selector is open, route messages to it
	if m.showFileSelector && m.fileSelector != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" {
				// Close file selector
				m.showFileSelector = false
				m.fileSelector = nil
				return m, nil
			}
		case FileSelectedMsg:
			// File was selected, add to attachments
			if msg.Path != "" {
				m.attachments = append(m.attachments, msg.Path)
			}
			m.showFileSelector = false
			m.fileSelector = nil
			return m, nil
		}

		// Update file selector
		var cmd tea.Cmd
		*m.fileSelector, cmd = m.fileSelector.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+p":
			// Open contact picker if on email fields
			if m.FocusIndex >= toInput && m.FocusIndex <= bccInput {
				picker := NewContactPicker(m.contacts, m.FocusIndex)
				m.picker = &picker
				m.showPicker = true
				return m, nil
			}

		case "ctrl+t":
			// Open template picker
			if m.FocusIndex == subjectInput || m.FocusIndex == bodyInput {
				picker := NewTemplatePicker(m.templates)
				m.picker = &picker
				m.showPicker = true
				return m, nil
			}

		case "tab", "shift+tab", "up", "down":
			s := msg.String()

			// Handle navigation
			if s == "up" || s == "shift+tab" {
				m.FocusIndex--
			} else {
				m.FocusIndex++
			}

			if m.FocusIndex > sendButton {
				m.FocusIndex = providerSelector
			} else if m.FocusIndex < providerSelector {
				m.FocusIndex = sendButton
			}

			// Update focus
			for i := range m.inputs {
				m.inputs[i].Blur()
			}

			if m.FocusIndex > providerSelector && m.FocusIndex < bodyInput {
				cmds = append(cmds, m.inputs[m.FocusIndex-1].Focus())
			} else if m.FocusIndex == bodyInput {
				cmds = append(cmds, m.textarea.Focus())
			} else {
				m.textarea.Blur()
			}

			return m, tea.Batch(cmds...)

		case "left", "right":
			// Change provider if on provider selector
			if m.FocusIndex == providerSelector && len(m.providers) > 0 {
				if msg.String() == "left" {
					m.providerIdx--
					if m.providerIdx < 0 {
						m.providerIdx = len(m.providers) - 1
					}
				} else {
					m.providerIdx++
					if m.providerIdx >= len(m.providers) {
						m.providerIdx = 0
					}
				}
				if len(m.providers) > 0 {
					m.selectedProvider = m.providers[m.providerIdx]
				}
			}

		case "ctrl+f":
			// Open file selector if on attachment field
			if m.FocusIndex == attachmentInput {
				fs := NewFileSelectModel("")
				m.fileSelector = &fs
				m.showFileSelector = true
				return m, nil
			}

		case "enter":
			// Add attachment if on attachment field
			if m.FocusIndex == attachmentInput {
				path := strings.TrimSpace(m.inputs[attachmentInput-1].Value())
				if path != "" {
					m.attachments = append(m.attachments, path)
					m.inputs[attachmentInput-1].SetValue("")
				}
				return m, nil
			}

			// Send email if on send button
			if m.FocusIndex == sendButton {
				m.isSending = true
				return m, m.sendEmail()
			}

		case "ctrl+d":
			// Delete last attachment
			if m.FocusIndex == attachmentInput && len(m.attachments) > 0 {
				m.attachments = m.attachments[:len(m.attachments)-1]
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Update focused input/textarea
	if m.FocusIndex == bodyInput {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.FocusIndex > providerSelector && m.FocusIndex < bodyInput {
		var cmd tea.Cmd
		m.inputs[m.FocusIndex-1], cmd = m.inputs[m.FocusIndex-1].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the compose model
func (m ComposeModel) View() string {
	// If variable prompt is open, show it instead
	if m.showVarPrompt && m.variablePrompt != nil {
		return m.variablePrompt.View()
	}

	// If picker is open, show it instead
	if m.showPicker && m.picker != nil {
		return m.picker.View()
	}

	// If file selector is open, show it instead
	if m.showFileSelector && m.fileSelector != nil {
		return m.fileSelector.View()
	}

	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Compose Email"))
	b.WriteString("\n\n")

	// Provider selector
	focused := m.FocusIndex == providerSelector
	label := ui.LabelStyle
	if focused {
		label = label.Foreground(ui.Primary)
	}
	b.WriteString(label.Render("Provider:"))
	b.WriteString("\n")

	if len(m.providers) == 0 {
		emptyState := ui.ErrorStyle.Render("⚠ No providers configured")
		b.WriteString(emptyState)
		b.WriteString("\n\n")

		helpText := ui.InfoStyle.Render(
			"To send emails, you need to configure at least one email provider.\n\n" +
				"Steps to get started:\n" +
				"  1. Press '3' or Tab to go to Settings\n" +
				"  2. Add a new email provider (Mailgun, SMTP, SendGrid, etc.)\n" +
				"  3. Configure your provider credentials\n" +
				"  4. Return to Compose tab to send emails",
		)
		b.WriteString(helpText)
		return b.String()
	} else {
		providerDisplay := m.selectedProvider
		if focused {
			providerDisplay = "< " + providerDisplay + " >"
			b.WriteString(ui.FocusedInputStyle.Render(providerDisplay))
		} else {
			b.WriteString(providerDisplay)
		}
	}
	b.WriteString("\n\n")

	// Render input fields
	labels := []string{"From", "To", "CC", "BCC", "Subject", "Attachments"}
	for i, label := range labels {
		fieldIdx := i + 1 // Offset by 1 because providerSelector is 0
		focused := fieldIdx == m.FocusIndex

		labelStyle := ui.LabelStyle
		if focused {
			labelStyle = labelStyle.Foreground(ui.Primary)
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

	// Show attachments list
	if len(m.attachments) > 0 {
		b.WriteString("\n")
		b.WriteString(ui.ListTitleStyle.Render("Attached Files:"))
		b.WriteString("\n")
		for i, att := range m.attachments {
			b.WriteString(ui.ListItemStyle.Render("  " + att))
			if i < len(m.attachments)-1 {
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	// Body field
	b.WriteString("\n")
	bodyLabel := ui.LabelStyle
	if m.FocusIndex == bodyInput {
		bodyLabel = bodyLabel.Foreground(ui.Primary)
	}
	b.WriteString(bodyLabel.Render("Body:"))
	b.WriteString("\n")

	textareaView := m.textarea.View()
	if m.FocusIndex == bodyInput {
		textareaView = ui.FocusedInputStyle.Render(textareaView)
	}
	b.WriteString(textareaView)
	b.WriteString("\n\n")

	// Send button
	buttonText := "[ Send Email ]"
	if m.FocusIndex == sendButton {
		b.WriteString(ui.ButtonFocusedStyle.Render(buttonText))
	} else {
		b.WriteString(ui.ButtonStyle.Render(buttonText))
	}

	// Show spinner if sending
	if m.isSending {
		b.WriteString("  ")
		b.WriteString(m.spinner.View())
		b.WriteString(" Sending email...")
	}

	b.WriteString("\n\n")
	b.WriteString(ui.RenderHelp(
		"Tab", "next field",
		"←/→", "change provider",
		"Ctrl+P", "contacts",
		"Ctrl+T", "templates",
		"Ctrl+F", "file browser",
	))

	return b.String()
}

// GetEmailData returns the current email data
func (m ComposeModel) GetEmailData() (EmailData, error) {
	to, err := splitEmails(m.inputs[toInput-1].Value())
	if err != nil {
		return EmailData{}, fmt.Errorf("To field: %w", err)
	}

	cc, err := splitEmails(m.inputs[ccInput-1].Value())
	if err != nil {
		return EmailData{}, fmt.Errorf("CC field: %w", err)
	}

	bcc, err := splitEmails(m.inputs[bccInput-1].Value())
	if err != nil {
		return EmailData{}, fmt.Errorf("BCC field: %w", err)
	}

	// Parse From address and name if provided
	fromAddr := ""
	fromName := ""
	fromInput := m.inputs[fromInput-1].Value()
	if fromInput != "" {
		providerConfig, _ := m.config.GetProvider(m.selectedProvider)
		var err error
		fromAddr, fromName, err = parseFromField(fromInput, providerConfig)
		if err != nil {
			return EmailData{}, fmt.Errorf("From field: %w", err)
		}
	}

	return EmailData{
		From:        fromAddr,
		FromName:    fromName,
		To:          to,
		CC:          cc,
		BCC:         bcc,
		Subject:     m.inputs[subjectInput-1].Value(),
		Body:        m.textarea.Value(),
		Attachments: m.attachments,
	}, nil
}

// Clear resets all fields
func (m *ComposeModel) Clear() {
	for i := range m.inputs {
		m.inputs[i].SetValue("")
	}
	m.textarea.SetValue("")
	m.attachments = []string{}
	m.FocusIndex = providerSelector
	m.fileSelector = nil
	m.showFileSelector = false
}

// UpdateProviders updates the provider list from config
func (m *ComposeModel) UpdateProviders(cfg *config.Config) {
	m.config = cfg
	m.providers = cfg.ListProviders()
	m.selectedProvider = cfg.DefaultProvider

	// Find index of default provider
	m.providerIdx = 0
	for i, p := range m.providers {
		if p == m.selectedProvider {
			m.providerIdx = i
			break
		}
	}
}

// sendEmail creates a command to send the email
func (m ComposeModel) sendEmail() tea.Cmd {
	return func() tea.Msg {
		data, err := m.GetEmailData()
		if err != nil {
			return EmailValidationErrorMsg{Error: err.Error()}
		}
		return SendEmailMsg{
			Data:         data,
			ProviderName: m.selectedProvider,
		}
	}
}

// EmailData represents the email composition data
type EmailData struct {
	From        string
	FromName    string
	To          []string
	CC          []string
	BCC         []string
	Subject     string
	Body        string
	Attachments []string
}

// SendEmailMsg is sent when the user wants to send an email
type SendEmailMsg struct {
	Data         EmailData
	ProviderName string
}

// EmailValidationErrorMsg is sent when email validation fails
type EmailValidationErrorMsg struct {
	Error string
}

// splitEmails splits comma-separated email addresses and validates them
func splitEmails(s string) ([]string, error) {
	if s == "" {
		return []string{}, nil
	}

	parts := strings.Split(s, ",")
	emails := make([]string, 0, len(parts))

	for _, part := range parts {
		email := strings.TrimSpace(part)
		if email != "" {
			// Validate email format
			if _, err := mail.ParseAddress(email); err != nil {
				return nil, fmt.Errorf("invalid email '%s': %w", email, err)
			}
			emails = append(emails, email)
		}
	}

	return emails, nil
}

// parseFromField parses the From field to extract name and email
// Supports formats: "Name <email@example.com>", "<email@example.com> Name", or "email@example.com"
func parseFromField(input string, providerConfig *config.ProviderConfig) (email, name string, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", nil
	}

	// Auto-append domain if input ends with @ and provider has a domain field
	if strings.HasSuffix(input, "@") {
		if providerConfig != nil && providerConfig.Mailgun != nil && providerConfig.Mailgun.Domain != "" {
			input += providerConfig.Mailgun.Domain
		}
	}

	// Try to parse as a standard email address with optional name
	addr, parseErr := mail.ParseAddress(input)
	if parseErr == nil {
		// Successfully parsed - mail.ParseAddress handles "Name <email>" format
		return addr.Address, addr.Name, nil
	}

	// If parsing failed, check if it's in the format "<email> Name" (reverse format)
	if strings.Contains(input, "<") && strings.Contains(input, ">") {
		startIdx := strings.Index(input, "<")
		endIdx := strings.Index(input, ">")

		if startIdx < endIdx {
			emailPart := strings.TrimSpace(input[startIdx+1 : endIdx])

			// Extract name - could be before or after the email
			var namePart string
			if startIdx > 0 {
				namePart = strings.TrimSpace(input[:startIdx])
			}
			if endIdx < len(input)-1 && namePart == "" {
				namePart = strings.TrimSpace(input[endIdx+1:])
			}

			// Auto-append domain to email part if needed
			if strings.HasSuffix(emailPart, "@") {
				if providerConfig != nil && providerConfig.Mailgun != nil && providerConfig.Mailgun.Domain != "" {
					emailPart += providerConfig.Mailgun.Domain
				}
			}

			// Validate the email part
			if _, err := mail.ParseAddress(emailPart); err == nil {
				return emailPart, namePart, nil
			}
		}
	}

	// Return the original parse error
	return "", "", fmt.Errorf("invalid email address '%s': %w", input, parseErr)
}
