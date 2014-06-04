package terraform

// State keeps track of a snapshot state-of-the-world that Terraform
// can use to keep track of what real world resources it is actually
// managing.
type State struct {
	resources map[string]ResourceState
}
