package applying

import (
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/provisioners"
)

// Dependencies is an interface used to obtain external dependencies (providers
// and provisioners) needed to apply a plan.
//
// FIXME: This is just a copy of terraform.contextComponentFactory to work
// around the fact that we have no shared/exported interface for this right
// now. Perhaps in future we'll expose some interfaces directly from the
// "providers" and "provisioners" packages to cover these things, to simplify.
// For now, we're just using this to more easily glue this in to the old
// API in the "terraform" package.
type Dependencies interface {
	// ResourceProvider creates a new ResourceProvider with the given
	// type. The "uid" is a unique identifier for this provider being
	// initialized that can be used for internal tracking.
	ResourceProvider(typ, uid string) (providers.Interface, error)
	ResourceProviders() []string

	// ResourceProvisioner creates a new ResourceProvisioner with the
	// given type. The "uid" is a unique identifier for this provisioner
	// being initialized that can be used for internal tracking.
	ResourceProvisioner(typ, uid string) (provisioners.Interface, error)
	ResourceProvisioners() []string
}
