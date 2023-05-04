// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package getproviders

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	// TODO: replace crypto/openpgp since it is deprecated
	// https://github.com/golang/go/issues/44226
	//lint:file-ignore SA1019 openpgp is deprecated but there are no good alternatives yet
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
// implements Stringer.
//
// A failed PackageAuthentication attempt will return an "unauthenticated"
// result, which is represented by nil.
type PackageAuthenticationResult struct {
	result packageAuthenticationResult
	KeyID  string
}

func (t *PackageAuthenticationResult) String() string {
	if t == nil {
		return "unauthenticated"
	}
	return []string{
		"verified checksum",
		"signed by HashiCorp",
		"signed by a HashiCorp partner",
		"self-signed",
	}[t.result]
}

// SignedByHashiCorp returns whether the package was authenticated as signed
// by HashiCorp.
func (t *PackageAuthenticationResult) SignedByHashiCorp() bool {
	if t == nil {
		return false
	}
	if t.result == officialProvider {
		return true
	}

	return false
}

// SignedByAnyParty returns whether the package was authenticated as signed
// by either HashiCorp or by a third-party.
func (t *PackageAuthenticationResult) SignedByAnyParty() bool {
	if t == nil {
		return false
	}
	if t.result == officialProvider || t.result == partnerProvider || t.result == communityProvider {
		return true
	}

	return false
}

