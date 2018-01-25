package terraform

import (
	"bufio"
	"bytes"
	aes "crypto/aes"
	cipher "crypto/cipher"
	"crypto/hmac"
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
//   !seal!<version #>!<base32 encoded salt>!<base32 encrypted hmac>!<base32 encoded, encrypted payload>

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
	// fmt.Fprintf(dst, "%v!", currentVersion)
	writeField(dst, []byte(currentVersion))
	return writeSealedStateV001(d, dst, password)
}

func writeSealedStateV001(d *State, dst io.Writer, password []byte) error {
	salt := make([]byte, keySize)
	_, err := srand.Read(salt)
	if err != nil {
		return fmt.Errorf("Could not generate salt for encryption: %v", err)
	}
	if err := writeField(dst, salt); err != nil {
		return fmt.Errorf("Could not write salt: %v", err)
	}
	key := generateKey(password, salt)
	base32Encoder := base32.NewEncoder(base32.StdEncoding, dst)
	defer base32Encoder.Close()
	dec, err := serializeState(d)
	if err != nil {
		return fmt.Errorf("Could not serialize state: %v", err)
	}
	enc, err := encrypt(dec, key)
	if err != nil {
		return fmt.Errorf("Could not encrypt state: %v", err)
	}
	mac := sign(enc, key)
	if err := writeField(dst, mac); err != nil {
		return fmt.Errorf("Could not write HMAC signature: %v", err)
	}
	if _, err := base32Encoder.Write(enc); err != nil {
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
	rawVersion, err := readField(bufSrc)
	if err != nil {
		return nil, fmt.Errorf("Could not read seal version: %v", err)
	}
	version := string(rawVersion)
	// Add additional version checks here
	switch version {
	case currentVersion:
		return readSealedStateV001(password, bufSrc)
	default:
		return nil, fmt.Errorf("Seal version not recognized: %v", version)
	}
}

func readSealedStateV001(password []byte, bufSrc *bufio.Reader) (*State, error) {
	salt, err := readField(bufSrc)
	if err != nil {
		return nil, fmt.Errorf("Could not decode salt: %v", err)
	}
	expectedMac, err := readField(bufSrc)
	key := generateKey(password, salt)
	base32Decoder := base32.NewDecoder(base32.StdEncoding, bufSrc)
	enc, err := ioutil.ReadAll(base32Decoder)
	if err != nil {
		return nil, fmt.Errorf("Could not decode sealed state: %v", err)
	}
	actualMac := sign(enc, key)
	if !hmac.Equal(expectedMac, actualMac) {
		return nil, fmt.Errorf("HMAC signature not matched")
	}
	dec, err := decrypt(enc, key)
	if err != nil {
		return nil, fmt.Errorf("Could not decrypt state:%v", err)
	}
	buffer := bytes.NewBuffer(dec)
	return ReadState(buffer)
}

func writeField(dst io.Writer, rawData []byte) error {
	base32Data := base32.StdEncoding.EncodeToString(rawData)
	if _, err := io.WriteString(dst, base32Data); err != nil {
		return fmt.Errorf("Could not write base32 encoded field: %v", err)
	}
	if _, err := io.WriteString(dst, "!"); err != nil {
		return fmt.Errorf("Could not write field delimiter: %v", err)
	}
	return nil
}

func readField(bufSrc *bufio.Reader) ([]byte, error) {
	rawData, err := bufSrc.ReadBytes('!')
	if err != nil {
		return nil, fmt.Errorf("No delimiter found, could not read data: %v", err)
	}
	// trim the '!' off the end, and convert to string
	base32Data := string(rawData[0 : len(rawData)-1])
	return base32.StdEncoding.DecodeString(base32Data)
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

func sign(msg []byte, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg)
	return mac.Sum(nil)
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
