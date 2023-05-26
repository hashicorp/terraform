package stackaddrs

type LocalValue struct {
	Name string
}

func (LocalValue) referenceableSigil()   {}
func (LocalValue) inStackConfigSigil()   {}
func (LocalValue) inStackInstanceSigil() {}

func (v LocalValue) String() string {
	return "local." + v.Name
}

// ConfigLocalValue places a [LocalValue] in the context of a particular [Stack].
type ConfigLocalValue = InStackConfig[LocalValue]

// AbsLocalValue places a [LocalValue] in the context of a particular [StackInstance].
type AbsLocalValue = InStackInstance[LocalValue]