// ThirdPartySigned returns whether the package was authenticated as signed by a party
// other than HashiCorp.
func (t *PackageAuthenticationResult) ThirdPartySigned() bool {
	if t == nil {
		return false
	}
	if t.result == partnerProvider || t.result == communityProvider {
		return true
	}

	return false
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

// PackageAuthenticationHashes is an optional interface implemented by
// PackageAuthentication implementations that are able to return a set of
// hashes they would consider valid if a given PackageLocation referred to
// a package that matched that hash string.
//
// This can be used to record a set of acceptable hashes for a particular
// package in a lock file so that future install operations can determine
// whether the package has changed since its initial installation.
type PackageAuthenticationHashes interface {
	PackageAuthentication

	// AcceptableHashes returns a set of hashes that this authenticator
	// considers to be valid for the current package or, where possible,
	// equivalent packages on other platforms. The order of the items in
	// the result is not significant, and it may contain duplicates
	// that are also not significant.
	//
	// This method's result should only be used to create a "lock" for a
	// particular provider if an earlier call to AuthenticatePackage for
	// the corresponding package succeeded. A caller might choose to apply
	// differing levels of trust for the acceptable hashes depending on
	// the authentication result: a "verified checksum" result only checked
	// that the downloaded package matched what the source claimed, which
	// could be considered to be less trustworthy than a check that includes
	// verifying a signature from the origin registry, depending on what the
	// hashes are going to be used for.
	//
	// Implementations of PackageAuthenticationHashes may return multiple
	// hashes with different schemes, which means that all of them are equally
	// acceptable. Implementors may also return hashes that use schemes the
	// current version of the authenticator would not allow but that could be
	// accepted by other versions of Terraform, e.g. if a particular hash
	// scheme has been deprecated.
	//
	// Authenticators that don't use hashes as their authentication procedure
	// will either not implement this interface or will have an implementation
	// that returns an empty result.
	AcceptableHashes() []Hash
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
//
// The returned object also implements the AcceptableHashes method from
// interface PackageAuthenticationHashes, returning the hashes from the
// last of the given checks that indicates at least one acceptable hash,
// or no hashes at all if none of the constituents indicate any. The result
// may therefore be incomplete if there is more than one check that can provide
// hashes and they disagree about which hashes are acceptable.
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

func (checks packageAuthenticationAll) AcceptableHashes() []Hash {
	// The elements of checks are expected to be ordered so that the strongest
	// one is later in the list, so we'll visit them in reverse order and
	// take the first one that implements the interface and returns a non-empty
	// result.
	for i := len(checks) - 1; i >= 0; i-- {
		check, ok := checks[i].(PackageAuthenticationHashes)
		if !ok {
			continue
		}
		allHashes := check.AcceptableHashes()
		if len(allHashes) > 0 {
			return allHashes
		}
	}
	return nil
}

type packageHashAuthentication struct {
	RequiredHashes []Hash
	AllHashes      []Hash
	Platform       Platform
}

// NewPackageHashAuthentication returns a PackageAuthentication implementation
// that checks whether the contents of the package match whatever subset of the
// given hashes are considered acceptable by the current version of Terraform.
//
// This uses the hash algorithms implemented by functions PackageHash and
// MatchesHash. The PreferredHashes function will select which of the given
// hashes are considered by Terraform to be the strongest verification, and
// authentication succeeds as long as one of those matches.
func NewPackageHashAuthentication(platform Platform, validHashes []Hash) PackageAuthentication {
	requiredHashes := PreferredHashes(validHashes)
	return packageHashAuthentication{
		RequiredHashes: requiredHashes,
		AllHashes:      validHashes,
		Platform:       platform,
	}
}

func (a packageHashAuthentication) AuthenticatePackage(localLocation PackageLocation) (*PackageAuthenticationResult, error) {
	if len(a.RequiredHashes) == 0 {
		// Indicates that none of the hashes given to
		// NewPackageHashAuthentication were considered to be usable by this
		// version of Terraform.
		return nil, fmt.Errorf("this version of Terraform does not support any of the checksum formats given for this provider")
	}

	matches, err := PackageMatchesAnyHash(localLocation, a.RequiredHashes)
	if err != nil {
		return nil, fmt.Errorf("failed to verify provider package checksums: %s", err)
	}

	if matches {
		return &PackageAuthenticationResult{result: verifiedChecksum}, nil
	}
	if len(a.RequiredHashes) == 1 {
		return nil, fmt.Errorf("provider package doesn't match the expected checksum %q", a.RequiredHashes[0].String())
	}
	// It's non-ideal that this doesn't actually list the expected checksums,
	// but in the many-checksum case the message would get pretty unweildy.
	// In practice today we typically use this authenticator only with a
	// single hash returned from a network mirror, so the better message
	// above will prevail in that case. Maybe we'll improve on this somehow
	// if the future introduction of a new hash scheme causes there to more
	// commonly be multiple hashes.
	return nil, fmt.Errorf("provider package doesn't match the any of the expected checksums")
}

func (a packageHashAuthentication) AcceptableHashes() []Hash {
	// In this case we include even hashes the current version of Terraform
	// doesn't prefer, because this result is used for building a lock file
	// and so it's helpful to include older hash formats that other Terraform
	// versions might need in order to do authentication successfully.
	return a.AllHashes
}

type archiveHashAuthentication struct {
	Platform      Platform
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
//
// NewPackageHashAuthentication is preferable to use when possible because
// it uses the newer hashing scheme (implemented by function PackageHash) that
// can work with both packed and unpacked provider packages.
func NewArchiveChecksumAuthentication(platform Platform, wantSHA256Sum [sha256.Size]byte) PackageAuthentication {
	return archiveHashAuthentication{platform, wantSHA256Sum}
}

func (a archiveHashAuthentication) AuthenticatePackage(localLocation PackageLocation) (*PackageAuthenticationResult, error) {
	archiveLocation, ok := localLocation.(PackageLocalArchive)
	if !ok {
		// A source should not use this authentication type for non-archive
		// locations.
		return nil, fmt.Errorf("cannot check archive hash for non-archive location %s", localLocation)
	}

	gotHash, err := PackageHashLegacyZipSHA(archiveLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to compute checksum for %s: %s", archiveLocation, err)
	}
	wantHash := HashLegacyZipSHAFromSHA(a.WantSHA256Sum)
	if gotHash != wantHash {
		return nil, fmt.Errorf("archive has incorrect checksum %s (expected %s)", gotHash, wantHash)
	}
	return &PackageAuthenticationResult{result: verifiedChecksum}, nil
}

func (a archiveHashAuthentication) AcceptableHashes() []Hash {
	return []Hash{HashLegacyZipSHAFromSHA(a.WantSHA256Sum)}
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
//  1. If the signing key is the HashiCorp official key, it is an official
//     provider;
//  2. Otherwise, if the signing key has a trust signature from the HashiCorp
//     Partners key, it is a partner provider;
//  3. If neither of the above is true, it is a community provider.
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
	signingKey, keyID, err := s.findSigningKey()
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
		return &PackageAuthenticationResult{result: officialProvider, KeyID: keyID}, nil
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

		return &PackageAuthenticationResult{result: partnerProvider, KeyID: keyID}, nil
	}

	// We have a valid signature, but it's not from the HashiCorp key, and it
	// also isn't a trusted partner. This is a community provider.
	return &PackageAuthenticationResult{result: communityProvider, KeyID: keyID}, nil
}

func (s signatureAuthentication) AcceptableHashes() []Hash {
	// This is a bit of an abstraction leak because signatureAuthentication
	// otherwise just treats the document as an opaque blob that's been
	// signed, but here we're making assumptions about its format because
	// we only want to trust that _all_ of the checksums are valid (rather
	// than just the current platform's one) if we've also verified that the
	// bag of checksums is signed.
	//
	// In recognition of that layering quirk this implementation is intended to
	// be somewhat resilient to potentially using this authenticator with
	// non-checksums files in future (in which case it'll return nothing at all)
	// but it might be better in the long run to instead combine
	// signatureAuthentication and matchingChecksumAuthentication together and
	// be explicit that the resulting merged authenticator is exclusively for
	// checksums files.

	var ret []Hash
	sc := bufio.NewScanner(bytes.NewReader(s.Document))
	for sc.Scan() {
		parts := bytes.Fields(sc.Bytes())
		if len(parts) != 0 && len(parts) < 2 {
			// Doesn't look like a valid sums file line, so we'll assume
			// this whole thing isn't a checksums file.
			return nil
		}

		// If this is a checksums file then the first part should be a
		// hex-encoded SHA256 hash, so it should be 64 characters long
		// and contain only hex digits.
		hashStr := parts[0]
		if len(hashStr) != 64 {
			return nil // doesn't look like a checksums file
		}

		var gotSHA256Sum [sha256.Size]byte
		if _, err := hex.Decode(gotSHA256Sum[:], hashStr); err != nil {
			return nil // doesn't look like a checksums file
		}

		ret = append(ret, HashLegacyZipSHAFromSHA(gotSHA256Sum))
	}

	return ret
}

// findSigningKey attempts to verify the signature using each of the keys
// returned by the registry. If a valid signature is found, it returns the
// signing key.
//
// Note: currently the registry only returns one key, but this may change in
// the future.
func (s signatureAuthentication) findSigningKey() (*SigningKey, string, error) {
	for _, key := range s.Keys {
		keyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(key.ASCIIArmor))
		if err != nil {
			return nil, "", fmt.Errorf("error decoding signing key: %s", err)
		}

		entity, err := openpgp.CheckDetachedSignature(keyring, bytes.NewReader(s.Document), bytes.NewReader(s.Signature))

		// If the signature issuer does not match the the key, keep trying the
		// rest of the provided keys.
		if err == openpgpErrors.ErrUnknownIssuer {
			continue
		}

		// Any other signature error is terminal.
		if err != nil {
			return nil, "", fmt.Errorf("error checking signature: %s", err)
		}

		keyID := "n/a"
		if entity.PrimaryKey != nil {
			keyID = entity.PrimaryKey.KeyIdString()
		}

		log.Printf("[DEBUG] Provider signed by %s", entityString(entity))
		return &key, keyID, nil
	}

	// If none of the provided keys issued the signature, this package is
	// unsigned. This is currently a terminal authentication error.
	return nil, "", fmt.Errorf("authentication signature from unknown issuer")
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
