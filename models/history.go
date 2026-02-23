package models

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"mailgloss/storage"
	"mailgloss/ui"
)

// HistoryModel represents the email history tab
type HistoryModel struct {
	history       *storage.History
	selectedIndex int
	viewingEmail  bool
	width         int
	height        int
}

// NewHistoryModel creates a new history model
func NewHistoryModel(history *storage.History) HistoryModel {
	return HistoryModel{
		history:       history,
		selectedIndex: 0,
		viewingEmail:  false,
	}
}

// Init initializes the history model
func (m HistoryModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the history model
func (m HistoryModel) Update(msg tea.Msg) (HistoryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		emails := m.history.GetAll()

		if m.viewingEmail {
			// Viewing an email, go back on any key
			switch msg.String() {
			case "esc", "q", "enter":
				m.viewingEmail = false
			}
			return m, nil
		}

		// Navigating list
		switch msg.String() {
		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
		case "down", "j":
			if m.selectedIndex < len(emails)-1 {
				m.selectedIndex++
			}
		case "enter":
			if len(emails) > 0 {
				m.viewingEmail = true
			}
		case "g":
			m.selectedIndex = 0
		case "G":
			if len(emails) > 0 {
				m.selectedIndex = len(emails) - 1
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case RefreshHistoryMsg:
		// Reload history from storage
		if h, err := storage.Load(); err == nil {
			m.history = h
			// Adjust selected index if needed
			emails := m.history.GetAll()
			if m.selectedIndex >= len(emails) {
				m.selectedIndex = len(emails) - 1
			}
			if m.selectedIndex < 0 {
				m.selectedIndex = 0
			}
		}
	}

	return m, nil
}

// View renders the history model
func (m HistoryModel) View() string {
	emails := m.history.GetAll()

	if m.viewingEmail && len(emails) > 0 && m.selectedIndex < len(emails) {
		return m.viewEmail(emails[m.selectedIndex])
	}

	return m.viewList(emails)
}

// viewList renders the list of sent emails
func (m HistoryModel) viewList(emails []storage.SentEmail) string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Sent Email History"))
	b.WriteString("\n\n")

	if len(emails) == 0 {
		b.WriteString(ui.InfoStyle.Render("No emails sent yet."))
		b.WriteString("\n\n")
		b.WriteString(ui.RenderHelp("Tab", "switch tabs"))
		return b.String()
	}

	b.WriteString(ui.SubtitleStyle.Render(fmt.Sprintf("Total: %d emails", len(emails))))
	b.WriteString("\n")

	// Show list of emails
	for i, email := range emails {
		var line string

		// Format timestamp
		timestamp := email.SentAt.Format("2006-01-02 15:04")

		// Get first recipient
		recipient := "no recipient"
		if len(email.To) > 0 {
			recipient = email.To[0]
			if len(email.To) > 1 {
				recipient += fmt.Sprintf(" +%d", len(email.To)-1)
			}
		}

		// Truncate subject if too long
		subject := email.Subject
		if len(subject) > 50 {
			subject = subject[:47] + "..."
		}

		// Status indicator
		statusIcon := "✓"
		if email.Status != "success" {
			statusIcon = "✗"
		}

		line = fmt.Sprintf("%s [%s] %s - %s",
			statusIcon,
			timestamp,
			recipient,
			subject,
		)

		if i == m.selectedIndex {
			b.WriteString(ui.SelectedItemStyle.Render("→ " + line))
		} else {
			b.WriteString(ui.ListItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(ui.RenderHelp(
		"↑/k", "up",
		"↓/j", "down",
		"Enter", "view",
		"g/G", "top/bottom",
	))

	return b.String()
}

// viewEmail renders a single email's details
func (m HistoryModel) viewEmail(email storage.SentEmail) string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Email Details"))
	b.WriteString("\n\n")

	// Status
	if email.Status == "success" {
		b.WriteString(ui.SuccessStyle.Render("✓ Sent Successfully"))
	} else {
		b.WriteString(ui.ErrorStyle.Render("✗ Failed to Send"))
		if email.Error != "" {
			b.WriteString("\n")
			b.WriteString(ui.ErrorStyle.Render("Error: " + email.Error))
		}
	}
	b.WriteString("\n\n")

	// Metadata
	b.WriteString(ui.LabelStyle.Render("Sent At:"))
	b.WriteString(" " + email.SentAt.Format("2006-01-02 15:04:05") + "\n")

	b.WriteString(ui.LabelStyle.Render("From:"))
	b.WriteString(" " + email.From + "\n")

	b.WriteString(ui.LabelStyle.Render("Provider:"))
	b.WriteString(" " + email.Provider + "\n")

	b.WriteString(ui.LabelStyle.Render("Provider Config:"))
	b.WriteString(" " + email.ProviderName + "\n\n")

	// Recipients
	b.WriteString(ui.LabelStyle.Render("To:"))
	b.WriteString(" " + strings.Join(email.To, ", ") + "\n")

	if len(email.CC) > 0 {
		b.WriteString(ui.LabelStyle.Render("CC:"))
		b.WriteString(" " + strings.Join(email.CC, ", ") + "\n")
	}

	if len(email.BCC) > 0 {
		b.WriteString(ui.LabelStyle.Render("BCC:"))
		b.WriteString(" " + strings.Join(email.BCC, ", ") + "\n")
	}

	b.WriteString("\n")

	// Subject
	b.WriteString(ui.LabelStyle.Render("Subject:"))
	b.WriteString("\n" + email.Subject + "\n\n")

	// Attachments
	if len(email.Attachments) > 0 {
		b.WriteString(ui.LabelStyle.Render("Attachments:"))
		b.WriteString("\n")
		for _, att := range email.Attachments {
			b.WriteString("  • " + att + "\n")
		}
		b.WriteString("\n")
	}

	// Body
	b.WriteString(ui.LabelStyle.Render("Body:"))
	b.WriteString("\n")
	b.WriteString(ui.DividerStyle.Render(strings.Repeat("─", 60)))
	b.WriteString("\n")
	b.WriteString(email.Body)
	b.WriteString("\n")
	b.WriteString(ui.DividerStyle.Render(strings.Repeat("─", 60)))
	b.WriteString("\n\n")

	b.WriteString(ui.RenderHelp("Esc/Enter", "back to list"))

	return b.String()
}

// RefreshHistoryMsg signals the history should be reloaded
type RefreshHistoryMsg struct{}
