package stackaddrs

import "github.com/hashicorp/terraform/internal/collections"

type OutputValue struct {
	Name string
}

func (OutputValue) inStackConfigSigil()   {}
func (OutputValue) inStackInstanceSigil() {}

func (v OutputValue) String() string {
	return "output." + v.Name
}

func (v OutputValue) UniqueKey() collections.UniqueKey[OutputValue] {
	return v
}

// An OutputValue is its own [collections.UniqueKey].
func (OutputValue) IsUniqueKey(OutputValue) {}

// ConfigOutputValue places an [OutputValue] in the context of a particular [Stack].
type ConfigOutputValue = InStackConfig[OutputValue]

// AbsOutputValue places an [OutputValue] in the context of a particular [StackInstance].
type AbsOutputValue = InStackInstance[OutputValue]
