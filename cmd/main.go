package main

import (
	// "bufio"
	// "file-encrypter/cmd/state"
	"file-encrypter/cmd/models"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	// "strings"
)

func main() {
	// state := state.State{
	// 	Files:   make([]string, 0),
	// 	Status:  state.WAITING,
	// 	Choices: []string{"encrypt, decrypt"},
	// }
	//
	// state.GenerateKey()

	p := tea.NewProgram(models.InitializeMainModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("bubble tea failed with error: %s", err)
		os.Exit(1)
	}

	// for true {
	// 	fmt.Println("Do you want to [e] for encrypt or [d] for decrypt or [q] to quit")
	// 	reader := bufio.NewReader(os.Stdin)
	// 	choice, err := reader.ReadString('\n')
	// 	choice = strings.TrimSpace(choice)
	// 	result, err := state.ParseChoice(choice)
	// 	if err != nil {
	// 		fmt.Println("user quit")
	// 		break
	// 	}
	//
	// 	fmt.Printf("Please enter a file name to %s: ", result)
	// 	fileReader := bufio.NewReader(os.Stdin)
	// 	command, err := fileReader.ReadString('\n')
	// 	if err != nil {
	// 		fmt.Printf("failed reading from standard in with error: %s", err)
	// 	}
	//
	// 	state.ParseCommand(command)
	//
	// }
}
