package disco

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

// OAuthClient represents an OAuth client configuration, which is used for
// unusual services that require an entire OAuth client configuration as part
// of their service discovery, rather than just a URL.
type OAuthClient struct {
	// ID is the identifier for the client, to be used as "client_id" in
	// OAuth requests.
	ID string

	// Authorization URL is the URL of the authorization endpoint that must
	// be used for this OAuth client, as defined in the OAuth2 specifications.
	//
	// Not all grant types use the authorization endpoint, so it may be omitted
	// if none of the grant types in SupportedGrantTypes require it.
	AuthorizationURL *url.URL

	// Token URL is the URL of the token endpoint that must be used for this
	// OAuth client, as defined in the OAuth2 specifications.
	//
	// Not all grant types use the token endpoint, so it may be omitted
	// if none of the grant types in SupportedGrantTypes require it.
	TokenURL *url.URL

	// MinPort and MaxPort define a range of TCP ports on localhost that this
	// client is able to use as redirect_uri in an authorization request.
	// Terraform will select a port from this range for the temporary HTTP
	// server it creates to receive the authorization response, giving
	// a URL like http://localhost:NNN/ where NNN is the selected port number.
	//
	// Terraform will reject any port numbers in this range less than 1024,
	// to respect the common convention (enforced on some operating systems)
	// that lower port numbers are reserved for "privileged" services.
	MinPort, MaxPort uint16

	// SupportedGrantTypes is a set of the grant types that the client may
	// choose from. This includes an entry for each distinct type advertised
	// by the server, even if a particular keyword is not supported by the
	// current version of Terraform.
	SupportedGrantTypes OAuthGrantTypeSet
}

// Endpoint returns an oauth2.Endpoint value ready to be used with the oauth2
// library, representing the URLs from the receiver.
func (c *OAuthClient) Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:  c.AuthorizationURL.String(),
		TokenURL: c.TokenURL.String(),

		// We don't actually auth because we're not a server-based OAuth client,
		// so this instead just means that we include client_id as an argument
		// in our requests.
		AuthStyle: oauth2.AuthStyleInParams,
	}
}

// OAuthGrantType is an enumeration of grant type strings that a host can
// advertise support for.
//
// Values of this type don't necessarily match with a known constant of the
// type, because they may represent grant type keywords defined in a later
// version of Terraform which this version doesn't yet know about.
type OAuthGrantType string

const (
	// OAuthAuthzCodeGrant represents an authorization code grant, as
	// defined in IETF RFC 6749 section 4.1.
	OAuthAuthzCodeGrant = OAuthGrantType("authz_code")

	// OAuthOwnerPasswordGrant represents a resource owner password
	// credentials grant, as defined in IETF RFC 6749 section 4.3.
	OAuthOwnerPasswordGrant = OAuthGrantType("password")
)

// UsesAuthorizationEndpoint returns true if the receiving grant type makes
// use of the authorization endpoint from the client configuration, and thus
// if the authorization endpoint ought to be required.
func (t OAuthGrantType) UsesAuthorizationEndpoint() bool {
	switch t {
	case OAuthAuthzCodeGrant:
		return true
	case OAuthOwnerPasswordGrant:
		return false
	default:
		// We'll default to false so that we don't impose any requirements
		// on any grant type keywords that might be defined for future
		// versions of Terraform.
		return false
	}
}

// UsesTokenEndpoint returns true if the receiving grant type makes
// use of the token endpoint from the client configuration, and thus
// if the authorization endpoint ought to be required.
func (t OAuthGrantType) UsesTokenEndpoint() bool {
	switch t {
	case OAuthAuthzCodeGrant:
		return true
	case OAuthOwnerPasswordGrant:
		return true
	default:
		// We'll default to false so that we don't impose any requirements
		// on any grant type keywords that might be defined for future
		// versions of Terraform.
		return false
	}
}

// OAuthGrantTypeSet represents a set of OAuthGrantType values.
type OAuthGrantTypeSet map[OAuthGrantType]struct{}

// NewOAuthGrantTypeSet constructs a new grant type set from the given list
// of grant type keyword strings. Any duplicates in the list are ignored.
func NewOAuthGrantTypeSet(keywords ...string) OAuthGrantTypeSet {
	ret := make(OAuthGrantTypeSet, len(keywords))
	for _, kw := range keywords {
		ret[OAuthGrantType(kw)] = struct{}{}
	}
	return ret
}

// Has returns true if the given grant type is in the receiving set.
func (s OAuthGrantTypeSet) Has(t OAuthGrantType) bool {
	_, ok := s[t]
	return ok
}

// RequiresAuthorizationEndpoint returns true if any of the grant types in
// the set are known to require an authorization endpoint.
func (s OAuthGrantTypeSet) RequiresAuthorizationEndpoint() bool {
	for t := range s {
		if t.UsesAuthorizationEndpoint() {
			return true
		}
	}
	return false
}

// RequiresTokenEndpoint returns true if any of the grant types in
// the set are known to require a token endpoint.
func (s OAuthGrantTypeSet) RequiresTokenEndpoint() bool {
	for t := range s {
		if t.UsesTokenEndpoint() {
			return true
		}
	}
	return false
}

// GoString implements fmt.GoStringer.
func (s OAuthGrantTypeSet) GoString() string {
	var buf strings.Builder
	i := 0
	buf.WriteString("disco.NewOAuthGrantTypeSet(")
	for t := range s {
		if i > 0 {
			buf.WriteString(", ")
		}
		fmt.Fprintf(&buf, "%q", string(t))
		i++
	}
	buf.WriteString(")")
	return buf.String()
}
