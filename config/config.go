package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"mailgloss/logger"
)

// Provider represents the email provider type
type Provider string

const (
	ProviderMailgun   Provider = "mailgun"
	ProviderSMTP      Provider = "smtp"
	ProviderSendGrid  Provider = "sendgrid"
	ProviderPostmark  Provider = "postmark"
	ProviderSparkPost Provider = "sparkpost"
	ProviderPostal    Provider = "postal"
)

// Config represents the application configuration
type Config struct {
	Providers       map[string]*ProviderConfig `yaml:"providers"`
	DefaultProvider string                     `yaml:"default_provider,omitempty"`
	Limits          *Limits                    `yaml:"limits,omitempty"`
	// DateFormat is the layout used for the {{date}} system variable.
	// Uses Go time layout syntax. Default: "02.01.2006" (DD.MM.YYYY).
	DateFormat string `yaml:"date_format,omitempty"`
}

// Limits represents configurable limits
type Limits struct {
	MaxAttachmentSizeMB int `yaml:"max_attachment_size_mb,omitempty"` // Default: 25
	MaxHistoryEntries   int `yaml:"max_history_entries,omitempty"`    // Default: 100
	MaxBodyLength       int `yaml:"max_body_length,omitempty"`        // Default: 10000
	MaxEmailsPerField   int `yaml:"max_emails_per_field,omitempty"`   // Default: 500
}

// ProviderConfig represents a single named provider configuration
type ProviderConfig struct {
	Name        string   `yaml:"name"`
	Type        Provider `yaml:"type"`
	FromAddress string   `yaml:"from_address"`
	FromName    string   `yaml:"from_name"`

	// Provider-specific configs (only one should be populated based on Type)
	SMTP      *SMTPConfig      `yaml:"smtp,omitempty"`
	Mailgun   *MailgunConfig   `yaml:"mailgun,omitempty"`
	SendGrid  *SendGridConfig  `yaml:"sendgrid,omitempty"`
	Postmark  *PostmarkConfig  `yaml:"postmark,omitempty"`
	SparkPost *SparkPostConfig `yaml:"sparkpost,omitempty"`
	Postal    *PostalConfig    `yaml:"postal,omitempty"`
}

// SMTPConfig contains SMTP-specific settings
type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// MailgunConfig contains Mailgun-specific settings
type MailgunConfig struct {
	APIKey string `yaml:"api_key"`
	Domain string `yaml:"domain"`
	URL    string `yaml:"url"` // e.g., https://api.mailgun.net or https://api.eu.mailgun.net
}

// SendGridConfig contains SendGrid-specific settings
type SendGridConfig struct {
	APIKey string `yaml:"api_key"`
}

// PostmarkConfig contains Postmark-specific settings
type PostmarkConfig struct {
	APIKey string `yaml:"api_key"`
}

// SparkPostConfig contains SparkPost-specific settings
type SparkPostConfig struct {
	APIKey string `yaml:"api_key"`
	URL    string `yaml:"url"` // e.g., https://api.sparkpost.com or https://api.eu.sparkpost.com
}

// PostalConfig contains Postal-specific settings
type PostalConfig struct {
	URL    string `yaml:"url"`
	APIKey string `yaml:"api_key"`
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "mailgloss")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "config.yaml"), nil
}

// Load reads the configuration from the config file
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	logger.Debug("Loading config", "path", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("Config file not found, using default config")
			// Return default config if file doesn't exist
			return DefaultConfig(), nil
		}
		logger.Error("Failed to read config file", "error", err)
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		logger.Error("Failed to parse config file", "error", err)
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	logger.Info("Config loaded successfully", "providers", len(cfg.Providers))
	return &cfg, nil
}

// Save writes the configuration to the config file
func (c *Config) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	logger.Debug("Saving config", "path", configPath)

	data, err := yaml.Marshal(c)
	if err != nil {
		logger.Error("Failed to marshal config", "error", err)
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		logger.Error("Failed to write config file", "error", err)
		return fmt.Errorf("failed to write config file: %w", err)
	}

	logger.Info("Config saved successfully")
	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Providers:       make(map[string]*ProviderConfig),
		DefaultProvider: "",
		Limits:          DefaultLimits(),
		DateFormat:      "02.01.2006",
	}
}

// DefaultLimits returns default limits
func DefaultLimits() *Limits {
	return &Limits{
		MaxAttachmentSizeMB: 25,
		MaxHistoryEntries:   100,
		MaxBodyLength:       10000,
		MaxEmailsPerField:   500,
	}
}

