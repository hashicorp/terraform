package getproviders

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/openpgp"
	openpgpArmor "golang.org/x/crypto/openpgp/armor"
	openpgpErrors "golang.org/x/crypto/openpgp/errors"
)

type packageAuthenticationResult int

const (
	verifiedChecksum packageAuthenticationResult = iota
	officialProvider
	partnerProvider
	communityProvider
)

// PackageAuthenticationResult is returned from a PackageAuthentication
// implementation. It is a mostly-opaque type intended for use in UI, which
// implements Stringer and includes an optional Warning field.
//
// A failed PackageAuthentication attempt will return an "unauthenticated"
// result, which is represented by nil.
type PackageAuthenticationResult struct {
	result  packageAuthenticationResult
	Warning string
}

func (t *PackageAuthenticationResult) String() string {
	if t == nil {
		return "unauthenticated"
	}
	return []string{
		"verified checksum",
		"official provider",
		"partner provider",
		"community provider",
	}[t.result]
}

// SigningKey represents a key used to sign packages from a registry, along
// with an optional trust signature from the registry operator. These are
// both in ASCII armored OpenPGP format.
//
// The JSON struct tags represent the field names used by the Registry API.
type SigningKey struct {
	ASCIIArmor     string `json:"ascii_armor"`
	TrustSignature string `json:"trust_signature"`
}

// PackageAuthentication is an interface implemented by the optional package
// authentication implementations a source may include on its PackageMeta
// objects.
//
// A PackageAuthentication implementation is responsible for authenticating
// that a package is what its distributor intended to distribute and that it
// has not been tampered with.
type PackageAuthentication interface {
	// AuthenticatePackage takes the local location of a package (which may or
	// may not be the same as the original source location), and returns a
	// PackageAuthenticationResult, or an error if the authentication checks
	// fail.
	//
	// The local location is guaranteed not to be a PackageHTTPURL: a remote
	// package will always be staged locally for inspection first.
	AuthenticatePackage(localLocation PackageLocation) (*PackageAuthenticationResult, error)
}

type packageAuthenticationAll []PackageAuthentication

// PackageAuthenticationAll combines several authentications together into a
// single check value, which passes only if all of the given ones pass.
//
// The checks are processed in the order given, so a failure of an earlier
// check will prevent execution of a later one.
//
// The returned result is from the last authentication, so callers should
// take care to order the authentications such that the strongest is last.
func PackageAuthenticationAll(checks ...PackageAuthentication) PackageAuthentication {
	return packageAuthenticationAll(checks)
}

func (checks packageAuthenticationAll) AuthenticatePackage(localLocation PackageLocation) (*PackageAuthenticationResult, error) {
	var authResult *PackageAuthenticationResult
	for _, check := range checks {
		var err error
		authResult, err = check.AuthenticatePackage(localLocation)
		if err != nil {
			return authResult, err
		}
	}
	return authResult, nil
}

type archiveHashAuthentication struct {
	WantSHA256Sum [sha256.Size]byte
}

// NewArchiveChecksumAuthentication returns a PackageAuthentication
// implementation that checks that the original distribution archive matches
// the given hash.
//
// This authentication is suitable only for PackageHTTPURL and
// PackageLocalArchive source locations, because the unpacked layout
// (represented by PackageLocalDir) does not retain access to the original
// source archive. Therefore this authenticator will return an error if its
// given localLocation is not PackageLocalArchive.
func NewArchiveChecksumAuthentication(wantSHA256Sum [sha256.Size]byte) PackageAuthentication {
	return archiveHashAuthentication{wantSHA256Sum}
}

func (a archiveHashAuthentication) AuthenticatePackage(localLocation PackageLocation) (*PackageAuthenticationResult, error) {
	archiveLocation, ok := localLocation.(PackageLocalArchive)
	if !ok {
		// A source should not use this authentication type for non-archive
		// locations.
		return nil, fmt.Errorf("cannot check archive hash for non-archive location %s", localLocation)
	}

	f, err := os.Open(string(archiveLocation))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return nil, err
	}

	gotHash := h.Sum(nil)
	if !bytes.Equal(gotHash, a.WantSHA256Sum[:]) {
		return nil, fmt.Errorf("archive has incorrect SHA-256 checksum %x (expected %x)", gotHash, a.WantSHA256Sum[:])
	}
	return &PackageAuthenticationResult{result: verifiedChecksum}, nil
}

type matchingChecksumAuthentication struct {
	Document      []byte
	Filename      string
	WantSHA256Sum [sha256.Size]byte
}

// NewMatchingChecksumAuthentication returns a PackageAuthentication
// implementation that scans a registry-provided SHA256SUMS document for a
// specified filename, and compares the SHA256 hash against the expected hash.
// This is necessary to ensure that the signed SHA256SUMS document matches the
// declared SHA256 hash for the package, and therefore that a valid signature
// of this document authenticates the package.
//
// This authentication always returns a nil result, since it alone cannot offer
// any assertions about package integrity. It should be combined with other
// authentications to be useful.
func NewMatchingChecksumAuthentication(document []byte, filename string, wantSHA256Sum [sha256.Size]byte) PackageAuthentication {
	return matchingChecksumAuthentication{
		Document:      document,
		Filename:      filename,
		WantSHA256Sum: wantSHA256Sum,
	}
}

