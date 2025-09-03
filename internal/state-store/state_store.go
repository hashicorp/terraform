package state_stores

import (
	"github.com/hashicorp/hcl/v2"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

type StateStoreDescriber interface {
	StoreType() string
	TypeHclRange() *hcl.Range
	DeclHclRange() *hcl.Range

	ProviderDetails() StateStoreProviderDescriber

	// IsConfig reports if the state store description comes from configuration
	// data, in contrast to backend state files. This can be used to control which
	// type of diagnostics are returned.
	IsConfig() bool
}

type StateStoreProviderDescriber interface {
	Name() string
	Addr() tfaddr.Provider
	DeclHclRange() *hcl.Range
}
