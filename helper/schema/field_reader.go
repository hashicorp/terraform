package schema

// FieldReaders are responsible for decoding fields out of data into
// the proper typed representation. ResourceData uses this to query data
// out of multiple sources: config, state, diffs, etc.
type FieldReader interface {
	ReadField([]string, *Schema) (interface{}, bool, bool, error)
}
