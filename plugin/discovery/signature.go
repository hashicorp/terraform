package discovery

import (
	"bytes"
	"strings"

	"golang.org/x/crypto/openpgp"
)

// Verify the data using the provided openpgp detached signature and the
// embedded hashicorp public key.
func verifySig(data, sig []byte, armor string) (*openpgp.Entity, error) {
	el, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armor))
	if err != nil {
		return nil, err
	}

	return openpgp.CheckDetachedSignature(el, bytes.NewReader(data), bytes.NewReader(sig))
}
