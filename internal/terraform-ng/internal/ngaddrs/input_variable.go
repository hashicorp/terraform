package ngaddrs

type InputVariable struct {
	Name string
}

type AbsInputVariable = Abs[InputVariable]

type ConfigInputVariable = Config[InputVariable]

func (addr InputVariable) String() string {
	return "var." + addr.Name
}

func (addr InputVariable) UniqueKey() UniqueKey {
	// An InputVariable is comparable and so can be its own unique key
	return addr
}

func (addr InputVariable) uniqueKeySigil() {}
