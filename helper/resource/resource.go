package resource

import (
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/terraform"
)

type Resource struct {
	ConfigValidator *config.Validator
	Create          CreateFunc
	Destroy         DestroyFunc
	Diff            DiffFunc
	Refresh         RefreshFunc
	Update          UpdateFunc
}

// CreateFunc is a function that creates a resource that didn't previously
// exist.
type CreateFunc func(
	*terraform.InstanceState,
	*terraform.InstanceDiff,
	interface{}) (*terraform.InstanceState, error)

// DestroyFunc is a function that destroys a resource that previously
// exists using the state.
type DestroyFunc func(
	*terraform.InstanceState,
	interface{}) error

// DiffFunc is a function that performs a diff of a resource.
type DiffFunc func(
	*terraform.InstanceState,
	*terraform.ResourceConfig,
	interface{}) (*terraform.InstanceDiff, error)

// RefreshFunc is a function that performs a refresh of a specific type
// of resource.
type RefreshFunc func(
	*terraform.InstanceState,
	interface{}) (*terraform.InstanceState, error)

// UpdateFunc is a function that is called to update a resource that
// previously existed. The difference between this and CreateFunc is that
// the diff is guaranteed to only contain attributes that don't require
// a new resource.
type UpdateFunc func(
	*terraform.InstanceState,
	*terraform.InstanceDiff,
	interface{}) (*terraform.InstanceState, error)
