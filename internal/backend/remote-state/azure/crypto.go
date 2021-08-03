package azure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const nonceSize int = 12

func encrypt(key []byte, plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}

	dst := make([]byte, 0, nonceSize+((len(plain)+len(key)-1)/len(key))*len(key))
	dst = append(dst, nonce...)
	encoded := aesgcm.Seal(dst, nonce, plain, nil)
	return encoded, nil
}

func decrypt(key []byte, encoded []byte) ([]byte, error) {
	if len(encoded) < nonceSize {
		return nil, fmt.Errorf("not enough data to read nonce")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plain, err := aesgcm.Open(nil, encoded[:nonceSize], encoded[nonceSize:], nil)
	if err != nil {
		return nil, err
	}

	return plain, nil
}
