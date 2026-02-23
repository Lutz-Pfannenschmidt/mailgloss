package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"mailgloss/logger"
)

// SentEmail represents a sent email in history
type SentEmail struct {
	ID           string    `json:"id"`
	From         string    `json:"from"`
	To           []string  `json:"to"`
	CC           []string  `json:"cc,omitempty"`
	BCC          []string  `json:"bcc,omitempty"`
	Subject      string    `json:"subject"`
	Body         string    `json:"body"`
	Attachments  []string  `json:"attachments,omitempty"`
	SentAt       time.Time `json:"sent_at"`
	Provider     string    `json:"provider"`
	ProviderName string    `json:"provider_name"`
	Status       string    `json:"status"` // "success" or "failed"
	Error        string    `json:"error,omitempty"`
}

// History manages the email history
type History struct {
	Emails     []SentEmail `json:"emails"`
	MaxEntries int         `json:"max_entries"`
}

// NewHistory creates a new history with a max entries limit
func NewHistory(maxEntries int) *History {
	if maxEntries <= 0 {
		maxEntries = 100 // Default
	}
	return &History{
		Emails:     []SentEmail{},
		MaxEntries: maxEntries,
	}
}

// GetHistoryPath returns the path to the history file
func GetHistoryPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "mailgloss")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "history.json"), nil
}

// Load reads the history from the history file
func Load() (*History, error) {
	return LoadWithMaxEntries(100)
}

// LoadWithMaxEntries reads the history with a custom max entries limit
func LoadWithMaxEntries(maxEntries int) (*History, error) {
	historyPath, err := GetHistoryPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty history if file doesn't exist
			return NewHistory(maxEntries), nil
		}
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	var history History
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("failed to parse history file: %w", err)
	}

	// Set max entries if not set or use provided value
	if history.MaxEntries == 0 {
		history.MaxEntries = maxEntries
	}

	return &history, nil
}

// Save writes the history to the history file
func (h *History) Save() error {
	historyPath, err := GetHistoryPath()
	if err != nil {
		return err
	}

	// Trim history to max entries
	if len(h.Emails) > h.MaxEntries {
		h.Emails = h.Emails[len(h.Emails)-h.MaxEntries:]
	}

	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(historyPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}

// Add adds a new email to the history
func (h *History) Add(email SentEmail) error {
	// Generate ID if not set
	if email.ID == "" {
		email.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Set sent time if not set
	if email.SentAt.IsZero() {
		email.SentAt = time.Now()
	}

	h.Emails = append(h.Emails, email)
	logger.Debug("Added email to history", "id", email.ID, "status", email.Status)
	return h.Save()
}

// GetRecent returns the most recent N emails
func (h *History) GetRecent(n int) []SentEmail {
	if n <= 0 || len(h.Emails) == 0 {
		return []SentEmail{}
	}

	if n > len(h.Emails) {
		n = len(h.Emails)
	}

	// Return in reverse order (most recent first)
	result := make([]SentEmail, n)
	for i := 0; i < n; i++ {
		result[i] = h.Emails[len(h.Emails)-1-i]
	}

	return result
}

// GetAll returns all emails in reverse chronological order
func (h *History) GetAll() []SentEmail {
	return h.GetRecent(len(h.Emails))
}

// Clear removes all emails from history
func (h *History) Clear() error {
	h.Emails = []SentEmail{}
	return h.Save()
}
