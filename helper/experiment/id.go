package experiment

// ID represents an experimental feature.
//
// The global vars defined on this package should be used as ID values.
// This interface is purposely not implement-able outside of this package
// so that we can rely on the Go compiler to enforce all experiment references.
type ID interface {
	Env() string
	Flag() string
	Default() bool

	unexported() // So the ID can't be implemented externally.
}

// basicID implements ID.
type basicID struct {
	EnvValue     string
	FlagValue    string
	DefaultValue bool
}

func newBasicID(flag, env string, def bool) ID {
	return &basicID{
		EnvValue:     env,
		FlagValue:    flag,
		DefaultValue: def,
	}
}

func (id *basicID) Env() string   { return id.EnvValue }
func (id *basicID) Flag() string  { return id.FlagValue }
func (id *basicID) Default() bool { return id.DefaultValue }
func (id *basicID) unexported()   {}
