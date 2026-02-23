package storage

import (
	"reflect"
	"sort"
	"testing"
)

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name     string
		texts    []string
		expected []string
	}{
		{
			name:     "single variable",
			texts:    []string{"Hello {{name}}"},
			expected: []string{"name"},
		},
		{
			name:     "multiple variables",
			texts:    []string{"Hello {{name}}, welcome to {{company}}"},
			expected: []string{"name", "company"},
		},
		{
			name:     "variables in subject and body",
			texts:    []string{"Subject: {{topic}}", "Body: Hi {{name}}, let's discuss {{topic}}"},
			expected: []string{"topic", "name"},
		},
		{
			name:     "duplicate variables",
			texts:    []string{"{{name}} {{name}} {{name}}"},
			expected: []string{"name"},
		},
		{
			name:     "variables with spaces",
			texts:    []string{"{{ name }} {{ company }}"},
			expected: []string{"name", "company"},
		},
		{
			name:     "no variables",
			texts:    []string{"Hello world"},
			expected: []string{},
		},
		{
			name:     "empty text",
			texts:    []string{""},
			expected: []string{},
		},
		{
			name:     "malformed - no closing",
			texts:    []string{"Hello {{name"},
			expected: []string{},
		},
		{
			name:     "malformed - no opening",
			texts:    []string{"Hello name}}"},
			expected: []string{},
		},
		{
			name:     "empty variable name",
			texts:    []string{"Hello {{}}"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVariables(tt.texts...)

			// Sort both slices for comparison (since map iteration order is random)
			sort.Strings(result)
			sort.Strings(tt.expected)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("extractVariables() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name            string
		template        Template
		variables       map[string]string
		expectedSubject string
		expectedBody    string
	}{
		{
			name: "simple replacement",
			template: Template{
				Subject: "Hello {{name}}",
				Body:    "Welcome {{name}}!",
			},
			variables: map[string]string{
				"name": "John",
			},
			expectedSubject: "Hello John",
			expectedBody:    "Welcome John!",
		},
		{
			name: "variables with spaces",
			template: Template{
				Subject: "Hello {{ name }}",
				Body:    "Welcome {{  name  }}!",
			},
			variables: map[string]string{
				"name": "John",
			},
			expectedSubject: "Hello John",
			expectedBody:    "Welcome John!",
		},
		{
			name: "mixed spacing",
			template: Template{
				Subject: "Hi {{name}} and {{ company }}",
				Body:    "Welcome {{name}} to {{  company  }}!",
			},
			variables: map[string]string{
				"name":    "John",
				"company": "Acme Corp",
			},
			expectedSubject: "Hi John and Acme Corp",
			expectedBody:    "Welcome John to Acme Corp!",
		},
		{
			name: "no variables",
			template: Template{
				Subject: "Hello World",
				Body:    "Welcome!",
			},
			variables:       map[string]string{},
			expectedSubject: "Hello World",
			expectedBody:    "Welcome!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject, body := RenderTemplate(tt.template, tt.variables)

			if subject != tt.expectedSubject {
				t.Errorf("RenderTemplate() subject = %v, want %v", subject, tt.expectedSubject)
			}
			if body != tt.expectedBody {
				t.Errorf("RenderTemplate() body = %v, want %v", body, tt.expectedBody)
			}
		})
	}
}
