package main

import (
	"crypto/rand"
	"file-encrypter/cmd/state"
	"os"
	"path/filepath"
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

	}
	return key, nil
}

func TestEncryption(t *testing.T) {
	testState := state.Crypter{}

	key, err := testGeneratingKey(t)
	if err != nil {
		t.Fatalf("failed generating key with error: %s", err)
	}

	testState.Key = &key

	tempDir := os.TempDir()
	inputFile := filepath.Join(tempDir, "example.txt")
	originalContent := []byte("Yo yo this is a test")
	if err := os.WriteFile(inputFile, originalContent, 0644); err != nil {
		t.Fatalf("failed to write to input file: %s", err)
	}
	// command := state.ENCRYPTING
	// testState.Command = &command
	if err := testState.StartEncrypting(inputFile); err != nil {
		t.Fatalf("failed starting ecryption with error: %s", err)
	}

	// command = state.DECRYPTING
	// testState.Command = &command
	if err := testState.StartDecrypting(inputFile); err != nil {
		t.Fatalf("failed starting decryption with error: %s", err)
	}

	file, err := os.ReadFile(inputFile)
	if err != nil {
		t.Fatalf("failed reading file in tmp directory error: %s", err)
	}
	if string(originalContent) != string(file) {
		t.Fatalf("Encrypting and Decrypting failed, does not match original content")
	}
}
