package regsrc

import (
	"fmt"

	"github.com/hashicorp/terraform/svchost"
)

var (
	// DefaultProviderNamespace represents the namespace for canonical
	// HashiCorp-controlled providers.
	// REVIEWERS: Naming things is hard.
	// * HashiCorpProviderNameSpace?
	// * OfficialP...?
	// * CanonicalP...?
	DefaultProviderNamespace = "terraform-providers"
)

// Provider describes a Terraform Registry Provider source.
type TerraformProvider struct {
	RawHost      *FriendlyHost
	RawNamespace string
	RawName      string
}

// NewProvider constructs a new provider source.
func NewTerraformProvider(name string) (*TerraformProvider, error) {
	p := &TerraformProvider{
		RawHost:      PublicRegistryHost,
		RawNamespace: DefaultProviderNamespace,
		RawName:      name,
	}
	return p, nil
}

// Provider returns just the registry ID of the provider
func (p *TerraformProvider) TerraformProvider() string {
	return fmt.Sprintf("%s/%s", p.RawNamespace, p.RawName)
}

// SvcHost returns the svchost.Hostname for this provider. The
// default PublicRegistryHost is returned.
func (p *TerraformProvider) SvcHost() (svchost.Hostname, error) {
	return svchost.ForComparison(PublicRegistryHost.Raw)
}
