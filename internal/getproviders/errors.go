package getproviders

import (
	"fmt"
	"net/url"

	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/addrs"
)

// ErrHostNoProviders is an error type used to indicate that a hostname given
// in a provider address does not support the provider registry protocol.
type ErrHostNoProviders struct {
	Hostname svchost.Hostname

	// HasOtherVersionis set to true if the discovery process detected
	// declarations of services named "providers" whose version numbers did not
	// match any version supported by the current version of Terraform.
	//
	// If this is set, it's helpful to hint to the user in an error message
	// that the provider host may be expecting an older or a newer version
	// of Terraform, rather than that it isn't a provider registry host at all.
	HasOtherVersion bool
}

func (err ErrHostNoProviders) Error() string {
	switch {
	case err.HasOtherVersion:
		return fmt.Sprintf("host %s does not support the provider registry protocol required by this Terraform version, but may be compatible with a different Terraform version", err.Hostname.ForDisplay())
	default:
		return fmt.Sprintf("host %s does not offer a Terraform provider registry", err.Hostname.ForDisplay())
	}
}

// ErrHostUnreachable is an error type used to indicate that a hostname
// given in a provider address did not resolve in DNS, did not respond to an
// HTTPS request for service discovery, or otherwise failed to correctly speak
// the service discovery protocol.
type ErrHostUnreachable struct {
	Hostname svchost.Hostname
	Wrapped  error
}

func (err ErrHostUnreachable) Error() string {
	return fmt.Sprintf("could not connect to %s: %s", err.Hostname.ForDisplay(), err.Wrapped.Error())
}

// Unwrap returns the underlying error that occurred when trying to reach the
// indicated host.
func (err ErrHostUnreachable) Unwrap() error {
	return err.Wrapped
}

// ErrUnauthorized is an error type used to indicate that a hostname
// given in a provider address returned a "401 Unauthorized" or "403 Forbidden"
// error response when we tried to access it.
type ErrUnauthorized struct {
	Hostname svchost.Hostname

	// HaveCredentials is true when the request that failed included some
	// credentials, and thus it seems that those credentials were invalid.
	// Conversely, HaveCredentials is false if the request did not include
	// credentials at all, in which case it seems that credentials must be
	// provided.
	HaveCredentials bool
}

func (err ErrUnauthorized) Error() string {
	switch {
	case err.HaveCredentials:
		return fmt.Sprintf("host %s rejected the given authentication credentials", err.Hostname)
	default:
		return fmt.Sprintf("host %s requires authentication credentials", err.Hostname)
	}
}

// ErrProviderNotFound is an error type used to indicate that requested provider
// was not found in the source(s) included in the Description field. This can be
// used to produce user-friendly error messages.
type ErrProviderNotFound struct {
	Provider addrs.Provider
	Sources  []string
}

func (err ErrProviderNotFound) Error() string {
	return fmt.Sprintf(
		"provider %s was not found in any of the search locations",
		err.Provider,
	)
}

// ErrRegistryProviderNotKnown is an error type used to indicate that the hostname
// given in a provider address does appear to be a provider registry but that
// registry does not know about the given provider namespace or type.
//
// A caller serving requests from an end-user should recognize this error type
// and use it to produce user-friendly hints for common errors such as failing
// to specify an explicit source for a provider not in the default namespace
// (one not under registry.terraform.io/hashicorp/). The default error message
// for this type is a direct description of the problem with no such hints,
// because we expect that the caller will have better context to decide what
// hints are appropriate, e.g. by looking at the configuration given by the
// user.
type ErrRegistryProviderNotKnown struct {
	Provider addrs.Provider
}

func (err ErrRegistryProviderNotKnown) Error() string {
	return fmt.Sprintf(
		"provider registry %s does not have a provider named %s",
		err.Provider.Hostname.ForDisplay(),
		err.Provider,
	)
}

// ErrPlatformNotSupported is an error type used to indicate that a particular
// version of a provider isn't available for a particular target platform.
//
// This is returned when DownloadLocation encounters a 404 Not Found response
// from the underlying registry, because it presumes that a caller will only
// ask for the DownloadLocation for a version it already found the existence
// of via AvailableVersions.
type ErrPlatformNotSupported struct {
	Provider addrs.Provider
	Version  Version
	Platform Platform

	// MirrorURL, if non-nil, is the base URL of the mirror that serviced
	// the request in place of the provider's origin registry. MirrorURL
	// is nil for a direct query.
	MirrorURL *url.URL
}

func (err ErrPlatformNotSupported) Error() string {
	if err.MirrorURL != nil {
		return fmt.Sprintf(
			"provider mirror %s does not have a package of %s %s for %s",
			err.MirrorURL.String(),
			err.Provider,
			err.Version,
			err.Platform,
		)
	}
	return fmt.Sprintf(
		"provider %s %s is not available for %s",
		err.Provider,
		err.Version,
		err.Platform,
	)
}

// ErrProtocolNotSupported is an error type used to indicate that a particular
// version of a provider is not supported by the current version of Terraform.
//
// Specfically, this is returned when the version's plugin protocol is not supported.
//
// When available, the error will include a suggested version that can be displayed to
// the user. Otherwise it will return UnspecifiedVersion
type ErrProtocolNotSupported struct {
	Provider   addrs.Provider
	Version    Version
	Suggestion Version
}

func (err ErrProtocolNotSupported) Error() string {
	return fmt.Sprintf(
		"provider %s %s is not supported by this version of terraform",
		err.Provider,
		err.Version,
	)
}

// ErrQueryFailed is an error type used to indicate that the hostname given
// in a provider address does appear to be a provider registry but that when
// we queried it for metadata for the given provider the server returned an
// unexpected error.
//
// This is used for any error responses other than "Not Found", which would
// indicate the absense of a provider and is thus reported using
// ErrProviderNotKnown instead.
type ErrQueryFailed struct {
	Provider addrs.Provider
	Wrapped  error

	// MirrorURL, if non-nil, is the base URL of the mirror that serviced
	// the request in place of the provider's origin registry. MirrorURL
	// is nil for a direct query.
	MirrorURL *url.URL
}

func (err ErrQueryFailed) Error() string {
	if err.MirrorURL != nil {
		return fmt.Sprintf(
			"failed to query provider mirror %s for %s: %s",
			err.MirrorURL.String(),
			err.Provider.String(),
			err.Wrapped.Error(),
		)
	}
	return fmt.Sprintf(
		"could not query provider registry for %s: %s",
		err.Provider.String(),
		err.Wrapped.Error(),
	)
}

// Unwrap returns the underlying error that occurred when trying to reach the
// indicated host.
func (err ErrQueryFailed) Unwrap() error {
	return err.Wrapped
}

// ErrIsNotExist returns true if and only if the given error is one of the
// errors from this package that represents an affirmative response that a
// requested object does not exist.
//
// This is as opposed to errors indicating that the source is unavailable
// or misconfigured in some way, where we therefore cannot say for certain
// whether the requested object exists.
//
// If a caller needs to take a special action based on something not existing,
// such as falling back on some other source, use this function rather than
// direct type assertions so that the set of possible "not exist" errors can
// grow in future.
func ErrIsNotExist(err error) bool {
	switch err.(type) {
	case ErrProviderNotFound, ErrRegistryProviderNotKnown, ErrPlatformNotSupported:
		return true
	default:
		return false
	}
}
