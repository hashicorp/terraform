package resource

import (
	"github.com/hashicorp/terraform/terraform"
)

type Resource struct {
	Create  CreateFunc
	Diff    DiffFunc
	Refresh RefreshFunc
}

// CreateFunc is a function that creates a resource that didn't previously
// exist.
type CreateFunc func(
	*terraform.ResourceState,
	*terraform.ResourceDiff,
	interface{}) (*terraform.ResourceState, error)

// DiffFunc is a function that performs a diff of a resource.
type DiffFunc func(
	*terraform.ResourceState,
	*terraform.ResourceConfig,
	interface{}) (*terraform.ResourceDiff, error)

// RefreshFunc is a function that performs a refresh of a specific type
// of resource.
type RefreshFunc func(
	*terraform.ResourceState,
	interface{}) (*terraform.ResourceState, error)
