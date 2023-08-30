// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package releaseauth

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
)

// ChecksumAuthentication is an archive Authenticator that ensures a given file
// matches a SHA-256 checksum. It is important to verify the authenticity of the
// given checksum prior to using this Authenticator.
type ChecksumAuthentication struct {
	Authenticator

	expected        SHA256Hash
	archiveLocation string
}

// ErrChecksumDoesNotMatch is the error returned when the archive checksum does
// not match the given checksum.
var ErrChecksumDoesNotMatch = errors.New("downloaded archive does not match the release checksum")

// NewChecksumAuthentication creates an instance of ChecksumAuthentication with the given
// checksum and file location.
func NewChecksumAuthentication(expected SHA256Hash, archiveLocation string) *ChecksumAuthentication {
	return &ChecksumAuthentication{
		expected:        expected,
		archiveLocation: archiveLocation,
	}
}

func (a ChecksumAuthentication) Authenticate() error {
	f, err := os.Open(a.archiveLocation)
	if err != nil {
		return fmt.Errorf("failed to open downloaded archive: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return fmt.Errorf("failed to hash downloaded archive: %w", err)
	}

	gotHash := h.Sum(nil)
	log.Printf("[TRACE] checksummed %q; got hash %x, expected %x", f.Name(), gotHash, a.expected)
	if !bytes.Equal(gotHash, a.expected[:]) {
		return ErrChecksumDoesNotMatch
	}

	return nil
}
