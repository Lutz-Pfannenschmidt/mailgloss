package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"mailgloss/logger"
	"mailgloss/models"
)

func main() {
	// Initialize logger
	if err := logger.Init(); err != nil {
		fmt.Printf("Warning: Failed to initialize logger: %v\n", err)
		// Continue without logging rather than failing
	} else {
		logger.Info("Starting mailgloss")
	}

	// Create app model
	m, err := models.NewAppModel()
	if err != nil {
		logger.Error("Failed to initialize application", "error", err)
		fmt.Printf("Error initializing application: %v\n", err)
		os.Exit(1)
	}

	// Create program with alternate screen
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Run the program
	if _, err := p.Run(); err != nil {
		logger.Error("Failed to run application", "error", err)
		fmt.Printf("Error running application: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Mailgloss exited normally")
}
