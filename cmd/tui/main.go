package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "AI SaaS Lab server base URL")
	apiKey := flag.String("key", "demo-key-pro", "API Key for authentication")
	flag.Parse()

	client := NewAPIClient(*serverURL)
	model := NewModel(client, *apiKey)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI client: %v\n", err)
		os.Exit(1)
	}
}
