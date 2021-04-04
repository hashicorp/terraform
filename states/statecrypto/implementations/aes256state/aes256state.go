package aes256state

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
)

type AES256StateWrapper struct {
	key         []byte
	previousKey []byte
}

func parseKey(hexKey string, name string) ([]byte, error) {
	if hexKey == "" {
		// this case is explicitly allowed to support planned decryption
		return []byte{}, nil
	}

	validator := regexp.MustCompile("^[0-9a-f]{64}$")
	if !validator.MatchString(hexKey) {
		return []byte{}, fmt.Errorf("%s was not a hex string representing 32 bytes, must match [0-9a-f]{64}", name)
	}

	key, _ := hex.DecodeString(hexKey)

	return key, nil
}

func (a *AES256StateWrapper) parseKeysFromConfiguration(config []string) error {
	if len(config) < 2 || len(config) > 3 {
		return fmt.Errorf("configuration for AES256 needs to be AES256:key[:previousKey] where keys are 32 byte lower case hexadecimals and previous key is optional")
	}

	key, err := parseKey(config[1], "main key")
	if err != nil {
		return err
	}
	a.key = key

	if len(config) == 3 {
		key, err := parseKey(config[2], "previous key")
		if err != nil {
			return err
		}
		a.previousKey = key
	} else {
		a.previousKey = []byte{}
	}
	return nil
}

//  determine if data (which is a []byte containing a json structure) is encrypted
//         (that is, of the form: {"crypted":"<hex containing iv and payload>"})
func (a *AES256StateWrapper) isEncrypted(data []byte) bool {
	validator := regexp.MustCompile(`^{"crypted":"[0-9a-f]+"}$`)
	return validator.Match(data)
}

// decrypt the hex-encoded contents of data, which is expected to be of the form
//         {"crypted":"<hex containing iv and payload>"}
func (a *AES256StateWrapper) attemptDecryption(jsonCryptedData []byte, key []byte) ([]byte, error) {
	// extract the hex part only, cutting off {"crypted":" (12 characters) and "} (2 characters)
	src := jsonCryptedData[12 : len(jsonCryptedData)-2]

	ciphertext := make([]byte, hex.DecodedLen(len(src)))
	n, err := hex.Decode(ciphertext, src)
	if err != nil {
		return []byte{}, err
	}
	if n != hex.DecodedLen(len(src)) {
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

	// TODO now split off the hash from the payload and verify it to check that we have successfully decrypted

	// payload is now decrypted
	return payload, nil
}

// encrypt data (which is a []byte containing a json structure)
//      into a json structure {"crypted":"<hex-encoded random iv + hex-encoded CFB encrypted data>"}
// fail if encryption is not possible to prevent writing unencrypted state, but
// the case that key is empty is explicitly allowed to support planned decryption
func (a *AES256StateWrapper) Encrypt(plaintextPayload []byte) ([]byte, error) {
	// allow planned decryption
	if a.key == nil || len(a.key) == 0 {
		// TODO log warning that we are writing unencrypted (i.e. decrypting the state)
		return plaintextPayload, nil
	}

	block, err := aes.NewCipher(a.key)
	if err != nil {
		return []byte{}, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintextPayload)) // TODO + hash over plaintextPayload
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return []byte{}, err
	}

	// TODO add hash over plaintext to end of plaintext using append (used to detect decryption errors, but also acts as a mac)

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintextPayload)

	prefix := []byte(`{"crypted":"`)
	postfix := []byte(`"}`)
	encryptedHex := make([]byte, hex.EncodedLen(len(ciphertext)))
	_ = hex.Encode(encryptedHex, ciphertext)

	jsonCryptedData := append(append(prefix, encryptedHex...), postfix...)
	return jsonCryptedData, nil
}

func (a *AES256StateWrapper) Decrypt(data []byte) ([]byte, error) {
	if a.isEncrypted(data) {
		candidate, err := a.attemptDecryption(data, a.key)
		if err != nil {
			// allow key rotation (just change some null resource that is in the state file so it is rewritten)
			if a.previousKey != nil && len(a.previousKey) != 0 {
				// TODO log: error with main key as info, trying secondary key
				candidate, err = a.attemptDecryption(data, a.previousKey)
			}
			if err != nil {
				return []byte{}, err
			}
		}
		return candidate, nil
	} else {
		// TODO log info that we have transparently read unencrypted state
		return data, nil
	}
}
