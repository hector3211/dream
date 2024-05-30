package utils

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

	tea "github.com/charmbracelet/bubbletea"
)

type (
	ErrMsg        struct{ Error error }
	Successmsg    struct{ Message string }
	ClearErrorMsg struct{}
	ClearMessage  struct{}
	KeyMsg        struct{ Value []byte }
)

func ClearMessageAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return ClearMessage{}
	})
}
func ClearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return ClearErrorMsg{}
	})
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

			return KeyMsg{Value: key}
		}

		trimedKey := strings.TrimSpace(string(key))
		keyInBytes := []byte(trimedKey)
		return KeyMsg{Value: keyInBytes}
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
