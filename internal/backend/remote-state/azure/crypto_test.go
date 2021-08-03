package azure

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"reflect"
	"testing"
)

func TestEncryptionShort(t *testing.T) {
	key, _ := hex.DecodeString("0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF")
	plaintext := []byte("Hello world")
	ciphertext, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Error encrypting plaintext: %q", err)
	}

	decrypted, err := decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Error decrypting ciphertext: %q", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("Expected cipher: %s, got: %s", plaintext, decrypted)
	}
}

func TestEncryptionLarge(t *testing.T) {
	key, _ := hex.DecodeString("0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF")
	plaintext := make([]byte, 10000)
	if _, err := io.ReadFull(rand.Reader, plaintext); err != nil {
		t.Fatalf("Error getting random data: %q", err)
	}

	ciphertext, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Error encrypting plaintext: %q", err)
	}

	decrypted, err := decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Error decrypting ciphertext: %q", err)
	}

	if !reflect.DeepEqual(decrypted, plaintext) {
		t.Fatal("Decrypted data isn't equal to the original data")
	}
}
