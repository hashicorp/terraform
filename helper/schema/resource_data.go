package schema

// ResourceData is used to query and set the attributes of a resource.
type ResourceData struct{}

// Get returns the data for the given key, or nil if the key doesn't exist.
func (d *ResourceData) Get(key string) interface{} {
	return nil
}
