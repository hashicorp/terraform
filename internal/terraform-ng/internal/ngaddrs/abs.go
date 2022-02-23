package ngaddrs

import (
	"fmt"
	"strings"
)

type Abs[T UniqueKeyStringer] struct {
	Container AbsComponentGroup
	Local     T
}

func (addr Abs[T]) String() string {
	if addr.Container.IsRoot() {
		return addr.Local.String()
	}

	return fmt.Sprintf("%s.%s", addr.Container.String(), addr.Local.String())
}

func (addr Abs[T]) UniqueKey() UniqueKey {
	return absUniqueKey{
		containerRaw: addr.Container.String(),
		localKey:     addr.Local.UniqueKey(),
	}
}

type Config[T UniqueKeyStringer] struct {
	CallPath ConfigComponentGroup
	Local    T
}

func (addr Config[T]) String() string {
	if addr.CallPath.IsRoot() {
		return addr.Local.String()
	}

	var buf strings.Builder
	for _, call := range addr.CallPath {
		buf.WriteString("component.")
		buf.WriteString(call.Name)
		buf.WriteString("[*].")
	}
	return buf.String()
}

func (addr Config[T]) UniqueKey() UniqueKey {
	return configUniqueKey{
		containerRaw: addr.CallPath.String(),
		localKey:     addr.Local.UniqueKey(),
	}
}

// all of the Config[T] types are "Evalable" because those types exist
// specifically to support the evaluation process.
func (addr Config[T]) evalableSigil() {}

type absUniqueKey struct {
	containerRaw string
	localKey     UniqueKey
}

func (k absUniqueKey) uniqueKeySigil() {}

type configUniqueKey struct {
	containerRaw string
	localKey     UniqueKey
}

func (k configUniqueKey) uniqueKeySigil() {}

type UniqueKeyStringer interface {
	UniqueKeyer
	String() string
}
