package main

import (
	"file-encrypter/cmd/models"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(models.InitializeMainModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("bubble tea failed with error: %s", err)
		os.Exit(1)
	}

}
