package schema

// FieldWriters are responsible for writing fields by address into
// a proper typed representation. ResourceData uses this to write new data
// into existing sources.
type FieldWriter interface {
	WriteField([]string, interface{}) error
}
