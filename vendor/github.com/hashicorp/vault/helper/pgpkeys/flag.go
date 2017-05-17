package pgpkeys

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/keybase/go-crypto/openpgp"
)

// PGPPubKeyFiles implements the flag.Value interface and allows
// parsing and reading a list of pgp public key files
type PubKeyFilesFlag []string

func (p *PubKeyFilesFlag) String() string {
	return fmt.Sprint(*p)
}

func (p *PubKeyFilesFlag) Set(value string) error {
	if len(*p) > 0 {
		return errors.New("pgp-keys can only be specified once")
	}

	splitValues := strings.Split(value, ",")

	keybaseMap, err := FetchKeybasePubkeys(splitValues)
	if err != nil {
		return err
	}

	// Now go through the actual flag, and substitute in resolved keybase
	// entries where appropriate
	for _, keyfile := range splitValues {
		if strings.HasPrefix(keyfile, kbPrefix) {
			key := keybaseMap[keyfile]
			if key == "" {
				return fmt.Errorf("key for keybase user %s was not found in the map", strings.TrimPrefix(keyfile, kbPrefix))
			}
			*p = append(*p, key)
			continue
		}

		pgpStr, err := ReadPGPFile(keyfile)
		if err != nil {
			return err
		}

		*p = append(*p, pgpStr)
	}
	return nil
}

func ReadPGPFile(path string) (string, error) {
	if path[0] == '@' {
		path = path[1:]
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	buf := bytes.NewBuffer(nil)
	_, err = buf.ReadFrom(f)
	if err != nil {
		return "", err
	}

	// First parse as an armored keyring file, if that doesn't work, treat it as a straight binary/b64 string
	keyReader := bytes.NewReader(buf.Bytes())
	entityList, err := openpgp.ReadArmoredKeyRing(keyReader)
	if err == nil {
		if len(entityList) != 1 {
			return "", fmt.Errorf("more than one key found in file %s", path)
		}
		if entityList[0] == nil {
			return "", fmt.Errorf("primary key was nil for file %s", path)
		}

		serializedEntity := bytes.NewBuffer(nil)
		err = entityList[0].Serialize(serializedEntity)
		if err != nil {
			return "", fmt.Errorf("error serializing entity for file %s: %s", path, err)
		}

		return base64.StdEncoding.EncodeToString(serializedEntity.Bytes()), nil
	}

	_, err = base64.StdEncoding.DecodeString(buf.String())
	if err == nil {
		return buf.String(), nil
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil

}
