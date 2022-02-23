package ngaddrs

import (
	"strings"
)

type ComponentGroupCall struct {
	Name string
}

func (addr ComponentGroupCall) String() string {
	return "group." + addr.Name
}

func (addr ComponentGroupCall) UniqueKey() UniqueKey {
	// This type is comparable and so can be its own unique key
	return addr
}

func (addr ComponentGroupCall) uniqueKeySigil() {}

type ComponentGroupCallInstance struct {
	Name string
	Key  InstanceKey
}

type AbsComponentGroupCall = Abs[ComponentGroupCall]

type ConfigComponentGroupCall = Config[ComponentGroupCall]

type AbsComponentGroup []ComponentGroupCallInstance

type ConfigComponentGroup []ComponentGroupCall

// RootComponentGroupInstance is the topmost object in the tree of component groups
// and components, containing all other components and component groups.
var RootComponentGroup AbsComponentGroup

func (addr AbsComponentGroup) IsRoot() bool {
	return len(addr) == 0
}

func (addr ConfigComponentGroup) IsRoot() bool {
	return len(addr) == 0
}

func (addr AbsComponentGroup) Child(name string, key InstanceKey) AbsComponentGroup {
	return append(addr[0:len(addr):len(addr)], ComponentGroupCallInstance{
		Name: name,
		Key:  key,
	})
}

func (addr ConfigComponentGroup) Child(name string) ConfigComponentGroup {
	return append(addr[0:len(addr):len(addr)], ComponentGroupCall{
		Name: name,
	})
}

func (addr AbsComponentGroup) Parent() AbsComponentGroup {
	if addr.IsRoot() {
		panic("root component group has no parent")
	}
	return addr[:len(addr)-1]
}

func (addr ConfigComponentGroup) Parent() ConfigComponentGroup {
	if addr.IsRoot() {
		panic("root component group call has no parent")
	}
	return addr[:len(addr)-1]
}

func (addr AbsComponentGroup) Call() AbsComponentGroupCall {
	if addr.IsRoot() {
		panic("root component group has no call")
	}
	return AbsComponentGroupCall{
		Container: addr.Parent(),
		Local:     ComponentGroupCall{Name: addr[len(addr)-1].Name},
	}
}

func (addr AbsComponentGroup) String() string {
	var buf strings.Builder
	for i, step := range addr {
		if i > 0 {
			buf.WriteByte('.')
		}
		buf.WriteString("component.")
		buf.WriteString(step.Name)
		if step.Key != nil {
			buf.WriteString(step.Key.String())
		}
	}
	return buf.String()
}

func (addr ConfigComponentGroup) String() string {
	var buf strings.Builder
	for i, step := range addr {
		if i > 0 {
			buf.WriteByte('.')
		}
		buf.WriteString("component.")
		buf.WriteString(step.Name)
		buf.WriteString("[*]")
	}
	return buf.String()
}
