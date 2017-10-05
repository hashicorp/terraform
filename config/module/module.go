package module

import "github.com/hashicorp/terraform/config"

// Module represents the metadata for a single module.
type Module struct {
	Name      string
	Source    string
	Version   string
	Providers []*config.ProviderConfig
}