func (m matchingChecksumAuthentication) AuthenticatePackage(location PackageLocation) (*PackageAuthenticationResult, error) {
	// Find the checksum in the list with matching filename. The document is
	// in the form "0123456789abcdef filename.zip".
	filename := []byte(m.Filename)
	var checksum []byte
	for _, line := range bytes.Split(m.Document, []byte("\n")) {
		parts := bytes.Fields(line)
		if len(parts) > 1 && bytes.Equal(parts[1], filename) {
			checksum = parts[0]
			break
		}
	}
	if checksum == nil {
		return nil, fmt.Errorf("checksum list has no SHA-256 hash for %q", m.Filename)
	}

	// Decode the ASCII checksum into a byte array for comparison.
	var gotSHA256Sum [sha256.Size]byte
	if _, err := hex.Decode(gotSHA256Sum[:], checksum); err != nil {
		return nil, fmt.Errorf("checksum list has invalid SHA256 hash %q: %s", string(checksum), err)
	}

	// If the checksums don't match, authentication fails.
	if !bytes.Equal(gotSHA256Sum[:], m.WantSHA256Sum[:]) {
		return nil, fmt.Errorf("checksum list has unexpected SHA-256 hash %x (expected %x)", gotSHA256Sum, m.WantSHA256Sum[:])
	}

	// Success! But this doesn't result in any real authentication, only a
	// lack of authentication errors, so we return a nil result.
	return nil, nil
}

type signatureAuthentication struct {
	Document  []byte
	Signature []byte
	Keys      []SigningKey
}

// NewSignatureAuthentication returns a PackageAuthentication implementation
// that verifies the cryptographic signature for a package against any of the
// provided keys.
//
// The signing key for a package will be auto detected by attempting each key
// in turn until one is successful. If such a key is found, there are three
// possible successful authentication results:
//
// 1. If the signing key is the HashiCorp official key, it is an official
//    provider;
// 2. Otherwise, if the signing key has a trust signature from the HashiCorp
//    Partners key, it is a partner provider;
// 3. If neither of the above is true, it is a community provider.
//
// Any failure in the process of validating the signature will result in an
// unauthenticated result.
func NewSignatureAuthentication(document, signature []byte, keys []SigningKey) PackageAuthentication {
	return signatureAuthentication{
		Document:  document,
		Signature: signature,
		Keys:      keys,
	}
}

func (s signatureAuthentication) AuthenticatePackage(location PackageLocation) (*PackageAuthenticationResult, error) {
	// Find the key that signed the checksum file. This can fail if there is no
	// valid signature for any of the provided keys.
	signingKey, err := s.findSigningKey()
	if err != nil {
		return nil, err
	}

	// Verify the signature using the HashiCorp public key. If this succeeds,
	// this is an official provider.
	hashicorpKeyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(HashicorpPublicKey))
	if err != nil {
		return nil, fmt.Errorf("error creating HashiCorp keyring: %s", err)
	}
	_, err = openpgp.CheckDetachedSignature(hashicorpKeyring, bytes.NewReader(s.Document), bytes.NewReader(s.Signature))
	if err == nil {
		return &PackageAuthenticationResult{result: officialProvider}, nil
	}

	// If the signing key has a trust signature, attempt to verify it with the
	// HashiCorp partners public key.
	if signingKey.TrustSignature != "" {
		hashicorpPartnersKeyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(HashicorpPartnersKey))
		if err != nil {
			return nil, fmt.Errorf("error creating HashiCorp Partners keyring: %s", err)
		}

		authorKey, err := openpgpArmor.Decode(strings.NewReader(signingKey.ASCIIArmor))
		if err != nil {
			return nil, fmt.Errorf("error decoding signing key: %s", err)
		}

		trustSignature, err := openpgpArmor.Decode(strings.NewReader(signingKey.TrustSignature))
		if err != nil {
			return nil, fmt.Errorf("error decoding trust signature: %s", err)
		}

		_, err = openpgp.CheckDetachedSignature(hashicorpPartnersKeyring, authorKey.Body, trustSignature.Body)
		if err != nil {
			return nil, fmt.Errorf("error verifying trust signature: %s", err)
		}

		return &PackageAuthenticationResult{result: partnerProvider}, nil
	}

	// We have a valid signature, but it's not from the HashiCorp key, and it
	// also isn't a trusted partner. This is a community provider.
	// FIXME: we may want to add a more detailed warning here explaining the
	// difference between partner and community providers.
	return &PackageAuthenticationResult{result: communityProvider}, nil
}

// findSigningKey attempts to verify the signature using each of the keys
// returned by the registry. If a valid signature is found, it returns the
// signing key.
//
// Note: currently the registry only returns one key, but this may change in
// the future.
func (s signatureAuthentication) findSigningKey() (*SigningKey, error) {
	for _, key := range s.Keys {
		keyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(key.ASCIIArmor))
		if err != nil {
			return nil, fmt.Errorf("error decoding signing key: %s", err)
		}

		entity, err := openpgp.CheckDetachedSignature(keyring, bytes.NewReader(s.Document), bytes.NewReader(s.Signature))

		// If the signature issuer does not match the the key, keep trying the
		// rest of the provided keys.
		if err == openpgpErrors.ErrUnknownIssuer {
			continue
		}

		// Any other signature error is terminal.
		if err != nil {
			return nil, fmt.Errorf("error checking signature: %s", err)
		}

		log.Printf("[DEBUG] Provider signed by %s", entityString(entity))
		return &key, nil
	}

	// If none of the provided keys issued the signature, this package is
	// unsigned. This is currently a terminal authentication error.
	return nil, fmt.Errorf("authentication signature from unknown issuer")
}

// entityString extracts the key ID and identity name(s) from an openpgp.Entity
// for logging.
func entityString(entity *openpgp.Entity) string {
	if entity == nil {
		return ""
	}

	keyID := "n/a"
	if entity.PrimaryKey != nil {
		keyID = entity.PrimaryKey.KeyIdString()
	}

	var names []string
	for _, identity := range entity.Identities {
		names = append(names, identity.Name)
	}

	return fmt.Sprintf("%s %s", keyID, strings.Join(names, ", "))
}
