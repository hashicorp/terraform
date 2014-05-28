package terraform

// State keeps track of a snapshot state-of-the-world that Terraform
// can use to keep track of what real world resources it is actually
// managing.
type State struct {
	resources map[string]resourceState
}

// resourceState is the state of a single resource.
//
// The ID is required and is some opaque string used to recognize
// the realized resource.
//
// Extra is arbitrary extra metadata that the resource provider returns
// that is sent back into the resource provider when it is needed.
type resourceState struct {
	ID    string
	Extra map[string]interface{}
}
