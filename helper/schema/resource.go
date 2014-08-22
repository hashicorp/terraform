package schema

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/terraform"
)

// The functions below are the CRUD function types for a Resource.
//
// The second parameter is the meta value sent to the resource when
// different operations are called.
type CreateFunc func(*ResourceData, interface{}) error
type ReadFunc func(*ResourceData, interface{}) error
type UpdateFunc func(*ResourceData, interface{}) error
type DeleteFunc func(*ResourceData, interface{}) error

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

// Apply creates, updates, and/or deletes a resource.
func (r *Resource) Apply(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	data, err := schemaMap(r.Schema).Data(s, d)
	if err != nil {
		return s, err
	}

	if s == nil {
		// The Terraform API dictates that this should never happen, but
		// it doesn't hurt to be safe in this case.
		s = new(terraform.ResourceState)
	}

	if d.Destroy || d.RequiresNew() {
		if s.ID != "" {
			// Destroy the resource since it is created
			if err := r.Delete(data, meta); err != nil {
				return data.State(), err
			}

			// Reset the data to be empty
			data, err = schemaMap(r.Schema).Data(nil, d)
			if err != nil {
				return nil, err
			}
		}

		// If we're only destroying, and not creating, then return
		// now since we're done!
		if !d.RequiresNew() {
			return nil, nil
		}
	}

	err = nil
	if s.ID == "" {
		// We're creating, it is a new resource.
		err = r.Create(data, meta)
	} else {
		if r.Update == nil {
			return s, fmt.Errorf("%s doesn't support update", s.Type)
		}

		err = r.Update(data, meta)
	}

	return data.State(), err
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

// Refresh refreshes the state of the resource.
func (r *Resource) Refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	data, err := schemaMap(r.Schema).Data(s, nil)
	if err != nil {
		return nil, err
	}

	err = r.Read(data, meta)
	state := data.State()
	if state != nil && state.ID == "" {
		state = nil
	}

	return state, err
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
