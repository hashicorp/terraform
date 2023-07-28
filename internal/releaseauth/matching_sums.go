package releaseauth

import "fmt"

// ErrChecksumMismatch is the error returned when a reported checksum does not match
// what is stored in a SHA256SUMS file
type ErrChecksumMismatch struct {
	Inner error
}

func (e ErrChecksumMismatch) Error() string {
	return fmt.Sprintf("failed to authenticate that release checksum matches checksum provided by the manifest: %v", e.Inner)
}

func (e ErrChecksumMismatch) Unwrap() error {
	return e.Inner
}

// MatchingChecksumsAuthentication is an archive Authenticator that checks if a reported checksum
// matches the checksum that was stored in a SHA256SUMS file
type MatchingChecksumsAuthentication struct {
	Authenticator

	expected SHA256Hash
	sums     SHA256Checksums
	baseName string
}

var _ Authenticator = MatchingChecksumsAuthentication{}

// NewMatchingChecksumsAuthentication creates the Authenticator given an expected hash,
// the parsed SHA256SUMS data, and a filename.
func NewMatchingChecksumsAuthentication(expected SHA256Hash, baseName string, sums SHA256Checksums) *MatchingChecksumsAuthentication {
	return &MatchingChecksumsAuthentication{
		expected: expected,
		sums:     sums,
		baseName: baseName,
	}
}

// Authenticate ensures that the given hash matches what is found in the SHA256SUMS file
// for the corresponding filename
func (a MatchingChecksumsAuthentication) Authenticate() error {
	err := a.sums.Validate(a.baseName, a.expected)
	if err != nil {
		return ErrChecksumMismatch{
			Inner: err,
		}
	}

	return nil
}
