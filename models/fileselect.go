package models

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"mailgloss/ui"
)

// FileSelectModel represents the file selector interface
type FileSelectModel struct {
	currentPath string
	entries     []fs.DirEntry
	selectedIdx int
	offset      int
	height      int
	width       int
	showHidden  bool
	filterExt   []string // Optional file extension filters (e.g., []string{".pdf", ".txt"})
	err         error
}

// NewFileSelectModel creates a new file selector model
func NewFileSelectModel(startPath string) FileSelectModel {
	if startPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			startPath = "."
		} else {
			startPath = home
		}
	}

	m := FileSelectModel{
		currentPath: startPath,
		selectedIdx: 0,
		offset:      0,
		height:      20,
		showHidden:  false,
		filterExt:   []string{},
	}

	m.loadDirectory()
	return m
}

// Init initializes the file selector
func (m FileSelectModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the file selector
func (m FileSelectModel) Update(msg tea.Msg) (FileSelectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
				// Adjust offset for scrolling
				if m.selectedIdx < m.offset {
					m.offset = m.selectedIdx
				}
			}

		case "down", "j":
			if m.selectedIdx < len(m.entries)-1 {
				m.selectedIdx++
				// Adjust offset for scrolling
				if m.selectedIdx >= m.offset+m.height {
					m.offset = m.selectedIdx - m.height + 1
				}
			}

		case "enter", " ":
			// Enter directory or select file
			if len(m.entries) == 0 {
				return m, nil
			}
			entry := m.entries[m.selectedIdx]
			if entry.IsDir() {
				// Navigate into directory
				newPath := filepath.Join(m.currentPath, entry.Name())
				m.currentPath = newPath
				m.selectedIdx = 0
				m.offset = 0
				m.loadDirectory()
			} else {
				// File selected - return command with selected file path
				fullPath := filepath.Join(m.currentPath, entry.Name())
				return m, func() tea.Msg {
					return FileSelectedMsg{Path: fullPath}
				}
			}

		case "backspace", "h":
			// Go up one directory
			parent := filepath.Dir(m.currentPath)
			if parent != m.currentPath {
				m.currentPath = parent
				m.selectedIdx = 0
				m.offset = 0
				m.loadDirectory()
			}

		case ".":
			// Toggle hidden files
			m.showHidden = !m.showHidden
			m.selectedIdx = 0
			m.offset = 0
			m.loadDirectory()

		case "~":
			// Go to home directory
			home, err := os.UserHomeDir()
			if err == nil {
				m.currentPath = home
				m.selectedIdx = 0
				m.offset = 0
				m.loadDirectory()
			}

		case "/":
			// Go to root directory
			m.currentPath = "/"
			m.selectedIdx = 0
			m.offset = 0
			m.loadDirectory()

		case "g":
			// Go to top
			m.selectedIdx = 0
			m.offset = 0

		case "G":
			// Go to bottom
			if len(m.entries) > 0 {
				m.selectedIdx = len(m.entries) - 1
				if m.selectedIdx >= m.height {
					m.offset = m.selectedIdx - m.height + 1
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height - 10 // Reserve space for header and footer
	}

	return m, nil
}

// View renders the file selector
func (m FileSelectModel) View() string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Select File"))
	b.WriteString("\n\n")

	// Show current path
	b.WriteString(ui.LabelStyle.Render("Current Directory:"))
	b.WriteString("\n")
	b.WriteString(ui.InfoStyle.Render(m.currentPath))
	b.WriteString("\n\n")

	// Show error if any
	if m.err != nil {
		b.WriteString(ui.ErrorStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n\n")
	}

	// Show entries
	if len(m.entries) == 0 {
		b.WriteString(ui.WarningStyle.Render("Empty directory"))
		b.WriteString("\n")
	} else {
		// Calculate visible range
		start := m.offset
		end := m.offset + m.height
		if end > len(m.entries) {
			end = len(m.entries)
		}

		// Render visible entries
		for i := start; i < end; i++ {
			entry := m.entries[i]
			name := entry.Name()

			// Add indicator for directories
			if entry.IsDir() {
				name = name + "/"
			}

			// Style based on selection
			if i == m.selectedIdx {
				b.WriteString(ui.SelectedItemStyle.Render("▸ " + name))
			} else {
				b.WriteString(ui.ListItemStyle.Render("  " + name))
			}
			b.WriteString("\n")
		}

		// Show scroll indicator
		if len(m.entries) > m.height {
			scrollPos := float64(m.selectedIdx) / float64(len(m.entries)-1)
			b.WriteString("\n")
			b.WriteString(ui.HelpStyle.Render(
				fmt.Sprintf("Showing %d-%d of %d (%.0f%%)",
					start+1, end, len(m.entries), scrollPos*100),
			))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(ui.RenderHelp(
		"↑/↓", "navigate",
		"Enter", "select/open",
		"Backspace", "parent dir",
		"~", "home",
		"/", "root",
		".", "toggle hidden",
		"Esc", "cancel",
	))

	return b.String()
}

// GetSelectedPath returns the currently selected file/directory path
func (m FileSelectModel) GetSelectedPath() string {
	if len(m.entries) == 0 || m.selectedIdx >= len(m.entries) {
		return ""
	}
	return filepath.Join(m.currentPath, m.entries[m.selectedIdx].Name())
}

// SetFilter sets file extension filters (e.g., []string{".pdf", ".txt"})
func (m *FileSelectModel) SetFilter(extensions []string) {
	m.filterExt = extensions
	m.loadDirectory()
}

// SetShowHidden sets whether to show hidden files
func (m *FileSelectModel) SetShowHidden(show bool) {
	m.showHidden = show
	m.loadDirectory()
}

// loadDirectory loads the current directory contents
func (m *FileSelectModel) loadDirectory() {
	entries, err := os.ReadDir(m.currentPath)
	if err != nil {
		m.err = err
		m.entries = []fs.DirEntry{}
		return
	}

	m.err = nil

	// Filter entries
	filtered := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()

		// Filter hidden files
		if !m.showHidden && strings.HasPrefix(name, ".") {
			continue
		}

		// Filter by extension (only for files)
		if !entry.IsDir() && len(m.filterExt) > 0 {
			hasValidExt := false
			for _, ext := range m.filterExt {
				if strings.HasSuffix(strings.ToLower(name), strings.ToLower(ext)) {
					hasValidExt = true
					break
				}
			}
			if !hasValidExt {
				continue
			}
		}

		filtered = append(filtered, entry)
	}

	// Sort: directories first, then files, alphabetically
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].IsDir() && !filtered[j].IsDir() {
			return true
		}
		if !filtered[i].IsDir() && filtered[j].IsDir() {
			return false
		}
		return strings.ToLower(filtered[i].Name()) < strings.ToLower(filtered[j].Name())
	})

	m.entries = filtered
}

// FileSelectedMsg is sent when a file is selected
type FileSelectedMsg struct {
	Path string
}
