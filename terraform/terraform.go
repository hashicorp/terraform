package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// Terraform is the primary structure that is used to interact with
// Terraform from code, and can perform operations such as returning
// all resources, a resource tree, a specific resource, etc.
type Terraform struct {
	config    *config.Config
	providers []ResourceProvider
}

// Config is the configuration that must be given to instantiate
// a Terraform structure.
type Config struct {
	Config    *config.Config
	Providers map[string]ResourceProviderFactory
	Variables map[string]string
}

// New creates a new Terraform structure, initializes resource providers
// for the given configuration, etc.
//
// Semantic checks of the entire configuration structure are done at this
// time, as well as richer checks such as verifying that the resource providers
// can be properly initialized, can be configured, etc.
func New(c *Config) (*Terraform, error) {
	return nil, nil
}

func (t *Terraform) Apply(*State, *Diff) (*State, error) {
	return nil, nil
}

func (t *Terraform) Diff(*State) (*Diff, error) {
	return nil, nil
}

func (t *Terraform) Refresh(*State) (*State, error) {
	return nil, nil
}
