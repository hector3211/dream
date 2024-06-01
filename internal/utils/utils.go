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

const MagicHeader = "ENCRYPTED"

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
			// TODO: filePerm := 0400 only owner can read but not write
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

func IsFileEncrypted(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("failed %s", err.Error())
	}
	defer file.Close()

	header := make([]byte, len(MagicHeader))
	_, err = file.Read(header)
	if err != nil {
		return false, fmt.Errorf("failed %s", err.Error())
	}

	return string(header) == MagicHeader, nil
}

func StartEncrypting(key []byte, filePath string) tea.Cmd {
	return func() tea.Msg {
		isEncrypted, _ := IsFileEncrypted(filePath)
		if isEncrypted {
			return Successmsg{Message: "File is already encrypted."}
		}
		// TODO: put key lenght in .env
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
		content := fmt.Sprintf("%s\n%s", MagicHeader, cipherText)

		// encryptionFileTag := ".enc"
		// currFilePath := strings.Split(filePath, ".")
		// newFilePath := fmt.Sprintf("%s%s", currFilePath[0], encryptionFileTag)
		//
		// Make new encrypted file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return ErrMsg{fmt.Errorf("failed writing file: %s", err.Error())}
		}

		// if err := os.Rename(filePath, fmt.Sprintf("%s.enc", filePath)); err != nil {
		// 	return ErrMsg{fmt.Errorf("failed removing old dir: %s", err.Error())}
		// }

		return Successmsg{"Successfully Encrypted file!"}
	}
}

func StartDecrypting(key []byte, filePath string) tea.Cmd {
	return func() tea.Msg {
		isEncrypted, err := IsFileEncrypted(filePath)
		if !isEncrypted && err == nil {
			return Successmsg{Message: "Cannot decrypt a file that's not encrypted."}
		}
		fileContents, err := os.ReadFile(filePath)
		if err != nil {
			return ErrMsg{errors.New("failed reading file")}
		}

		splitFileContents := strings.Split(string(fileContents), "\n")
		cipherText := []byte(splitFileContents[1])

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

		return Successmsg{"Successfully decrypted file!"}
	}
}
