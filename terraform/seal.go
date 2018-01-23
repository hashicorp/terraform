package terraform

import (
	"bufio"
	aes "crypto/aes"
	cipher "crypto/cipher"
	srand "crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	home "github.com/mitchellh/go-homedir"

	pbkdf2 "golang.org/x/crypto/pbkdf2"
)

const sealPrefix = "!seal!"
const keyGenerationIterations = 4096
const keySize = 32

// WriteSealedState writes state in encrypted form onto the destination
// if passwordFilePath is not nil
func WriteSealedState(d *State, dst io.Writer, rawPasswordFilePath string, encrypt bool) error {
	if !encrypt {
		return WriteState(d, dst)
	}
	if rawPasswordFilePath == "" {
		return WriteState(d, dst)
	}
	passwordFilePath, err := home.Expand(rawPasswordFilePath)
	if err != nil {
		return fmt.Errorf("Could not expand file path: %v", err)
	}
	password, err := ioutil.ReadFile(passwordFilePath)
	if err != nil {
		return fmt.Errorf("Password file could not be read: %v", err)
	}
	salt := make([]byte, keySize)
	_, err = srand.Read(salt)
	if err != nil {
		return fmt.Errorf("Could not generate salt for encryption: %v", err)
	}
	key := pbkdf2.Key(password, salt, keyGenerationIterations, keySize, sha1.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("Could not create cipher for encryption: %v", err)
	}
	// Format of sealed state is:
	//   !seal!<base32 encoded salt>!<base32 encoded, encrypted payload
	io.WriteString(dst, sealPrefix)
	encodedSalt := base32.StdEncoding.EncodeToString(salt)
	io.WriteString(dst, encodedSalt)
	io.WriteString(dst, "!")
	iv := make([]byte, aes.BlockSize)
	stream := cipher.NewOFB(block, iv)
	base32Encoder := base32.NewEncoder(base32.StdEncoding, dst)
	encryptedDst := &cipher.StreamWriter{S: stream, W: base32Encoder}
	defer encryptedDst.Close()
	return WriteState(d, encryptedDst)
}

// ReadSealedState reads state in encrypted from from the source
// if passwordFilePath is not nil
func ReadSealedState(src io.Reader, rawPasswordFilePath string) (*State, error) {
	if rawPasswordFilePath == "" {
		log.Printf("[INFO] No password_file_path; not decrypting state")
		return ReadState(src)
	}
	passwordFilePath, err := home.Expand(rawPasswordFilePath)
	if err != nil {
		return nil, fmt.Errorf("Could not expand file path: %v", err)
	}
	password, err := ioutil.ReadFile(passwordFilePath)
	if err != nil {
		log.Printf("[INFO] Password file could not be read: %v", err)
		return nil, fmt.Errorf("Password file could not be read: %v", err)
	}
	log.Printf("[INFO] Using password file %v", passwordFilePath)
	bufSrc := bufio.NewReader(src)
	header, err := bufSrc.Peek(len(sealPrefix))
	if err != nil || string(header) != sealPrefix {
		// assume not a sealed file, default to just reading state
		log.Printf("[INFO] State not encrypted; no header found")
		return ReadState(bufSrc)
	}
	// we assume we read the prefix, so from here on out it must be a well-formed
	// sealed state
	_, err = bufSrc.Discard(len(sealPrefix))
	if err != nil {
		return nil, fmt.Errorf("Could not discard sealed header: %v", err)
	}
	// salt is separated from main encrypted state by '!'
	rawSalt, err := bufSrc.ReadBytes('!')
	if err != nil {
		return nil, fmt.Errorf("Could not read salt from sealed state: %v", err)
	}
	// trim the '!' off the end, and convert to string
	base32Salt := string(rawSalt[0 : len(rawSalt)-1])
	salt, err := base32.StdEncoding.DecodeString(base32Salt)
	if err != nil {
		return nil, fmt.Errorf("Could not decode salt: %v", err)
	}
	key := pbkdf2.Key(password, salt, keyGenerationIterations, keySize, sha1.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("Could not create cipher for encryption: %v", err)
	}
	iv := make([]byte, aes.BlockSize)
	stream := cipher.NewOFB(block, iv)
	base32Decoder := base32.NewDecoder(base32.StdEncoding, bufSrc)
	decryptedSrc := &cipher.StreamReader{S: stream, R: base32Decoder}
	return ReadState(decryptedSrc)
}
