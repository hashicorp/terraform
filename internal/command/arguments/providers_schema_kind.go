// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import "sort"

// Kind identifies a provider schema category that can be selected with the
// -kind flag of `terraform providers schema`.
//
// The canonical label vocabulary lives here (and not in jsonprovider) because
// jsonprovider must remain kind-ignorant apart from the resource-emission
// directive (see proposals/provider-subcommand-filtering/design_decisions.md
// #7). The command package owns the kind -> struct-field pruning and the
// translation to the jsonprovider directive.
type Kind string

const (
	KindProvider          Kind = "provider"
	KindResource          Kind = "resource"
	KindDataSource        Kind = "data-source"
	KindEphemeralResource Kind = "ephemeral-resource"
	KindListResource      Kind = "list-resource"
	KindFunction          Kind = "function"
	KindResourceIdentity  Kind = "resource-identity"
	KindAction            Kind = "action"
	KindStateStore        Kind = "state-store"
)

// providerSchemaKinds maps each canonical -kind label to whether it is
// map-backed (keyed by object type). The provider kind is the only
// non-map-backed kind, so -type does not apply to it.
var providerSchemaKinds = map[Kind]bool{
	KindProvider:          false,
	KindResource:          true,
	KindDataSource:        true,
	KindEphemeralResource: true,
	KindListResource:      true,
	KindFunction:          true,
	KindResourceIdentity:  true,
	KindAction:            true,
	KindStateStore:        true,
}

// ParseProviderSchemaKind validates a raw -kind value against the canonical
// labels, returning the Kind and whether it is valid. No plural, shorthand, or
// alternate spellings are accepted.
func ParseProviderSchemaKind(raw string) (Kind, bool) {
	k := Kind(raw)
	_, ok := providerSchemaKinds[k]
	return k, ok
}

// IsMapBacked reports whether the kind is keyed by object type (and therefore
// compatible with -type). Only the provider kind is not map-backed.
func (k Kind) IsMapBacked() bool {
	return providerSchemaKinds[k]
}

// ProviderSchemaKinds returns the canonical -kind labels sorted alphabetically,
// suitable for inclusion in help text and diagnostics.
func ProviderSchemaKinds() []string {
	labels := make([]string, 0, len(providerSchemaKinds))
	for k := range providerSchemaKinds {
		labels = append(labels, string(k))
	}
	sort.Strings(labels)
	return labels
}
