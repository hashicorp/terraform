package diff

// ChangeMeta represents metadata about changes in a diff. These do not
// affect the meaning of the diff itself but provide additional context
// that may be useful to an end-user when reading a diff.
type ChangeMeta struct {
	ForcesNew bool
}
