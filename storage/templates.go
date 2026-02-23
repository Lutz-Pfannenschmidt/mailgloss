package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Template represents an email template
type Template struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`
	Variables   []string  `json:"variables,omitempty"`   // List of variable names used in template
	Tags        []string  `json:"tags,omitempty"`        // For categorization
	Description string    `json:"description,omitempty"` // Optional description
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Templates manages the template storage
type Templates struct {
	TemplatesList []Template `json:"templates"`
	filePath      string
}

// NewTemplates creates a new Templates storage instance
func NewTemplates(configDir string) (*Templates, error) {
	filePath := filepath.Join(configDir, "templates.json")
	templates := &Templates{
		TemplatesList: []Template{},
		filePath:      filePath,
	}

	// Create file if it doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := templates.save(); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		// Load existing templates
		if err := templates.load(); err != nil {
			return nil, err
		}
	}

	return templates, nil
}

// load reads templates from the JSON file
func (t *Templates) load() error {
	data, err := os.ReadFile(t.filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, t); err != nil {
		return err
	}

	return nil
}

// save writes templates to the JSON file
func (t *Templates) save() error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(t.filePath, data, 0600)
}

// Add adds a new template
func (t *Templates) Add(template Template) error {
	now := time.Now()
	template.CreatedAt = now
	template.UpdatedAt = now

	// Generate ID if not provided
	if template.ID == "" {
		template.ID = now.Format("20060102150405")
	}

	// Extract variables from subject and body
	template.Variables = extractVariables(template.Subject, template.Body)

	t.TemplatesList = append(t.TemplatesList, template)

	return t.save()
}

// Update updates an existing template by ID
func (t *Templates) Update(id string, updated Template) error {
	for i, template := range t.TemplatesList {
		if template.ID == id {
			updated.ID = id
			updated.CreatedAt = template.CreatedAt
			updated.UpdatedAt = time.Now()

			// Extract variables from subject and body
			updated.Variables = extractVariables(updated.Subject, updated.Body)

			t.TemplatesList[i] = updated

			return t.save()
		}
	}

	return nil
}

// Delete removes a template by ID
func (t *Templates) Delete(id string) error {
	for i, template := range t.TemplatesList {
		if template.ID == id {
			t.TemplatesList = append(t.TemplatesList[:i], t.TemplatesList[i+1:]...)

			_ = template // Avoid unused variable warning
			return t.save()
		}
	}

	return nil
}

// Get returns a template by ID
func (t *Templates) Get(id string) *Template {
	for _, template := range t.TemplatesList {
		if template.ID == id {
			return &template
		}
	}
	return nil
}

// GetAll returns all templates
func (t *Templates) GetAll() []Template {
	return t.TemplatesList
}

// GetByTag returns all templates with a specific tag
func (t *Templates) GetByTag(tag string) []Template {
	var results []Template
	for _, template := range t.TemplatesList {
		for _, t := range template.Tags {
			if t == tag {
				results = append(results, template)
				break
			}
		}
	}
	return results
}

// RenderTemplate replaces variables in template with provided values
// Variables in templates are in the format {{variable_name}}
// Handles both {{name}} and {{ name }} (with or without spaces)
func RenderTemplate(template Template, variables map[string]string) (subject string, body string) {
	subject = template.Subject
	body = template.Body

	for key, value := range variables {
		// Use regex to match {{key}} with any amount of whitespace around the key
		// This handles {{name}}, {{ name }}, {{  name  }}, etc.
		pattern := `\{\{\s*` + regexp.QuoteMeta(key) + `\s*\}\}`
		re := regexp.MustCompile(pattern)

		subject = re.ReplaceAllString(subject, value)
		body = re.ReplaceAllString(body, value)
	}

	return subject, body
}

// extractVariables finds all {{variable}} placeholders in text
func extractVariables(texts ...string) []string {
	variableMap := make(map[string]bool)

	for _, text := range texts {
		pos := 0
		for pos < len(text) {
			// Find opening {{
			startIdx := strings.Index(text[pos:], "{{")
			if startIdx == -1 {
				break
			}
			startIdx += pos

			// Find closing }}
			endIdx := strings.Index(text[startIdx:], "}}")
			if endIdx == -1 {
				break
			}
			endIdx += startIdx

			// Extract variable name
			varName := text[startIdx+2 : endIdx]
			varName = strings.TrimSpace(varName)
			if varName != "" {
				variableMap[varName] = true
			}

			// Move position past this variable
			pos = endIdx + 2
		}
	}

	// Convert map to slice
	variables := make([]string, 0, len(variableMap))
	for v := range variableMap {
		variables = append(variables, v)
	}

	return variables
}
