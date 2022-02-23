package ngaddrs

type LocalValue struct {
	Name string
}

type AbsLocalValue = Abs[LocalValue]

type ConfigLocalValue = Config[LocalValue]

func (addr LocalValue) String() string {
	return "local." + addr.Name
}

func (addr LocalValue) UniqueKey() UniqueKey {
	// An InputVariable is comparable and so can be its own unique key
	return addr
}

func (addr LocalValue) uniqueKeySigil() {}
