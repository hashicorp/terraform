package auth

import (
	"net/http"

	"github.com/hashicorp/terraform/svchost"
)

// Credentials is a list of CredentialsSource objects that can be tried in
// turn until one returns credentials for a host, or one returns an error.
//
// A Credentials is itself a CredentialsSource, wrapping its members.
// In principle one CredentialsSource can be nested inside another, though
// there is no good reason to do so.
type Credentials []CredentialsSource

// A CredentialsSource is an object that may be able to provide credentials
// for a given host.
//
// Credentials lookups are not guaranteed to be concurrency-safe. Callers
// using these facilities in concurrent code must use external concurrency
// primitives to prevent race conditions.
type CredentialsSource interface {
	// ForHost returns a non-nil HostCredentials if the source has credentials
	// available for the host, and a nil HostCredentials if it does not.
	//
	// If an error is returned, progress through a list of CredentialsSources
	// is halted and the error is returned to the user.
	ForHost(host svchost.Hostname) (HostCredentials, error)
}

// HostCredentials represents a single set of credentials for a particular
// host.
type HostCredentials interface {
	// PrepareRequest modifies the given request in-place to apply the
	// receiving credentials. The usual behavior of this method is to
	// add some sort of Authorization header to the request.
	PrepareRequest(req *http.Request)
}

// ForHost iterates over the contained CredentialsSource objects and
// tries to obtain credentials for the given host from each one in turn.
//
// If any source returns either a non-nil HostCredentials or a non-nil error
// then this result is returned. Otherwise, the result is nil, nil.
func (c Credentials) ForHost(host svchost.Hostname) (HostCredentials, error) {
	for _, source := range c {
		creds, err := source.ForHost(host)
		if creds != nil || err != nil {
			return creds, err
		}
	}
	return nil, nil
}
