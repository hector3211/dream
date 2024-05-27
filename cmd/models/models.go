package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
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

type MainModel struct {
	State      State
	Command    Command
	FilePicker filepicker.Model
	FilePath   string
	Key        *[]byte
	Quitting   bool
	Err        error
	FileView   FileModel
}
type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func InitializeMainModel() MainModel {
	fp := filepicker.New()
	fp.AllowedTypes = []string{".mod", ".sum", ".go", ".txt", ".md"}
	fp.CurrentDirectory, _ = os.Getwd()
	return MainModel{
		State:      MainView,
		Command:    PENDING,
		FilePicker: fp,
		FileView:   InitializeFileModel(),
	}
}

func (m MainModel) Init() tea.Cmd {
	err := m.GenerateKey()
	if err != nil {
		m.Err = err
		return tea.Printf("failed generating key")
	}
	m.FileView.Init()
	return m.FilePicker.Init()
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
			if m.State == FileView {
				if len(m.FilePath) > 0 {
					m.Command = ENCRYPTING
					// then start encrypting
					if err := m.StartEncrypting(); err != nil {
						m.Err = errors.New("Failed encrypting")
						return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
					}
					//
					// cmd := m.StartEncrypting()
					// return m, cmd
				}
			}
		case "d":
			if m.State == FileView {
				if len(m.FilePath) > 0 {
					m.Command = DECRYPTING
					// then start encrypting
					if err := m.StartDecrypting(); err != nil {
						m.Err = errors.New("Failed decrypting")
						return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
					}
					// cmd := m.StartDecrypting()
					// return m, cmd
				}
			}
		}
	case clearErrorMsg:
		m.Err = nil
	}

	m.FilePicker, cmd = m.FilePicker.Update(msg)
	//
	// Did the user select a file?
	if didSelect, path := m.FilePicker.DidSelectFile(msg); didSelect {
		// Get the path of the selected file.
		m.FilePath = path
		pathSlice := strings.Split(path, ".")
		tag := pathSlice[len(pathSlice)-1]
		m.FileView.AddFileContents(path, tag)
		m.State = FileView
		return m, tea.Batch(cmd, tea.ClearScreen)
	}
	//
	// Did the user select a disabled file?
	// This is only necessary to display an error to the user.
	if didSelect, path := m.FilePicker.DidSelectDisabledFile(msg); didSelect {
		// Let's clear the selectedFile and display an error.
		m.Err = errors.New(path + " is not valid.")
		m.FilePath = ""
		return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
	}

	return m, cmd
}

func (m MainModel) View() string {
	switch m.State {
	case MainView:
		if m.Quitting {
			return ""
		}

		s := "\n  "
		s += fmt.Sprintf("key %v", m.Key)
		s += fmt.Sprintf("%s", m.FilePath)
		s += fmt.Sprintf("\n %s \n", m.FilePicker.FileSelected)
		if m.Err != nil {
			s += fmt.Sprintf("%v", m.FilePicker.Styles.DisabledFile.Render(m.Err.Error()))
			s += fmt.Sprintf("%v", m.Err.Error())
		} else if m.FilePath == "" {
			s += fmt.Sprintf("Command: %v Key: %s", m.Command, m.Key)
			s += "Pick a file:"
		} else {
			s += fmt.Sprintf("Selected file: %s", m.FilePicker.Styles.Selected.Render(m.FilePath))
		}
		s += fmt.Sprintf("\n\n %s\n", m.FilePicker.View())
		return s
	case FileView:
		s := fmt.Sprintf("Command %v\n\n", m.Command)
		s += m.FileView.View()
		return s
	}
	return "Nothing"
}

func (m *MainModel) GenerateKey() error {
	// check if user has a .drm file with encrypted key in it
	// TODO: make key file name an env variable
	keyFileName := "key.drm"
	key, keyFileErr := os.ReadFile(keyFileName)
	if keyFileErr != nil {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		if err != nil {
			fmt.Printf("failed creating a key with error: %s", err)
			return err
		}

		// create a file storing the key
		if err = os.WriteFile(keyFileName, key, 0644); err != nil {
			fmt.Printf("failed writing saving key to file .drm with error: %s", err)
			return err
		}

		m.Key = &key
		return nil
	}

	trimedKey := strings.TrimSpace(string(key))
	keyInBytes := []byte(trimedKey)
	m.Key = &keyInBytes
	return nil
}
func (m *MainModel) StartEncrypting() error {
	// if f.CommandState != DECRYPTING {
	// 	return fmt.Errorf("command was not 'Decrypting")
	// }
	// tmpFile := "data.txt"
	if m.Key == nil {
		return errors.New("Key is nil")
	}
	cipherText, err := os.ReadFile(m.FilePath)
	if err != nil {
		return errors.New("failed reading file")
	}

	block, err := aes.NewCipher(*m.Key)
	if err != nil {
		return errors.New("failed creating block")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return errors.New("failed creating gcm")
	}

	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return errors.New("ciphertext too short")
	}

	nonce, cicipherText := cipherText[:nonceSize], cipherText[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, cicipherText, nil)
	if err != nil {
		return errors.New("failed decoding cipher")
	}

	if err := os.WriteFile(m.FilePath, plainText, 0644); err != nil {
		return errors.New("failed writing encrypted file")
	}

	return nil
}

func (m *MainModel) StartDecrypting() error {
	// if f.CommandState != DECRYPTING {
	// 	return fmt.Errorf("command was not 'Decrypting")
	// }
	if m.Key == nil {
		return errors.New("Key is nil")
	}
	cipherText, err := os.ReadFile(m.FilePath)
	if err != nil {
		return errors.New("failed reading file")
	}

	block, err := aes.NewCipher(*m.Key)
	if err != nil {
		return fmt.Errorf("failed creating block with error %s", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return errors.New("failed")
	}

	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return errors.New("ciphertext too short")
	}

	nonce, cicipherText := cipherText[:nonceSize], cipherText[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, cicipherText, nil)
	if err != nil {
		return fmt.Errorf("failed decoding cipher with error %s", err)
	}

	if err := os.WriteFile(m.FilePath, plainText, 0644); err != nil {
		return errors.New("Fialed writing decryption")
	}

	return errors.New("Successfully decrypted file")
}

//---------------------------------------------------------------------------

func (f *FileModel) ReadFileContents() string {
	contents, _ := os.ReadFile(f.FilePath)
	return string(contents)
}

func InitializeFileModel() FileModel {
	return FileModel{FilePath: "", Tag: ""}
}

func (f FileModel) Init() tea.Cmd {
	return nil
}

func (f *FileModel) AddFileContents(path, tag string) {
	f.FilePath = path
	f.Tag = tag
}

func (f FileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl-c":
			return f, tea.Quit
		}
	}
	return f, tea.Batch(cmds...)
}

func (f FileModel) View() string {
	s := "File View Model\n\n"

	s += fmt.Sprintf("📄 FilePath: %s  🏷️: %s\n\n", f.FilePath, f.Tag)

	s += fmt.Sprintf("\n %s", f.ReadFileContents())

	// s += fmt.Sprintf("\n %s \n", f.Progress.View())
	s += fmt.Sprintf("\n\n [b] 🔙  [e] to Encrypt file [d] to Decrypt file or [q] to Quit")
	return s

}
