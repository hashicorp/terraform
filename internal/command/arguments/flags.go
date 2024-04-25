// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"flag"
	"fmt"
)

// FlagStringSlice is a flag.Value implementation which allows collecting
// multiple instances of a single flag into a slice. This is used for flags
// such as -target=aws_instance.foo and -var x=y.
type FlagStringSlice []string

var _ flag.Value = (*FlagStringSlice)(nil)

func (v *FlagStringSlice) String() string {
	return ""
}
func (v *FlagStringSlice) Set(raw string) error {
	*v = append(*v, raw)

	return nil
}

// FlagNameValueSlice is a flag.Value implementation that appends raw flag
// names and values to a slice. This is used to collect a sequence of flags
// with possibly different names, preserving the overall order.
type FlagNameValueSlice struct {
	FlagName string
	Items    *[]FlagNameValue
}

var _ flag.Value = FlagNameValueSlice{}

func NewFlagNameValueSlice(flagName string) FlagNameValueSlice {
	var items []FlagNameValue
	return FlagNameValueSlice{
		FlagName: flagName,
		Items:    &items,
	}
}

func (f FlagNameValueSlice) Empty() bool {
	if f.Items == nil {
		return true
	}
	return len(*f.Items) == 0
}

func (f FlagNameValueSlice) AllItems() []FlagNameValue {
	if f.Items == nil {
		return nil
	}
	return *f.Items
}

func (f FlagNameValueSlice) Alias(flagName string) FlagNameValueSlice {
	return FlagNameValueSlice{
		FlagName: flagName,
		Items:    f.Items,
	}
}

func (f FlagNameValueSlice) String() string {
	return ""
}

func (f FlagNameValueSlice) Set(str string) error {
	*f.Items = append(*f.Items, FlagNameValue{
		Name:  f.FlagName,
		Value: str,
	})
	return nil
}

type FlagNameValue struct {
	Name  string
	Value string
}

func (f FlagNameValue) String() string {
	return fmt.Sprintf("%s=%q", f.Name, f.Value)
}

// FlagIsSet returns whether a flag is explicitly set in a set of flags
func FlagIsSet(flags *flag.FlagSet, name string) bool {
	isSet := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == name {
			isSet = true
		}
	})
	return isSet
}
