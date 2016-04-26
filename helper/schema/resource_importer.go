package schema

// ResourceImporter defines how a resource is imported in Terraform. This
// can be set onto a Resource struct to make it Importable. Not all resources
// have to be importable; if a Resource doesn't have a ResourceImporter then
// it won't be importable.
//
// "Importing" in Terraform is the process of taking an already-created
// resource and bringing it under Terraform management. This can include
// updating Terraform state, generating Terraform configuration, etc.
type ResourceImporter struct {
	// The functions below must all be implemented for importing to work.

	// State is called to convert an ID to one or more InstanceState to
	// insert into the Terraform state.
	State StateFunc
}

// StateFunc is the function called to import a resource into the
// Terraform state. It is given a ResourceData with only ID set. This
// ID is going to be an arbitrary value given by the user and may not map
// directly to the ID format that the resource expects, so that should
// be validated.
//
// This should return a slice of ResourceData that turn into the state
// that was imported. This might be as simple as returning only the argument
// that was given to the function. In other cases (such as AWS security groups),
// an import may fan out to multiple resources and this will have to return
// multiple.
type StateFunc func(*ResourceData, interface{}) ([]*ResourceData, error)
