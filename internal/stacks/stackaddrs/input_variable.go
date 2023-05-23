package stackaddrs

type InputVariable struct {
	Name string
}

func (InputVariable) referenceableSigil()   {}
func (InputVariable) inStackConfigSigil()   {}
func (InputVariable) inStackInstanceSigil() {}

// ConfigInputVariable places an [InputVariable] in the context of a particular [Stack].
type ConfigInputVariable = InStackConfig[InputVariable]

// AbsInputVariable places an [InputVariable] in the context of a particular [StackInstance].
type AbsInputVariable = InStackInstance[InputVariable]
