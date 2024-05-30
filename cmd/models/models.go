package models

import (
	"errors"
	"fmt"
	"os"
	"time"

	"file-encrypter/internal/utils"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type State int

const (
	MainView State = iota
	FileView
)

type Command int

const (
	PENDING Command = iota
	ENCRYPTING
	DECRYPTING
	DONE
)

type MainModel struct {
	State             State
	Command           Command
	Key               []byte
	FilePicker        filepicker.Model
	FilePath          string
	FileContents      string
	FileContentStyles lipgloss.Style
	HelpMenu          help.Model
	Quitting          bool
	Err               error
	Message           string
}

type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Encrypt   key.Binding
	Decrypt   key.Binding
	FilesMenu key.Binding
	Help      key.Binding
}

var keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("⬆️/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("⬇️/j", "move down"),
	),
	Encrypt: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("🔒/e", "encrypt"),
	),
	Decrypt: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("🔓/d", "decrypt"),
	),
	FilesMenu: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("🔙/b", "go back"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
}

func InitializeMainModel() MainModel {
	fp := filepicker.New()
	// fp.AllowedTypes = []string{".mod", ".sum", ".go", ".txt", ".md", ".drm"}
	fp.AllowedTypes = []string{".txt", ".md"}
	fp.CurrentDirectory, _ = os.Getwd()
	return MainModel{
		State:             MainView,
		Command:           PENDING,
		FilePicker:        fp,
		FileContents:      "",
		FileContentStyles: lipgloss.NewStyle().Foreground(lipgloss.Color("202")).Border(lipgloss.RoundedBorder()).Padding(2),
		HelpMenu:          help.New(),
		Key:               make([]byte, 0),
		// FileView:   InitializeFileModel(),
	}
}

func (m MainModel) Init() tea.Cmd {
	m.ReadFileContents()
	return tea.Batch(m.FilePicker.Init(), utils.GenerateKey())
}
func (m *MainModel) ReadFileContents() {
	contents, _ := os.ReadFile(m.FilePath)
	m.FileContents = string(contents)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit
		case "b":
			if m.State == FileView {
				m.Command = PENDING
				m.State = MainView
			}
		case "e":
			if len(m.FilePath) > 0 && len(m.Key) > 0 {
				// then start encrypting
				m.Command = ENCRYPTING
				return m, utils.StartEncrypting(m.Key, m.FilePath)
			}
		case "d":
			if len(m.FilePath) > 0 && len(m.Key) > 0 {
				// then start decrypting
				m.Command = DECRYPTING
				return m, utils.StartDecrypting(m.Key, m.FilePath)
			}
		}
	case utils.KeyMsg:
		m.Key = msg.Value
		return m, nil
	case utils.Successmsg:
		m.State = MainView
		m.Message = msg.Message
		return m, tea.Batch(cmd, utils.ClearMessageAfter(5*time.Second))
	case utils.ClearMessage:
		m.Message = ""
	case utils.ErrMsg:
		m.Err = fmt.Errorf("%s", msg.Error.Error())
		return m, tea.Batch(cmd, utils.ClearErrorAfter(5*time.Second))
	case utils.ClearErrorMsg:
		m.Err = nil
	}

	m.FilePicker, cmd = m.FilePicker.Update(msg)
	//
	// Did the user select a file?
	if didSelect, path := m.FilePicker.DidSelectFile(msg); didSelect {
		m.FilePath = path
		m.State = FileView
		m.ReadFileContents()
		return m, tea.Batch(cmd, tea.ClearScreen)
	}

	// Did the user select a disabled file?
	// This is only necessary to display an error to the user.
	if didSelect, path := m.FilePicker.DidSelectDisabledFile(msg); didSelect {
		// Let's clear the selectedFile and display an error.
		m.Err = errors.New(path + " is not valid.")
		m.FilePath = ""
		return m, tea.Batch(cmd, utils.ClearErrorAfter(2*time.Second))
	}

	return m, tea.Batch(cmd)
}

func (m MainModel) View() string {
	switch m.State {
	case MainView:
		if m.Quitting {
			return ""
		}
		s := "\n  "
		s += fmt.Sprintln(m.Message)
		// s += fmt.Sprintf("\n key: %s\n", string(m.Key))
		if m.Err != nil {
			s += fmt.Sprintf("%v", m.FilePicker.Styles.DisabledFile.Render(m.Err.Error()))
			s += fmt.Sprintf("\nGetting ERROR: %s\n", m.Err.Error())
		} else if m.FilePath == "" {
			s += "Pick a file:"
		} else {
			s += fmt.Sprintf("Selected file: %s\n", m.FilePicker.Styles.Selected.Render(m.FilePath))
			// s += fmt.Sprintf("File Path is.. :%s", m.FilePath)
		}
		s += fmt.Sprintf("\n\n %s\n", m.FilePicker.View())
		return s
	case FileView:
		// s := fmt.Sprintf("Command %v\n", m.Command)
		// s := fmt.Sprintf("\n key: %s\n", string(m.Key))
		s := "\n"
		if m.Err != nil {
			s += fmt.Sprintf("%v", m.FilePicker.Styles.DisabledFile.Render(m.Err.Error()))
			s += fmt.Sprintf("\nGetting ERROR: %v\n", m.Err.Error())
		}
		s += fmt.Sprintf("%s\n\n", m.Message)
		s += fmt.Sprintf("\n%s", m.FileContentStyles.Render(m.FileContents))
		s += fmt.Sprintf("\n\n[b] 🔙  [e] to Encrypt file [d] to Decrypt file or [q] to Quit\n")
		return s
	}
	return "Nothing"
}
