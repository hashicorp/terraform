package regsrc

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/hashicorp/terraform/svchost"
)

var (
	// DefaultProviderNamespace represents the namespace for canonical
	// HashiCorp-controlled providers.
	DefaultProviderNamespace = "-"
)

// TerraformProvider describes a Terraform Registry Provider source.
type TerraformProvider struct {
	RawHost      *FriendlyHost
	RawNamespace string
	RawName      string
	OS           string
	Arch         string
}

// NewTerraformProvider constructs a new provider source.
func NewTerraformProvider(name, os, arch string) *TerraformProvider {
	if os == "" {
		os = runtime.GOOS
	}
	if arch == "" {
		arch = runtime.GOARCH
	}

	// separate namespace if included
	namespace := DefaultProviderNamespace
	if names := strings.SplitN(name, "/", 2); len(names) == 2 {
		namespace, name = names[0], names[1]
	}
	p := &TerraformProvider{
		RawHost:      PublicRegistryHost,
		RawNamespace: namespace,
		RawName:      name,
		OS:           os,
		Arch:         arch,
	}

	return p
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
