package mailer

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"mailgloss/config"
	"mailgloss/logger"

	"github.com/ainsleyclark/go-mail/drivers"
	"github.com/ainsleyclark/go-mail/mail"
)

// EmailData represents an email to send
type EmailData struct {
	From        string // Optional override for From address
	FromName    string // Optional override for From name
	To          []string
	CC          []string
	BCC         []string
	Subject     string
	Body        string
	Attachments []string // File paths
}

// Mailer wraps the go-mail functionality
type Mailer struct {
	driver          mail.Mailer
	providerConfig  *config.ProviderConfig
	maxAttachmentMB int
}

// New creates a new Mailer from a provider configuration
func New(pc *config.ProviderConfig) (*Mailer, error) {
	return NewWithLimits(pc, 25)
}

// NewWithLimits creates a new Mailer with custom limits
func NewWithLimits(pc *config.ProviderConfig, maxAttachmentMB int) (*Mailer, error) {
	if err := pc.Validate(); err != nil {
		return nil, fmt.Errorf("invalid provider configuration: %w", err)
	}

	var driver mail.Mailer
	var err error

	mailConfig := mail.Config{
		FromAddress: pc.FromAddress,
		FromName:    pc.FromName,
	}

	switch pc.Type {
	case config.ProviderSMTP:
		mailConfig.URL = pc.SMTP.Host
		mailConfig.Port = pc.SMTP.Port
		mailConfig.FromAddress = pc.SMTP.Username // SMTP uses FromAddress as username
		mailConfig.Password = pc.SMTP.Password
		driver, err = drivers.NewSMTP(mailConfig)

	case config.ProviderMailgun:
		mailConfig.URL = pc.Mailgun.URL
		mailConfig.APIKey = pc.Mailgun.APIKey
		mailConfig.Domain = pc.Mailgun.Domain
		driver, err = drivers.NewMailgun(mailConfig)

	case config.ProviderSendGrid:
		mailConfig.APIKey = pc.SendGrid.APIKey
		driver, err = drivers.NewSendGrid(mailConfig)

	case config.ProviderPostmark:
		mailConfig.APIKey = pc.Postmark.APIKey
		driver, err = drivers.NewPostmark(mailConfig)

	case config.ProviderSparkPost:
		mailConfig.URL = pc.SparkPost.URL
		mailConfig.APIKey = pc.SparkPost.APIKey
		driver, err = drivers.NewSparkPost(mailConfig)

	case config.ProviderPostal:
		mailConfig.URL = pc.Postal.URL
		mailConfig.APIKey = pc.Postal.APIKey
		driver, err = drivers.NewPostal(mailConfig)

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", pc.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create mail driver: %w", err)
	}

	return &Mailer{
		driver:          driver,
		providerConfig:  pc,
		maxAttachmentMB: maxAttachmentMB,
	}, nil
}

// Send sends an email using the configured provider
func (m *Mailer) Send(data EmailData) error {
	logger.Debug("Sending email", "provider", m.providerConfig.Name, "to", data.To, "subject", data.Subject)

	if len(data.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	if data.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if data.Body == "" {
		return fmt.Errorf("body is required")
	}

	// Note: Custom From address should be set in the provider config before creating the mailer.
	// The driver's configured From address will be used for sending.

	// Convert plain text body to simple HTML for providers that require it
	htmlBody := convertPlainTextToHTML(data.Body)

	// Create transmission
	tx := &mail.Transmission{
		Recipients: data.To,
		CC:         data.CC,
		BCC:        data.BCC,
		Subject:    data.Subject,
		PlainText:  data.Body,
		HTML:       htmlBody,
	}

	// Add attachments if any
	if len(data.Attachments) > 0 {
		logger.Debug("Processing attachments", "count", len(data.Attachments))
		attachments := make([]mail.Attachment, 0, len(data.Attachments))
		for _, path := range data.Attachments {
			// Validate attachment before reading
			if err := m.validateAttachment(path); err != nil {
				logger.Error("Attachment validation failed", "path", path, "error", err)
				return fmt.Errorf("invalid attachment %s: %w", path, err)
			}

			// Read file
			fileData, err := os.ReadFile(path)
			if err != nil {
				logger.Error("Failed to read attachment", "path", path, "error", err)
				return fmt.Errorf("failed to read attachment %s: %w", path, err)
			}

			// Extract filename from path
			filename := filepath.Base(path)
			logger.Debug("Attachment added", "filename", filename, "size", len(fileData))

			attachments = append(attachments, mail.Attachment{
				Filename: filename,
				Bytes:    fileData,
			})
		}
		tx.Attachments = attachments
	}

	// Send email
	_, err := m.driver.Send(tx)
	if err != nil {
		logger.Error("Failed to send email", "provider", m.providerConfig.Name, "error", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	logger.Info("Email sent successfully", "provider", m.providerConfig.Name, "to", data.To, "subject", data.Subject)
	return nil
}

// GetProviderName returns the name of the current provider config
func (m *Mailer) GetProviderName() string {
	return m.providerConfig.Name
}

// GetProviderType returns the type of the current provider
func (m *Mailer) GetProviderType() string {
	return string(m.providerConfig.Type)
}

// convertPlainTextToHTML converts plain text to simple HTML
// Preserves line breaks and escapes HTML special characters
func convertPlainTextToHTML(plainText string) string {
	// Use proper HTML escaping
	escaped := html.EscapeString(plainText)
	escaped = strings.ReplaceAll(escaped, "\n", "<br>\n")
	return "<html><body><p>" + escaped + "</p></body></html>"
}

// validateAttachment validates an attachment file path and properties
func (m *Mailer) validateAttachment(path string) error {
	// Check file exists and get info
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist")
		}
		return fmt.Errorf("cannot access file: %w", err)
	}

	// Check it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file")
	}

	// Check size limit (configurable)
	maxSize := int64(m.maxAttachmentMB) * 1024 * 1024
	if info.Size() > maxSize {
		return fmt.Errorf("file too large (max %dMB, got %d bytes)", m.maxAttachmentMB, info.Size())
	}

	// Get absolute path to prevent directory traversal
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check for directory traversal attempts
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("directory traversal not allowed")
	}

	// Verify file is readable
	file, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("file not readable: %w", err)
	}
	file.Close()

	return nil
}
