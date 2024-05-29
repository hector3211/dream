package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testGeneratingKey(t *testing.T) ([]byte, error) {
	tempDir := os.TempDir()
	keyFileName := filepath.Join(tempDir, "key.drm")
	key, keyFileErr := os.ReadFile(keyFileName)
	if keyFileErr != nil {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		if err != nil {
			t.Fatalf("failed creating a key with error: %s", err)
			return nil, err
		}

		// create a file storing the key
		if err = os.WriteFile(keyFileName, key, 0644); err != nil {
			t.Fatalf("failed writing saving key to file .drm with error: %s", err)
			return nil, err
		}
		return key, nil

	}

	trimedKey := strings.TrimSpace(string(key))
	keyInBytes := []byte(trimedKey)
	return keyInBytes, nil
}

func TestEncryption(t *testing.T) {

	key, err := testGeneratingKey(t)
	if err != nil && len(key) == 0 {
		t.Fatalf("failed generating key with error: %s", err)
	}

	tempDir := os.TempDir()
	inputFile := filepath.Join(tempDir, "example.txt")
	originalContent := []byte("Yo yo this is a test")
	if err := os.WriteFile(inputFile, originalContent, 0644); err != nil {
		t.Fatalf("failed to write to input file: %s", err)
	}
	// command := state.ENCRYPTING
	// testState.Command = &command
	keyLength := len(key)
	if keyLength != 32 {
		t.Logf("Invalid key length: %d", keyLength)
	}
	fileContents, err := os.ReadFile(inputFile)
	if err != nil {
		t.Fatalf("failed reading file")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("failed creating cypher")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("failed creating gcm")
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		t.Errorf("io rader failed")
	}

	cipherText := gcm.Seal(nonce, nonce, fileContents, nil)

	if err := os.WriteFile(inputFile, cipherText, 0644); err != nil {
		t.Fatalf("failed writing file: %s", err.Error())
	}

	// command = state.DECRYPTING
	// testState.Command = &command
	if err := testDecryption(inputFile, key, t); err != nil {
		t.Fatalf("%s", err)
	}

	theTruthFile, err := os.ReadFile(inputFile)
	if err != nil {
		t.Fatalf("failed reading file in tmp directory error: %s", err)
	}
	if string(originalContent) != string(theTruthFile) {
		t.Fatalf("Encrypting and Decrypting failed, does not match original content")
	}
}

func testDecryption(filePath string, key []byte, t *testing.T) error {
	fileContents, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed reading file")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("failed creating block with error %s", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("failed gcm")
	}

	nonceSize := gcm.NonceSize()
	if len(fileContents) < nonceSize {
		t.Fatalf("ciphertext too short")
	}

	nonce, cipherText := fileContents[:nonceSize], fileContents[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		t.Fatalf("failed decoding cipher with error %s", err)
	}

	if err := os.WriteFile(filePath, plainText, 0644); err != nil {
		t.Fatalf("Fialed writing decryption")
	}
	return nil
}
