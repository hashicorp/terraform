package authentication

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/hashicorp/errwrap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type SSHAgentSigner struct {
	formattedKeyFingerprint string
	keyFingerprint          string
	accountName             string
	keyIdentifier           string

	agent agent.Agent
	key   ssh.PublicKey
}

func NewSSHAgentSigner(keyFingerprint, accountName string) (*SSHAgentSigner, error) {
	sshAgentAddress := os.Getenv("SSH_AUTH_SOCK")
	if sshAgentAddress == "" {
		return nil, errors.New("SSH_AUTH_SOCK is not set")
	}

	conn, err := net.Dial("unix", sshAgentAddress)
	if err != nil {
		return nil, errwrap.Wrapf("Error dialing SSH agent: {{err}}", err)
	}

	ag := agent.NewClient(conn)

	keys, err := ag.List()
	if err != nil {
		return nil, errwrap.Wrapf("Error listing keys in SSH Agent: %s", err)
	}

	keyFingerprintMD5 := strings.Replace(keyFingerprint, ":", "", -1)

	var matchingKey ssh.PublicKey
	for _, key := range keys {
		h := md5.New()
		h.Write(key.Marshal())
		fp := fmt.Sprintf("%x", h.Sum(nil))

		if fp == keyFingerprintMD5 {
			matchingKey = key
		}
	}

	if matchingKey == nil {
		return nil, fmt.Errorf("No key in the SSH Agent matches fingerprint: %s", keyFingerprint)
	}

	formattedKeyFingerprint := formatPublicKeyFingerprint(matchingKey, true)

	return &SSHAgentSigner{
		formattedKeyFingerprint: formattedKeyFingerprint,
		keyFingerprint:          keyFingerprint,
		accountName:             accountName,
		agent:                   ag,
		key:                     matchingKey,
		keyIdentifier:           fmt.Sprintf("/%s/keys/%s", accountName, formattedKeyFingerprint),
	}, nil
}

func (s *SSHAgentSigner) Sign(dateHeader string) (string, error) {
	const headerName = "date"

	signature, err := s.agent.Sign(s.key, []byte(fmt.Sprintf("%s: %s", headerName, dateHeader)))
	if err != nil {
		return "", errwrap.Wrapf("Error signing date header: {{err}}", err)
	}

	keyFormat, err := keyFormatToKeyType(signature.Format)
	if err != nil {
		return "", errwrap.Wrapf("Error reading signature: {{err}}", err)
	}

	var authSignature httpAuthSignature
	switch keyFormat {
	case "rsa":
		authSignature, err = newRSASignature(signature.Blob)
		if err != nil {
			return "", errwrap.Wrapf("Error reading signature: {{err}}", err)
		}
	case "ecdsa":
		authSignature, err = newECDSASignature(signature.Blob)
		if err != nil {
			return "", errwrap.Wrapf("Error reading signature: {{err}}", err)
		}
	default:
		return "", fmt.Errorf("Unsupported algorithm from SSH agent: %s", signature.Format)
	}

	return fmt.Sprintf(authorizationHeaderFormat, s.keyIdentifier,
		authSignature.SignatureType(), headerName, authSignature.String()), nil
}