// GetLimits returns the limits, using defaults if not configured
func (c *Config) GetLimits() *Limits {
	if c.Limits == nil {
		c.Limits = DefaultLimits()
	}
	// Fill in any missing values with defaults
	defaults := DefaultLimits()
	if c.Limits.MaxAttachmentSizeMB == 0 {
		c.Limits.MaxAttachmentSizeMB = defaults.MaxAttachmentSizeMB
	}
	if c.Limits.MaxHistoryEntries == 0 {
		c.Limits.MaxHistoryEntries = defaults.MaxHistoryEntries
	}
	if c.Limits.MaxBodyLength == 0 {
		c.Limits.MaxBodyLength = defaults.MaxBodyLength
	}
	if c.Limits.MaxEmailsPerField == 0 {
		c.Limits.MaxEmailsPerField = defaults.MaxEmailsPerField
	}
	return c.Limits
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Providers) == 0 {
		return nil // Empty config is valid (will show settings on first run)
	}

	// Validate each provider config
	for name, pc := range c.Providers {
		if err := pc.Validate(); err != nil {
			return fmt.Errorf("provider '%s': %w", name, err)
		}
	}

	// Validate default provider exists if set
	if c.DefaultProvider != "" {
		if _, ok := c.Providers[c.DefaultProvider]; !ok {
			return fmt.Errorf("default provider '%s' does not exist", c.DefaultProvider)
		}
	}

	return nil
}

// Validate checks if a provider configuration is valid
func (pc *ProviderConfig) Validate() error {
	if pc.Name == "" {
		return fmt.Errorf("name is required")
	}

	if pc.FromAddress == "" {
		return fmt.Errorf("from_address is required")
	}

	switch pc.Type {
	case ProviderSMTP:
		if pc.SMTP == nil {
			return fmt.Errorf("smtp configuration is required")
		}
		if pc.SMTP.Host == "" {
			return fmt.Errorf("smtp.host is required")
		}
		if pc.SMTP.Port == 0 {
			return fmt.Errorf("smtp.port is required")
		}
	case ProviderMailgun:
		if pc.Mailgun == nil {
			return fmt.Errorf("mailgun configuration is required")
		}
		if pc.Mailgun.APIKey == "" {
			return fmt.Errorf("mailgun.api_key is required")
		}
		if pc.Mailgun.Domain == "" {
			return fmt.Errorf("mailgun.domain is required")
		}
	case ProviderSendGrid:
		if pc.SendGrid == nil {
			return fmt.Errorf("sendgrid configuration is required")
		}
		if pc.SendGrid.APIKey == "" {
			return fmt.Errorf("sendgrid.api_key is required")
		}
	case ProviderPostmark:
		if pc.Postmark == nil {
			return fmt.Errorf("postmark configuration is required")
		}
		if pc.Postmark.APIKey == "" {
			return fmt.Errorf("postmark.api_key is required")
		}
	case ProviderSparkPost:
		if pc.SparkPost == nil {
			return fmt.Errorf("sparkpost configuration is required")
		}
		if pc.SparkPost.APIKey == "" {
			return fmt.Errorf("sparkpost.api_key is required")
		}
	case ProviderPostal:
		if pc.Postal == nil {
			return fmt.Errorf("postal configuration is required")
		}
		if pc.Postal.URL == "" {
			return fmt.Errorf("postal.url is required")
		}
		if pc.Postal.APIKey == "" {
			return fmt.Errorf("postal.api_key is required")
		}
	default:
		return fmt.Errorf("unknown provider type: %s", pc.Type)
	}

	return nil
}

// GetProvider retrieves a provider config by name
func (c *Config) GetProvider(name string) (*ProviderConfig, error) {
	pc, ok := c.Providers[name]
	if !ok {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}
	return pc, nil
}

// AddProvider adds or updates a provider configuration
func (c *Config) AddProvider(pc *ProviderConfig) error {
	if err := pc.Validate(); err != nil {
		return err
	}

	if c.Providers == nil {
		c.Providers = make(map[string]*ProviderConfig)
	}

	c.Providers[pc.Name] = pc

	// Set as default if it's the first provider
	if len(c.Providers) == 1 {
		c.DefaultProvider = pc.Name
	}

	return nil
}

// DeleteProvider removes a provider configuration
func (c *Config) DeleteProvider(name string) error {
	if _, ok := c.Providers[name]; !ok {
		return fmt.Errorf("provider '%s' not found", name)
	}

	delete(c.Providers, name)

	// Update default if we deleted it
	if c.DefaultProvider == name {
		c.DefaultProvider = ""
		// Set to first available provider
		for n := range c.Providers {
			c.DefaultProvider = n
			break
		}
	}

	return nil
}

// ListProviders returns a slice of all provider names
func (c *Config) ListProviders() []string {
	names := make([]string, 0, len(c.Providers))
	for name := range c.Providers {
		names = append(names, name)
	}
	return names
}

// ConfigExists checks if a config file exists
func ConfigExists() bool {
	configPath, err := GetConfigPath()
	if err != nil {
		return false
	}

	_, err = os.Stat(configPath)
	return err == nil
}
