package remote

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"
)

var KeyEnvName = "BACKEND_CRYPT_AES256_32_BYTE_HEX_KEY"

func hasKey() bool {
	keyString := os.Getenv(KeyEnvName)
	return keyString != ""
}

func readKeyFromEnv() ([]byte, error) {
	keyString := os.Getenv(KeyEnvName)

	validator := regexp.MustCompile("^[0-9a-f]{64}$")
	if !validator.MatchString(keyString) {
		return []byte{}, fmt.Errorf("key was not a hex string with 32 bytes")
	}

	key, err := hex.DecodeString(keyString)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to decode key, not a valid hex string")
	}
	return key, nil
}

//  determine if data (which is a []byte containing a json structure) is encrypted
//         (that is, of the form: {"crypted":"<hex containing iv and payload>"})
func isEncrypted(data []byte) bool {
	validator := regexp.MustCompile(`^{"crypted":"[0-9a-f]+"}$`)
	return validator.Match(data)
}

// decrypt the hex-encoded contents of data, which is expected to be of the form
//         {"crypted":"<hex containing iv and payload>"}
func decrypt(jsonCryptedData []byte) ([]byte, error) {
	key, err := readKeyFromEnv()
	if err != nil {
		return []byte{}, err
	}

	// extract the hex part only, cutting off {"crypted":" (12 characters) and "} (2 characters)
	src := jsonCryptedData[12:len(jsonCryptedData)-2]

	ciphertext := make([]byte, hex.DecodedLen(len(src)))
	n, err := hex.Decode(ciphertext, src)
	if err != nil {
		return []byte{}, err
	}
	if n != len(src) {
		return []byte{}, fmt.Errorf("did not fully decode, only read %d characters before encountering an error", n)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to create cipher implementation: %v", err.Error())
	}

	if len(ciphertext) < aes.BlockSize {
		return []byte{}, fmt.Errorf("ciphertext too short, did not contain initial vector")
	}
	iv := ciphertext[:aes.BlockSize]
	payload := ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(payload, payload)

	// payload is now decrypted
	return payload, nil
}

//   encrypt data (which is a []byte containing a json structure)
//      into a json structure {"crypted":"<hex-encoded random iv + hex-encoded CFB encrypted data>"}
func encrypt(payload []byte) ([]byte, error) {
	key, err := readKeyFromEnv()
	if err != nil {
		return payload, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte{}, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(payload))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return []byte{}, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], payload)

	// TODO do we need this here?
	// It's important to remember that ciphertexts must be authenticated
	// (i.e. by using crypto/hmac) as well as being encrypted in order to
	// be secure.

	prefix := []byte(`{"crypted":"`)
	postfix := []byte(`"}`)
	encryptedHex := make([]byte, hex.EncodedLen(len(ciphertext)))
	_ = hex.Encode(encryptedHex, ciphertext)

	jsonCryptedData := append(append(prefix, encryptedHex...), postfix...)
	return jsonCryptedData, nil
}

func possiblyDecrypt(data []byte) ([]byte, error) {
	if hasKey() && isEncrypted(data) {
		return decrypt(data)
	} else {
		return data, nil
	}
}

func possiblyEncrypt(data []byte) ([]byte, error) {
	if hasKey() {
		return encrypt(data)
	} else {
		return data, nil
	}
}
