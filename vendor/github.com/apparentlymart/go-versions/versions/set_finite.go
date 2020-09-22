package versions

// setFinite is the interface implemented by set implementations that
// represent a finite number of versions, and can thus list those versions.
type setFinite interface {
	isFinite() bool
	listVersions() List
}

// IsFinite returns true if the set represents a finite number of versions,
// and can thus support List without panicking.
func (s Set) IsFinite() bool {
	return isFinite(s.setI)
}

// List returns the specific versions represented by a finite list, in an
// undefined order. If desired, the caller can sort the resulting list
// using its Sort method.
//
// If the set is not finite, this method will panic. Use IsFinite to check
// unless a finite set was guaranteed by whatever operation(s) constructed
// the set.
func (s Set) List() List {
	finite, ok := s.setI.(setFinite)
	if !ok || !finite.isFinite() {
		panic("List called on infinite set")
	}
	return finite.listVersions()
}

func isFinite(s setI) bool {
	finite, ok := s.(setFinite)
	return ok && finite.isFinite()
}
