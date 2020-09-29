package getproviders

import (
	"fmt"

	svchost "github.com/hashicorp/terraform-svchost"

	"github.com/hashicorp/terraform/addrs"
)

// LookupLegacyProvider attempts to resolve a legacy provider address (whose
// registry host and namespace are implied, rather than explicit) into a
// fully-qualified provider address, by asking the main Terraform registry
// to resolve it.
//
// If the given address is not a legacy provider address then it will just be
// returned verbatim without making any outgoing requests.
//
// Legacy provider lookup is possible only if the given source is either a
// *RegistrySource directly or if it is a MultiSource containing a
// *RegistrySource whose selector matching patterns include the
// public registry hostname registry.terraform.io.
//
// This is a backward-compatibility mechanism for compatibility with existing
// configurations that don't include explicit provider source addresses. New
// configurations should not rely on it, and this fallback mechanism is
// likely to be removed altogether in a future Terraform version.
func LookupLegacyProvider(addr addrs.Provider, source Source) (addrs.Provider, addrs.Provider, error) {
	if addr.Namespace != "-" {
		return addr, addrs.Provider{}, nil
	}
	if addr.Hostname != defaultRegistryHost { // condition above assures namespace is also "-"
		// Legacy providers must always belong to the default registry host.
		return addrs.Provider{}, addrs.Provider{}, fmt.Errorf("invalid provider type %q: legacy provider addresses must always belong to %s", addr, defaultRegistryHost)
	}

	// Now we need to derive a suitable *RegistrySource from the given source,
	// either directly or indirectly. This will not be possible if the user
	// has configured Terraform to disable direct installation from
	// registry.terraform.io; in that case, fully-qualified provider addresses
	// are always required.
	regSource := findLegacyProviderLookupSource(addr.Hostname, source)
	if regSource == nil {
		// This error message is assuming that the given Source was produced
		// based on the CLI configuration, which isn't necessarily true but
		// is true in all cases where this error message will ultimately be
		// presented to an end-user, so good enough for now.
		return addrs.Provider{}, addrs.Provider{}, fmt.Errorf("unqualified provider type %q cannot be resolved because direct installation from %s is disabled in the CLI configuration; declare an explicit provider namespace for this provider", addr.Type, addr.Hostname)
	}

	defaultNamespace, redirectNamespace, err := regSource.LookupLegacyProviderNamespace(addr.Hostname, addr.Type)
	if err != nil {
		return addrs.Provider{}, addrs.Provider{}, err
	}
	provider := addrs.Provider{
		Hostname:  addr.Hostname,
		Namespace: defaultNamespace,
		Type:      addr.Type,
	}
	var redirect addrs.Provider
	if redirectNamespace != "" {
		redirect = addrs.Provider{
			Hostname:  addr.Hostname,
			Namespace: redirectNamespace,
			Type:      addr.Type,
		}
	}

	return provider, redirect, nil
}

// findLegacyProviderLookupSource tries to find a *RegistrySource that can talk
// to the given registry host in the given Source. It might be given directly,
// or it might be given indirectly via a MultiSource where the selector
// includes a wildcard for registry.terraform.io.
//
// Returns nil if the given source does not have any configured way to talk
// directly to the given host.
//
// If the given source contains multiple sources that can talk to the given
// host directly, the first one in the sequence takes preference. In practice
// it's pointless to have two direct installation sources that match the same
// hostname anyway, so this shouldn't arise in normal use.
func findLegacyProviderLookupSource(host svchost.Hostname, source Source) *RegistrySource {
	switch source := source.(type) {

	case *RegistrySource:
		// Easy case: the source is a registry source directly, and so we'll
		// just use it.
		return source

	case *MemoizeSource:
		// Also easy: the source is a memoize wrapper, so defer to its
		// underlying source.
		return findLegacyProviderLookupSource(host, source.underlying)

	case MultiSource:
		// Trickier case: if it's a multisource then we need to scan over
		// its selectors until we find one that is a *RegistrySource _and_
		// that is configured to accept arbitrary providers from the
		// given hostname.

		// For our matching purposes we'll use an address that would not be
		// valid as a real provider FQN and thus can only match a selector
		// that has no filters at all or a selector that wildcards everything
		// except the hostname, like "registry.terraform.io/*/*"
		matchAddr := addrs.Provider{
			Hostname: host,
			// Other fields are intentionally left empty, to make this invalid
			// as a specific provider address.
		}

		for _, selector := range source {
			// If this source has suitable matching patterns to install from
			// the given hostname then we'll recursively search inside it
			// for *RegistrySource objects.
			if selector.CanHandleProvider(matchAddr) {
				ret := findLegacyProviderLookupSource(host, selector.Source)
				if ret != nil {
					return ret
				}
			}
		}

		// If we get here then there were no selectors that are both configured
		// to handle modules from the given hostname and that are registry
		// sources, so we fail.
		return nil

	default:
		// This source cannot be and cannot contain a *RegistrySource, so
		// we fail.
		return nil
	}
}
