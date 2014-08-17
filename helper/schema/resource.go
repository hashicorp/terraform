package schema

import (
	"errors"

	"github.com/hashicorp/terraform/terraform"
)

// The functions below are the CRUD function types for a Resource.
type CreateFunc func(*ResourceData) error
type ReadFunc func(*ResourceData) error
type UpdateFunc func(*ResourceData) error
type DeleteFunc func(*ResourceData) error

// Resource represents a thing in Terraform that has a set of configurable
// attributes and generally also has a lifecycle (create, read, update,
// delete).
//
// The Resource schema is an abstraction that allows provider writers to
// worry only about CRUD operations while off-loading validation, diff
// generation, etc. to this higher level library.
type Resource struct {
	Schema map[string]*Schema

	Create CreateFunc
	Read   ReadFunc
	Update UpdateFunc
	Delete DeleteFunc
}

// Diff returns a diff of this resource and is API compatible with the
// ResourceProvider interface.
func (r *Resource) Diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
	return schemaMap(r.Schema).Diff(s, c)
}

// Validate validates the resource configuration against the schema.
func (r *Resource) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	return schemaMap(r.Schema).Validate(c)
}

// InternalValidate should be called to validate the structure
// of the resource.
//
// This should be called in a unit test for any resource to verify
// before release that a resource is properly configured for use with
// this library.
func (r *Resource) InternalValidate() error {
	if r == nil {
		return errors.New("resource is nil")
	}

	return schemaMap(r.Schema).InternalValidate()
}
