//Original source: https://github.com/gokyle/marchat

package signature

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

const KeySize = 16

var (
	ErrPadding       = fmt.Errorf("invalid padding")
	ErrRandomFailure = fmt.Errorf("failed to read enough random data")
	ErrInvalidIV     = fmt.Errorf("invalid IV")
)

func Decrypt(key string, encryptedMessage []byte) ([]byte, error) {
	cypher, err := aes.NewCipher(getEncryptionKey(key))
	if err != nil {
		return nil, err
	}

	clonedEncryptedMessage := make([]byte, len(encryptedMessage))
	copy(clonedEncryptedMessage, encryptedMessage)

	if len(clonedEncryptedMessage) < aes.BlockSize {
		return nil, ErrInvalidIV
	}

	iv := clonedEncryptedMessage[:aes.BlockSize]
	message := clonedEncryptedMessage[aes.BlockSize:]
	cbc := cipher.NewCBCDecrypter(cypher, iv)
	cbc.CryptBlocks(message, message)
	return unpadBuffer(message)
}

func DigestBytes(message []byte) []byte {
	hasher := sha256.New()
	hasher.Write(message)
	hashedKey := hasher.Sum(nil)

	return hashedKey
}

func Encrypt(key string, message []byte) ([]byte, error) {
	cypher, err := aes.NewCipher(getEncryptionKey(key))
	if err != nil {
		return nil, err
	}

	iv, err := generateIV()
	if err != nil {
		return nil, err
	}

	paddedMessage, err := padBuffer(message)
	if err != nil {
		return nil, err
	}

	cbc := cipher.NewCBCEncrypter(cypher, iv)
	cbc.CryptBlocks(paddedMessage, paddedMessage)
	encryptedMessage := append(iv, paddedMessage...)

	return encryptedMessage, nil
}

func getEncryptionKey(key string) []byte {
	hasher := sha256.New()
	io.WriteString(hasher, key)
	hashedKey := hasher.Sum(nil)
	return []byte(hashedKey)[:16]
}

func generateIV() ([]byte, error) {
	return random(aes.BlockSize)
}

func random(size int) ([]byte, error) {
	randomBytes := make([]byte, size)
	n, err := rand.Read(randomBytes)
	if err != nil {
		return []byte{}, err
	} else if size != n {
		err = ErrRandomFailure
	}
	return randomBytes, err
}

func padBuffer(message []byte) ([]byte, error) {
	messageLen := len(message)

	paddedMessage := make([]byte, messageLen)
	copy(paddedMessage, message)

	if len(paddedMessage) != messageLen {
		return paddedMessage, ErrPadding
	}

	bytesToPad := aes.BlockSize - messageLen%aes.BlockSize

	paddedMessage = append(paddedMessage, 0x80)
	for i := 1; i < bytesToPad; i++ {
		paddedMessage = append(paddedMessage, 0x0)
	}
	return paddedMessage, nil
}

func unpadBuffer(paddedMessage []byte) ([]byte, error) {
	message := paddedMessage
	var paddedMessageLen int
	origLen := len(message)

	for paddedMessageLen = origLen - 1; paddedMessageLen >= 0; paddedMessageLen-- {
		if message[paddedMessageLen] == 0x80 {
			break
		}

		if message[paddedMessageLen] != 0x0 || (origLen-paddedMessageLen) > aes.BlockSize {
			err := ErrPadding
			return nil, err
		}
	}
	message = message[:paddedMessageLen]
	return message, nil
}
