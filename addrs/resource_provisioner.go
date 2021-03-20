package addrs

// AbsResourceInstanceProvisioner represents a specific provisioner block
// from a resource block, bound to a particular instance of that resource.
//
// Provisioners don't have user-provided identifiers, so an
// AbsResourceInstanceProvisioner address is valid only for a single
// graph walk and is not necessarily stable between operations.
type AbsResourceInstanceProvisioner struct {
	debuggable

	// ResourceInstance is the resource instance that the identified
	// provisioner block will run in the context of.
	ResourceInstance AbsResourceInstance

	// Index is the zero-based index into the sequence of provisioner blocks
	// within the resource configuration.
	Index int
}

func (p AbsResourceInstanceProvisioner) DebugAncestorFrames() []Debuggable {
	caller := p.ResourceInstance.DebugAncestorFrames()
	// It's safe for us to append to caller here because our
	// DebugAncestorFrames methods all construct fresh slices on each
	// call, either directly or indirectly.
	return append(caller, p.ResourceInstance)
}
