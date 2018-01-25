package terraform

import (
	"bufio"
	"bytes"
	aes "crypto/aes"
	cipher "crypto/cipher"
	srand "crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	home "github.com/mitchellh/go-homedir"

	pbkdf2 "golang.org/x/crypto/pbkdf2"
)

const sealPrefix = "!seal!"
const keyGenerationIterations = 3 * 4096
const keySize = 32
const currentVersion = "001"

// Format of sealed state is:
//   !seal!<version #>!<base32 encoded salt>!<base32 encoded, encrypted payload

// WriteSealedState writes state in encrypted form onto the destination
// if rawPasswordFilePath is not nil
func WriteSealedState(d *State, dst io.Writer, rawPasswordFilePath string, encrypt bool) error {
	if !encrypt {
		return WriteState(d, dst)
	}
	if rawPasswordFilePath == "" {
		return WriteState(d, dst)
	}
	password, err := readPassword(rawPasswordFilePath)
	if err != nil {
		return err
	}
	io.WriteString(dst, sealPrefix)
	fmt.Fprintf(dst, "%v!", currentVersion)
	return writeSealedStateV001(d, dst, password)
}

func writeSealedStateV001(d *State, dst io.Writer, password []byte) error {
	salt := make([]byte, keySize)
	_, err := srand.Read(salt)
	if err != nil {
		return fmt.Errorf("Could not generate salt for encryption: %v", err)
	}
	encodedSalt := base32.StdEncoding.EncodeToString(salt)
	io.WriteString(dst, encodedSalt)
	io.WriteString(dst, "!")
	key := generateKey(password, salt)
	base32Encoder := base32.NewEncoder(base32.StdEncoding, dst)
	defer base32Encoder.Close()
	decBytes, err := serializeState(d)
	if err != nil {
		return fmt.Errorf("Could not serialize state: %v", err)
	}
	encBytes, err := encrypt(decBytes, key)
	if err != nil {
		return fmt.Errorf("Could not encrypt state: %v", err)
	}
	if _, err := base32Encoder.Write(encBytes); err != nil {
		return fmt.Errorf("Could not encode encrypted state: %v", err)
	}
	return err
}

// ReadSealedState reads state in encrypted from from the source
// if rawPasswordFilePath is not nil
func ReadSealedState(src io.Reader, rawPasswordFilePath string) (*State, error) {
	if rawPasswordFilePath == "" {
		log.Printf("[INFO] No password_file_path; not decrypting state")
		return ReadState(src)
	}
	password, err := readPassword(rawPasswordFilePath)
	if err != nil {
		return nil, err
	}
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
	rawVersion, err := bufSrc.ReadBytes('!')
	if err != nil {
		return nil, fmt.Errorf("Could not read seal version: %v", err)
	}
	version := string(rawVersion[0 : len(rawVersion)-1])
	// Add additional version checks here
	switch version {
	case currentVersion:
		return readSealedStateV001(password, bufSrc)
	default:
		return nil, fmt.Errorf("Seal version not recognized: %v", version)
	}
}

func readSealedStateV001(password []byte, bufSrc *bufio.Reader) (*State, error) {
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
	key := generateKey(password, salt)
	base32Decoder := base32.NewDecoder(base32.StdEncoding, bufSrc)
	encBytes, err := ioutil.ReadAll(base32Decoder)
	if err != nil {
		return nil, fmt.Errorf("Could not decode sealed state: %v", err)
	}
	decBytes, err := decrypt(encBytes, key)
	if err != nil {
		return nil, fmt.Errorf("Could not decrypt state:%v", err)
	}
	buffer := bytes.NewBuffer(decBytes)
	return ReadState(buffer)
}

func serializeState(d *State) ([]byte, error) {
	var buffer bytes.Buffer
	err := WriteState(d, &buffer)
	return buffer.Bytes(), err
}

func deserializeState(data []byte) (*State, error) {
	buffer := bytes.NewBuffer(data)
	state, err := ReadState(buffer)
	return state, err
}

func encrypt(dec []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("Could not create cipher for encryption: %v", err)
	}
	iv := make([]byte, aes.BlockSize)
	stream := cipher.NewOFB(block, iv)
	var buffer bytes.Buffer
	encryptedDst := &cipher.StreamWriter{S: stream, W: &buffer}
	if _, err := encryptedDst.Write(dec); err != nil {
		return nil, fmt.Errorf("Could not encrypt: %v", err)
	}
	return buffer.Bytes(), nil
}

func decrypt(enc []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("Could not create cipher for encryption: %v", err)
	}
	iv := make([]byte, aes.BlockSize)
	stream := cipher.NewOFB(block, iv)
	buffer := bytes.NewBuffer(enc)
	decryptedSrc := &cipher.StreamReader{S: stream, R: buffer}
	return ioutil.ReadAll(decryptedSrc)
}

func readPassword(rawPasswordFilePath string) ([]byte, error) {
	passwordFilePath, err := home.Expand(rawPasswordFilePath)
	log.Printf("[INFO] Using password file %v", passwordFilePath)
	if err != nil {
		return nil, fmt.Errorf("Could not expand file path: %v", err)
	}
	password, err := ioutil.ReadFile(passwordFilePath)
	if err != nil {
		return nil, fmt.Errorf("Password file could not be read: %v", err)
	}
	return password, nil
}

func generateKey(password []byte, salt []byte) []byte {
	return pbkdf2.Key(password, salt, keyGenerationIterations, keySize, sha256.New)
}
