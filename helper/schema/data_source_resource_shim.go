package schema

// DataSourceResourceShim takes a Resource instance describing a data source
// (with a Read implementation and a Schema, at least) and returns a new
// Resource instance with additional Create and Delete implementations that
// allow the data source to be used as a resource.
//
// This is a backward-compatibility layer for data sources that were formerly
// read-only resources before the data source concept was added. It should not
// be used for any *new* data sources.
//
// The Read function for the data source *must* call d.SetId with a non-empty
// id in order for this shim to function as expected.
//
// The provided Resource instance, and its schema, will be modified in-place
// to make it suitable for use as a full resource.
func DataSourceResourceShim(dataSource *Resource) *Resource {
	// TODO: Somehow emit a warning when a shim resource is used,
	// recommending that the user switch to using the data source
	// instead.

	// Recursively, in-place adjust the schema so that it has ForceNew
	// on any user-settable resource.
	dataSourceResourceShimAdjustSchema(dataSource.Schema)

	dataSource.Create = CreateFunc(dataSource.Read)
	dataSource.Delete = func(d *ResourceData, meta interface{}) error {
		d.SetId("")
		return nil
	}
	dataSource.Update = nil // should already be nil, but let's make sure

	return dataSource
}

func dataSourceResourceShimAdjustSchema(schema map[string]*Schema) {
	for _, s := range schema {
		// If the attribute is configurable then it must be ForceNew,
		// since we have no Update implementation.
		if s.Required || s.Optional {
			s.ForceNew = true
		}

		// If the attribute is a nested resource, we need to recursively
		// apply these same adjustments to it.
		if s.Elem != nil {
			if r, ok := s.Elem.(*Resource); ok {
				dataSourceResourceShimAdjustSchema(r.Schema)
			}
		}
	}
}
