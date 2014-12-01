package schema

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/terraform"
)

// Resource represents a thing in Terraform that has a set of configurable
// attributes and a lifecycle (create, read, update, delete).
//
// The Resource schema is an abstraction that allows provider writers to
// worry only about CRUD operations while off-loading validation, diff
// generation, etc. to this higher level library.
type Resource struct {
	// Schema is the schema for the configuration of this resource.
	//
	// The keys of this map are the configuration keys, and the values
	// describe the schema of the configuration value.
	//
	// The schema is used to represent both configurable data as well
	// as data that might be computed in the process of creating this
	// resource.
	Schema map[string]*Schema

	// The functions below are the CRUD operations for this resource.
	//
	// The only optional operation is Update. If Update is not implemented,
	// then updates will not be supported for this resource.
	//
	// The ResourceData parameter in the functions below are used to
	// query configuration and changes for the resource as well as to set
	// the ID, computed data, etc.
	//
	// The interface{} parameter is the result of the ConfigureFunc in
	// the provider for this resource. If the provider does not define
	// a ConfigureFunc, this will be nil. This parameter should be used
	// to store API clients, configuration structures, etc.
	//
	// If any errors occur during each of the operation, an error should be
	// returned. If a resource was partially updated, be careful to enable
	// partial state mode for ResourceData and use it accordingly.
	Create CreateFunc
	Read   ReadFunc
	Update UpdateFunc
	Delete DeleteFunc
}

// See Resource documentation.
type CreateFunc func(*ResourceData, interface{}) error

// See Resource documentation.
type ReadFunc func(*ResourceData, interface{}) error

// See Resource documentation.
type UpdateFunc func(*ResourceData, interface{}) error

// See Resource documentation.
type DeleteFunc func(*ResourceData, interface{}) error

func (r *Resource) FormatResourceConfig(
	c *terraform.ResourceConfig) (map[string]interface{}, error) {
	return r.formatResourceConfig(c.Config, r.Schema)
}

func (r *Resource) formatResourceConfig(
	cfg map[string]interface{},
	sm map[string]*Schema) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	source := getSourceSet | getSourceExact

	// Build a temp *ResourceData to use for the conversion
	tempD := &ResourceData{
		setMap: make(map[string]string),
	}

	for k, s := range sm {
		// If the value is nil, just move along...
		if cfg[k] == nil {
			continue
		}

		// If the value type is not a TypeSet, just add the key and
		// value to the result and move on
		if s.Type != TypeSet {
			result[k] = cfg[k]
			continue
		}

		// Set the entire list, this lets us get sane values out of it
		if err := tempD.setList(k, nil, s, cfg[k]); err != nil {
			return nil, err
		}

		hash := s.Set
		m := make(map[string]interface{})

		switch t := s.Elem.(type) {
		case *Schema:
			for i, v := range cfg[k].([]interface{}) {
				if v == nil {
					continue
				}

				// Get the current item from the list
				is := strconv.FormatInt(int64(i), 10)
				result := tempD.getList(k, []string{is}, s, source)

				// Continue if the result doesn't exist
				if !result.Exists {
					continue
				}

				// Calculate the hash and add the values to the map
				idx := hash(result.Value)
				m[strconv.Itoa(idx)] = v
			}
		case *Resource:
			for i, v := range cfg[k].([]map[string]interface{}) {
				if v == nil {
					continue
				}

				// Get the current item from the list
				is := strconv.FormatInt(int64(i), 10)
				result := tempD.getList(k, []string{is}, s, source)

				// Continue if the result doesn't exist
				if !result.Exists {
					continue
				}

				// Calculate the hash and get the formatted value
				// so it can be added to the map
				idx := hash(result.Value)
				fv, err := r.formatResourceConfig(v, t.Schema)
				if err != nil {
					return nil, err
				}
				m[strconv.Itoa(idx)] = fv
			}
		}

		if len(m) > 0 {
			result[k] = m
		}
	}

	return result, nil
}

// Apply creates, updates, and/or deletes a resource.
func (r *Resource) Apply(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	data, err := schemaMap(r.Schema).Data(s, d)
	if err != nil {
		return s, err
	}

	if s == nil {
		// The Terraform API dictates that this should never happen, but
		// it doesn't hurt to be safe in this case.
		s = new(terraform.InstanceState)
	}

	if d.Destroy || d.RequiresNew() {
		if s.ID != "" {
			// Destroy the resource since it is created
			if err := r.Delete(data, meta); err != nil {
				return data.State(), err
			}

			// Make sure the ID is gone.
			data.SetId("")
		}

		// If we're only destroying, and not creating, then return
		// now since we're done!
		if !d.RequiresNew() {
			return nil, nil
		}

		// Reset the data to be stateless since we just destroyed
		data, err = schemaMap(r.Schema).Data(nil, d)
		if err != nil {
			return nil, err
		}
	}

	err = nil
	if data.Id() == "" {
		// We're creating, it is a new resource.
		err = r.Create(data, meta)
	} else {
		if r.Update == nil {
			return s, fmt.Errorf("doesn't support update")
		}

		err = r.Update(data, meta)
	}

	return data.State(), err
}

// Diff returns a diff of this resource and is API compatible with the
// ResourceProvider interface.
func (r *Resource) Diff(
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
	return schemaMap(r.Schema).Diff(s, c)
}

// Validate validates the resource configuration against the schema.
func (r *Resource) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	return schemaMap(r.Schema).Validate(c)
}

// Refresh refreshes the state of the resource.
func (r *Resource) Refresh(
	s *terraform.InstanceState,
	meta interface{}) (*terraform.InstanceState, error) {
	data, err := schemaMap(r.Schema).Data(s, nil)
	if err != nil {
		return s, err
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
//
// Provider.InternalValidate() will automatically call this for all of
// the resources it manages, so you don't need to call this manually if it
// is part of a Provider.
func (r *Resource) InternalValidate() error {
	if r == nil {
		return errors.New("resource is nil")
	}

	return schemaMap(r.Schema).InternalValidate()
}
