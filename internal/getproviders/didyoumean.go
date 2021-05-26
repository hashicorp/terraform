package getproviders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/hashicorp/go-retryablehttp"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/addrs"
)

// MissingProviderSuggestion takes a provider address that failed installation
// due to the remote registry reporting that it didn't exist, and attempts
// to find another provider that the user might have meant to select.
//
// If the result is equal to the given address then that indicates that there
// is no suggested alternative to offer, either because the function
// successfully determined there is no recorded alternative or because the
// lookup failed somehow. We don't consider a failure to find a suggestion
// as an installation failure, because the caller should already be reporting
// that the provider didn't exist anyway and this is only extra context for
// that error message.
//
// The result of this is a best effort, so any UI presenting it should be
// careful to give it only as a possibility and not necessarily a suitable
// replacement for the given provider.
//
// In practice today this function only knows how to suggest alternatives for
// "default" providers, which is to say ones that are in the hashicorp
// namespace in the Terraform registry. It will always return no result for
// any other provider. That might change in future if we introduce other ways
// to discover provider suggestions.
//
// If the given context is cancelled then this function might not return a
// renaming suggestion even if one would've been available for a completed
// request.
func MissingProviderSuggestion(ctx context.Context, addr addrs.Provider, source Source, reqs Requirements) addrs.Provider {
	if !addr.IsDefault() {
		return addr
	}

	// Before possibly looking up legacy naming, see if the user has another provider
	// named in their requirements that is of the same type, and offer that
	// as a suggestion
	for req := range reqs {
		if req != addr && req.Type == addr.Type {
			return req
		}
	}

	// Our strategy here, for a default provider, is to use the default
	// registry's special API for looking up "legacy" providers and try looking
	// for a legacy provider whose type name matches the type of the given
	// provider. This should then find a suitable answer for any provider
	// that was originally auto-installable in v0.12 and earlier but moved
	// into a non-default namespace as part of introducing the hierarchical
	// provider namespace.
	//
	// To achieve that, we need to find the direct registry client in
	// particular from the given source, because that is the only Source
	// implementation that can actually handle a legacy provider lookup.
	regSource := findLegacyProviderLookupSource(addr.Hostname, source)
	if regSource == nil {
		// If there's no direct registry source in the installation config
		// then we can't provide a renaming suggestion.
		return addr
	}

	defaultNS, redirectNS, err := regSource.lookupLegacyProviderNamespace(ctx, addr.Hostname, addr.Type)
	if err != nil {
		return addr
	}

	switch {
	case redirectNS != "":
		return addrs.Provider{
			Hostname:  addr.Hostname,
			Namespace: redirectNS,
			Type:      addr.Type,
		}
	default:
		return addrs.Provider{
			Hostname:  addr.Hostname,
			Namespace: defaultNS,
			Type:      addr.Type,
		}
	}
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

// lookupLegacyProviderNamespace is a special method available only on
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
func (s *RegistrySource) lookupLegacyProviderNamespace(ctx context.Context, hostname svchost.Hostname, typeName string) (string, string, error) {
	client, err := s.registryClient(hostname)
	if err != nil {
		return "", "", err
	}
	return client.legacyProviderDefaultNamespace(ctx, typeName)
}

// legacyProviderDefaultNamespace returns the raw address strings produced by
// the registry when asked about the given unqualified provider type name.
// The returned namespace string is taken verbatim from the registry's response.
//
// This method exists only to allow compatibility with unqualified names
// in older configurations. New configurations should be written so as not to
// depend on it.
func (c *registryClient) legacyProviderDefaultNamespace(ctx context.Context, typeName string) (string, string, error) {
	endpointPath, err := url.Parse(path.Join("-", typeName, "versions"))
	if err != nil {
		// Should never happen because we're constructing this from
		// already-validated components.
		return "", "", err
	}
	endpointURL := c.baseURL.ResolveReference(endpointPath)

	req, err := retryablehttp.NewRequest("GET", endpointURL.String(), nil)
	if err != nil {
		return "", "", err
	}
	req = req.WithContext(ctx)
	c.addHeadersToRequest(req.Request)

	// This is just to give us something to return in error messages. It's
	// not a proper provider address.
	placeholderProviderAddr := addrs.NewLegacyProvider(typeName)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", c.errQueryFailed(placeholderProviderAddr, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Great!
	case http.StatusNotFound:
		return "", "", ErrProviderNotFound{
			Provider: placeholderProviderAddr,
		}
	case http.StatusUnauthorized, http.StatusForbidden:
		return "", "", c.errUnauthorized(placeholderProviderAddr.Hostname)
	default:
		return "", "", c.errQueryFailed(placeholderProviderAddr, errors.New(resp.Status))
	}

	type ResponseBody struct {
		Id      string `json:"id"`
		MovedTo string `json:"moved_to"`
	}
	var body ResponseBody

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&body); err != nil {
		return "", "", c.errQueryFailed(placeholderProviderAddr, err)
	}

	provider, diags := addrs.ParseProviderSourceString(body.Id)
	if diags.HasErrors() {
		return "", "", fmt.Errorf("Error parsing provider ID from Registry: %s", diags.Err())
	}

	if provider.Type != typeName {
		return "", "", fmt.Errorf("Registry returned provider with type %q, expected %q", provider.Type, typeName)
	}

	var movedTo addrs.Provider
	if body.MovedTo != "" {
		movedTo, diags = addrs.ParseProviderSourceString(body.MovedTo)
		if diags.HasErrors() {
			return "", "", fmt.Errorf("Error parsing provider ID from Registry: %s", diags.Err())
		}

		if movedTo.Type != typeName {
			return "", "", fmt.Errorf("Registry returned provider with type %q, expected %q", movedTo.Type, typeName)
		}
	}

	return provider.Namespace, movedTo.Namespace, nil
}
