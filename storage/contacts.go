package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
)

// Contact represents a contact in the address book
type Contact struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Notes     string    `json:"notes,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Contacts manages the contact storage
type Contacts struct {
	ContactsList []Contact `json:"contacts"`
	filePath     string
}

// NewContacts creates a new Contacts storage instance
func NewContacts(configDir string) (*Contacts, error) {
	filePath := filepath.Join(configDir, "contacts.json")
	contacts := &Contacts{
		ContactsList: []Contact{},
		filePath:     filePath,
	}

	// Create file if it doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := contacts.save(); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		// Load existing contacts
		if err := contacts.load(); err != nil {
			return nil, err
		}
	}

	return contacts, nil
}

// load reads contacts from the JSON file
func (c *Contacts) load() error {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, c); err != nil {
		return err
	}

	return nil
}

// save writes contacts to the JSON file
func (c *Contacts) save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.filePath, data, 0600)
}

// Add adds a new contact
func (c *Contacts) Add(contact Contact) error {
	now := time.Now()
	contact.CreatedAt = now
	contact.UpdatedAt = now

	// Generate ID if not provided
	if contact.ID == "" {
		contact.ID = now.Format("20060102150405")
	}

	c.ContactsList = append(c.ContactsList, contact)

	log.Info("Contact added", "name", contact.Name, "email", contact.Email)
	return c.save()
}

// Update updates an existing contact by ID
func (c *Contacts) Update(id string, updated Contact) error {
	for i, contact := range c.ContactsList {
		if contact.ID == id {
			updated.ID = id
			updated.CreatedAt = contact.CreatedAt
			updated.UpdatedAt = time.Now()
			c.ContactsList[i] = updated

			log.Info("Contact updated", "name", updated.Name, "email", updated.Email)
			return c.save()
		}
	}

	log.Warn("Contact not found for update", "id", id)
	return nil
}

// Delete removes a contact by ID
func (c *Contacts) Delete(id string) error {
	for i, contact := range c.ContactsList {
		if contact.ID == id {
			c.ContactsList = append(c.ContactsList[:i], c.ContactsList[i+1:]...)

			log.Info("Contact deleted", "name", contact.Name, "email", contact.Email)
			return c.save()
		}
	}

	log.Warn("Contact not found for deletion", "id", id)
	return nil
}

// Get returns a contact by ID
func (c *Contacts) Get(id string) *Contact {
	for _, contact := range c.ContactsList {
		if contact.ID == id {
			return &contact
		}
	}
	return nil
}

// GetAll returns all contacts
func (c *Contacts) GetAll() []Contact {
	return c.ContactsList
}

// GetByEmail returns a contact by email address
func (c *Contacts) GetByEmail(email string) *Contact {
	for _, contact := range c.ContactsList {
		if contact.Email == email {
			return &contact
		}
	}
	return nil
}

// GetByTag returns all contacts with a specific tag
func (c *Contacts) GetByTag(tag string) []Contact {
	var results []Contact
	for _, contact := range c.ContactsList {
		for _, t := range contact.Tags {
			if t == tag {
				results = append(results, contact)
				break
			}
		}
	}
	return results
}
