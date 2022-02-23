package ngaddrs

type OutputValue struct {
	Name string
}

type AbsOutputValue = Abs[LocalValue]

type ConfigOutputValue = Config[LocalValue]

func (addr OutputValue) String() string {
	// NOTE: There isn't actually any syntax for referring to an output
	// value in configuration directly -- it only ever appears as part of
	// the component group it's returned by -- so this notation is really
	// just for debugging and should not be shown to end-users.
	return "output." + addr.Name
}

func (addr OutputValue) UniqueKey() UniqueKey {
	// An OutputValue is comparable and so can be its own unique key
	return addr
}

func (addr OutputValue) uniqueKeySigil() {}
