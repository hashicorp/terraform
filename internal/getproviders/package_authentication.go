package getproviders

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// PackageAuthentication is an interface implemented by the optional package
// authentication implementations a source may include on its PackageMeta
// objects.
//
// A PackageAuthentication implementation is responsible for authenticating
// that a package is what its distributor intended to distribute and that it
// has not been tampered with.
type PackageAuthentication interface {
	// AuthenticatePackage takes the metadata about the package as returned
	// by its original source, and also the "localLocation" where it has
	// been staged for local inspection (which may or may not be the same
	// as the original source location) and returns an error if the
	// authentication checks fail.
	//
	// The localLocation is guaranteed not to be a PackageHTTPURL: a
	// remote package will always be staged locally for inspection first.
	AuthenticatePackage(meta PackageMeta, localLocation PackageLocation) error
}

type packageAuthenticationAll []PackageAuthentication

// PackageAuthenticationAll combines several authentications together into a
// single check value, which passes only if all of the given ones pass.
//
// The checks are processed in the order given, so a failure of an earlier
// check will prevent execution of a later one.
func PackageAuthenticationAll(checks ...PackageAuthentication) PackageAuthentication {
	return packageAuthenticationAll(checks)
}

func (checks packageAuthenticationAll) AuthenticatePackage(meta PackageMeta, localLocation PackageLocation) error {
	for _, check := range checks {
		err := check.AuthenticatePackage(meta, localLocation)
		if err != nil {
			return err
		}
	}
	return nil
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

func (a archiveHashAuthentication) AuthenticatePackage(meta PackageMeta, localLocation PackageLocation) error {
	archiveLocation, ok := localLocation.(PackageLocalArchive)
	if !ok {
		// A source should not use this authentication type for non-archive
		// locations.
		return fmt.Errorf("cannot check archive hash for non-archive location %s", localLocation)
	}

	f, err := os.Open(string(archiveLocation))
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return err
	}

	gotHash := h.Sum(nil)
	if !bytes.Equal(gotHash, a.WantSHA256Sum[:]) {
		return fmt.Errorf("archive has incorrect SHA-256 checksum %x (expected %x)", gotHash, a.WantSHA256Sum[:])
	}
	return nil
}
