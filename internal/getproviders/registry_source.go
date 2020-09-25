package getproviders

import (
	"fmt"

	svchost "github.com/hashicorp/terraform-svchost"
	disco "github.com/hashicorp/terraform-svchost/disco"

	"github.com/hashicorp/terraform/addrs"
)

// RegistrySource is a Source that knows how to find and install providers from
// their originating provider registries.
type RegistrySource struct {
	services *disco.Disco
}

var _ Source = (*RegistrySource)(nil)

// NewRegistrySource creates and returns a new source that will install
// providers from their originating provider registries.
func NewRegistrySource(services *disco.Disco) *RegistrySource {
	return &RegistrySource{
		services: services,
	}
}

// AvailableVersions returns all of the versions available for the provider
// with the given address, or an error if that result cannot be determined.
//
// If the request fails, the returned error might be an value of
// ErrHostNoProviders, ErrHostUnreachable, ErrUnauthenticated,
// ErrProviderNotKnown, or ErrQueryFailed. Callers must be defensive and
// expect errors of other types too, to allow for future expansion.
func (s *RegistrySource) AvailableVersions(provider addrs.Provider) (VersionList, Warnings, error) {
	client, err := s.registryClient(provider.Hostname)
	if err != nil {
		return nil, nil, err
	}

	versionsResponse, warnings, err := client.ProviderVersions(provider)
	if err != nil {
		return nil, nil, err
	}

	if len(versionsResponse) == 0 {
		return nil, warnings, nil
	}

	// We ignore protocols here because our goal is to find out which versions
	// are available _at all_. Which ones are compatible with the current
	// Terraform becomes relevant only once we've selected one, at which point
	// we'll return an error if the selected one is incompatible.
	//
	// We intentionally produce an error on incompatibility, rather than
	// silently ignoring an incompatible version, in order to give the user
	// explicit feedback about why their selection wasn't valid and allow them
	// to decide whether to fix that by changing the selection or by some other
	// action such as upgrading Terraform, using a different OS to run
	// Terraform, etc. Changes that affect compatibility are considered breaking
	// changes from a provider API standpoint, so provider teams should change
	// compatibility only in new major versions.
	ret := make(VersionList, 0, len(versionsResponse))
	for str := range versionsResponse {
		v, err := ParseVersion(str)
		if err != nil {
			return nil, nil, ErrQueryFailed{
				Provider: provider,
				Wrapped:  fmt.Errorf("registry response includes invalid version string %q: %s", str, err),
			}
		}
		ret = append(ret, v)
	}
	ret.Sort() // lowest precedence first, preserving order when equal precedence
	return ret, warnings, nil
}

// PackageMeta returns metadata about the location and capabilities of
// a distribution package for a particular provider at a particular version
// targeting a particular platform.
//
// Callers of PackageMeta should first call AvailableVersions and pass
// one of the resulting versions to this function. This function cannot
// distinguish between a version that is not available and an unsupported
// target platform, so if it encounters either case it will return an error
// suggesting that the target platform isn't supported under the assumption
// that the caller already checked that the version is available at all.
//
// To find a package suitable for the platform where the provider installation
// process is running, set the "target" argument to
// getproviders.CurrentPlatform.
//
// If the request fails, the returned error might be an value of
// ErrHostNoProviders, ErrHostUnreachable, ErrUnauthenticated,
// ErrPlatformNotSupported, or ErrQueryFailed. Callers must be defensive and
// expect errors of other types too, to allow for future expansion.
func (s *RegistrySource) PackageMeta(provider addrs.Provider, version Version, target Platform) (PackageMeta, error) {
	client, err := s.registryClient(provider.Hostname)
	if err != nil {
		return PackageMeta{}, err
	}

	return client.PackageMeta(provider, version, target)
}

// LookupLegacyProviderNamespace is a special method available only on
// RegistrySource which can deal with legacy provider addresses that contain
// only a type and leave the namespace implied.
//
// It asks the registry at the given hostname to provide a default namespace
// for the given provider type, which can be combined with the given hostname
// and type name to produce a fully-qualified provider address.
//
// Not all unqualified type names can be resolved to a default namespace. If
// the request fails, this method returns an error describing the failure.
//
// This method exists only to allow compatibility with unqualified names
// in older configurations. New configurations should be written so as not to
// depend on it, and this fallback mechanism will likely be removed altogether
// in a future Terraform version.
func (s *RegistrySource) LookupLegacyProviderNamespace(hostname svchost.Hostname, typeName string) (string, string, error) {
	client, err := s.registryClient(hostname)
	if err != nil {
		return "", "", err
	}
	return client.LegacyProviderDefaultNamespace(typeName)
}

func (s *RegistrySource) registryClient(hostname svchost.Hostname) (*registryClient, error) {
	host, err := s.services.Discover(hostname)
	if err != nil {
		return nil, ErrHostUnreachable{
			Hostname: hostname,
			Wrapped:  err,
		}
	}

	url, err := host.ServiceURL("providers.v1")
	switch err := err.(type) {
	case nil:
		// okay! We'll fall through and return below.
	case *disco.ErrServiceNotProvided:
		return nil, ErrHostNoProviders{
			Hostname: hostname,
		}
	case *disco.ErrVersionNotSupported:
		return nil, ErrHostNoProviders{
			Hostname:        hostname,
			HasOtherVersion: true,
		}
	default:
		return nil, ErrHostUnreachable{
			Hostname: hostname,
			Wrapped:  err,
		}
	}

	// Check if we have credentials configured for this hostname.
	creds, err := s.services.CredentialsForHost(hostname)
	if err != nil {
		// This indicates that a credentials helper failed, which means we
		// can't do anything better than just pass through the helper's
		// own error message.
		return nil, fmt.Errorf("failed to retrieve credentials for %s: %s", hostname, err)
	}

	return newRegistryClient(url, creds), nil
}

func (s *RegistrySource) ForDisplay(provider addrs.Provider) string {
	return fmt.Sprintf("registry %s", provider.Hostname.ForDisplay())
}
