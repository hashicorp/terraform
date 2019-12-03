package moduledeps

import (
	"strings"
)

// ProviderInstance describes a particular provider instance by its full name,
// like "null" or "aws.foo".
type ProviderInstance string

// Type returns the provider type of this instance. For example, for an instance
// named "aws.foo" the type is "aws".
func (p ProviderInstance) Type() string {
	t := string(p)
	if dotPos := strings.Index(t, "."); dotPos != -1 {
		t = t[:dotPos]
	}
	return t
}

// Alias returns the alias of this provider, if any. An instance named "aws.foo"
// has the alias "foo", while an instance named just "docker" has no alias,
// so the empty string would be returned.
func (p ProviderInstance) Alias() string {
	t := string(p)
	if dotPos := strings.Index(t, "."); dotPos != -1 {
		return t[dotPos+1:]
	}
	return ""
}
