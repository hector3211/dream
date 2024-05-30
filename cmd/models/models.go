package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
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

type Choice struct {
	Name string
}

type FileModel struct {
	FilePath string
	Tag      string
}

type Key struct {
	value []byte
}

type (
	ErrMsg        struct{ error }
	Successmsg    struct{ string }
	clearErrorMsg struct{}
	clearMessage  struct{}
	KeyMsg        Key
)

type MainModel struct {
	State        State
	Command      Command
	Key          []byte
	FilePicker   filepicker.Model
	FilePath     string
	FileContents string
	Quitting     bool
	Err          error
	Message      string
	// FileView   FileModel
}

func clearMessageAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearMessage{}
	})
}
func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func InitializeMainModel() MainModel {
	fp := filepicker.New()
	fp.AllowedTypes = []string{".mod", ".sum", ".go", ".txt", ".md", ".drm"}
	fp.CurrentDirectory, _ = os.Getwd()
	return MainModel{
		State:        MainView,
		Command:      PENDING,
		FilePicker:   fp,
		FileContents: "",
		Key:          make([]byte, 0),
		// FileView:   InitializeFileModel(),
	}
}

func (m MainModel) Init() tea.Cmd {
	m.ReadFileContents()
	return tea.Batch(m.FilePicker.Init(), GenerateKey())
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
				return m, StartEncrypting(m.Key, m.FilePath)
			}
		case "d":
			if len(m.FilePath) > 0 && len(m.Key) > 0 {
				// then start decrypting
				m.Command = DECRYPTING
				return m, StartDecrypting(m.Key, m.FilePath)
			}
		}
	case KeyMsg:
		m.Key = msg.value
		return m, nil
	case Successmsg:
		m.State = MainView
		m.Message = msg.string
		return m, tea.Batch(cmd, clearMessageAfter(5*time.Second))
	case clearMessage:
		m.Message = ""
	case ErrMsg:
		m.Err = fmt.Errorf("%s", msg.error.Error())
		return m, tea.Batch(cmd, clearErrorAfter(5*time.Second))
	case clearErrorMsg:
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
		return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
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
		// s += fmt.Sprintf("\n key: %s\n", string(m.Key))
		if m.Err != nil {
			s += fmt.Sprintf("%v", m.FilePicker.Styles.DisabledFile.Render(m.Err.Error()))
			s += fmt.Sprintf("\nGetting ERROR: %s\n", m.Err.Error())
		} else if m.FilePath == "" {
			s += "Pick a file:"
		} else {
			s += fmt.Sprintf("Selected file: %s\n", m.FilePicker.Styles.Selected.Render(m.FilePath))
			// s += fmt.Sprintf("File Path is.. :%s", m.FilePath)
			// s += fmt.Sprintf("\n Permissions: %v\n", m.FilePicker.ShowPermissions)
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
		s += fmt.Sprintf("\n\n📄: %s\n", m.FileContents)
		s += fmt.Sprintf("\n\n[b] 🔙  [e] to Encrypt file [d] to Decrypt file or [q] to Quit\n")
		return s
	}
	return "Nothing"
}

func GenerateKey() tea.Cmd {
	// check if user has a .drm file with encrypted key in it
	// TODO make key file name an env variable
	return func() tea.Msg {
		keyFileName := "key.drm"
		key, keyFileErr := os.ReadFile(keyFileName)
		if keyFileErr != nil {
			key := make([]byte, 32)
			_, err := rand.Read(key)
			if err != nil {
				return ErrMsg{errors.New("failed creating new random key")}
			}

			// create a file storing the key
			if err = os.WriteFile(keyFileName, key, 0644); err != nil {
				return ErrMsg{errors.New("failed writing to file for new key")}
			}

			return KeyMsg{value: key}
		}

		trimedKey := strings.TrimSpace(string(key))
		keyInBytes := []byte(trimedKey)
		return KeyMsg{value: keyInBytes}
	}
}

func StartEncrypting(key []byte, filePath string) tea.Cmd {
	return func() tea.Msg {
		keyLength := len(key)
		if keyLength != 32 {
			return ErrMsg{fmt.Errorf("Invalid key length: %d", keyLength)}
		}

		fileContents, err := os.ReadFile(filePath)
		if err != nil {
			return ErrMsg{fmt.Errorf("failed reading file")}
		}

		block, err := aes.NewCipher(key)
		if err != nil {
			return ErrMsg{fmt.Errorf("failed creating cypher")}
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return ErrMsg{fmt.Errorf("failed creating gcm")}
		}

		nonce := make([]byte, gcm.NonceSize())
		if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
			return ErrMsg{fmt.Errorf("failed io reader: %s", err.Error())}
		}

		cipherText := gcm.Seal(nonce, nonce, fileContents, nil)

		if err := os.WriteFile(filePath, cipherText, 0644); err != nil {
			return ErrMsg{fmt.Errorf("failed writing file: %s", err.Error())}
		}
		return Successmsg{"Successfully Encrypted file!"}
	}
}

func StartDecrypting(key []byte, filePath string) tea.Cmd {
	// if f.CommandState != DECRYPTING {
	// 	return fmt.Errorf("command was not 'Decrypting")
	// }
	return func() tea.Msg {
		cipherText, err := os.ReadFile(filePath)
		if err != nil {
			return ErrMsg{errors.New("failed reading file")}
		}

		block, err := aes.NewCipher(key)
		if err != nil {
			return ErrMsg{fmt.Errorf("failed creating block with error %s", err)}
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return ErrMsg{errors.New("failed")}
		}

		nonceSize := gcm.NonceSize()
		if len(cipherText) < nonceSize {
			return ErrMsg{errors.New("ciphertext too short")}
		}

		nonce, cipherText := cipherText[:nonceSize], cipherText[nonceSize:]
		plainText, err := gcm.Open(nil, nonce, cipherText, nil)
		if err != nil {
			return ErrMsg{fmt.Errorf("failed decoding cipher with error %s", err)}
		}

		if err := os.WriteFile(filePath, plainText, 0644); err != nil {
			return ErrMsg{errors.New("Fialed writing decryption")}
		}

		return Successmsg{"Successfully decrypted file!!!"}
	}
}

//---------------------------------------------------------------------------

// func (f *FileModel) ReadFileContents() string {
// 	contents, _ := os.ReadFile(f.FilePath)
// 	return string(contents)
// }
//
// func InitializeFileModel() FileModel {
// 	return FileModel{FilePath: "", Tag: ""}
// }
//
// func (f FileModel) Init() tea.Cmd {
// 	return nil
// }
//
// func (f *FileModel) AddFileContents(path, tag string) {
// 	f.FilePath = path
// 	f.Tag = tag
// }
//
// func (f FileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
// 	var cmds []tea.Cmd
// 	switch msg := msg.(type) {
// 	case tea.KeyMsg:
// 		switch msg.String() {
// 		case "q", "ctrl-c":
// 			return f, tea.Quit
// 		}
// 	}
// 	return f, tea.Batch(cmds...)
// }
//
// func (f FileModel) View() string {
// 	s := "File View Model\n\n"
//
// 	s += fmt.Sprintf("📄 FilePath: %s  🏷️: %s\n\n", f.FilePath, f.Tag)
//
// 	s += fmt.Sprintf("\n %s", f.ReadFileContents())
//
// 	// s += fmt.Sprintf("\n %s \n", f.Progress.View())
// 	s += fmt.Sprintf("\n\n [b] 🔙  [e] to Encrypt file [d] to Decrypt file or [q] to Quit")
// 	return s
//
// }
