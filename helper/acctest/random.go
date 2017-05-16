package acctest

import (
	"bufio"
	"bytes"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Helpers for generating random tidbits for use in identifiers to prevent
// collisions in acceptance tests.

// RandInt generates a random integer
func RandInt() int {
	reseed()
	return rand.New(rand.NewSource(time.Now().UnixNano())).Int()
}

// RandomWithPrefix is used to generate a unique name with a prefix, for
// randomizing names in acceptance tests
func RandomWithPrefix(name string) string {
	reseed()
	return fmt.Sprintf("%s-%d", name, rand.New(rand.NewSource(time.Now().UnixNano())).Int())
}

func RandIntRange(min int, max int) int {
	reseed()
	source := rand.New(rand.NewSource(time.Now().UnixNano()))
	rangeMax := max - min

	return int(source.Int31n(int32(rangeMax)))
}

// RandString generates a random alphanumeric string of the length specified
func RandString(strlen int) string {
	return RandStringFromCharSet(strlen, CharSetAlphaNum)
}

// RandStringFromCharSet generates a random string by selecting characters from
// the charset provided
func RandStringFromCharSet(strlen int, charSet string) string {
	reseed()
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = charSet[rand.Intn(len(charSet))]
	}
	return string(result)
}

// RandSSHKeyPair generates a public and private SSH key pair. The public key is
// returned in OpenSSH format, and the private key is PEM encoded.
func RandSSHKeyPair(comment string) (string, string, error) {
	privateKey, err := rsa.GenerateKey(crand.Reader, 1024)
	if err != nil {
		return "", "", err
	}

	var privateKeyBuffer bytes.Buffer
	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err := pem.Encode(bufio.NewWriter(&privateKeyBuffer), privateKeyPEM); err != nil {
		return "", "", err
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}
	keyMaterial := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey)))
	return fmt.Sprintf("%s %s", keyMaterial, comment), privateKeyBuffer.String(), nil
}

// Seeds random with current timestamp
func reseed() {
	rand.Seed(time.Now().UTC().UnixNano())
}

const (
	// CharSetAlphaNum is the alphanumeric character set for use with
	// RandStringFromCharSet
	CharSetAlphaNum = "abcdefghijklmnopqrstuvwxyz012346789"

	// CharSetAlpha is the alphabetical character set for use with
	// RandStringFromCharSet
	CharSetAlpha = "abcdefghijklmnopqrstuvwxyz"
)
