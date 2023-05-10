// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arguments

import (
	"flag"
	"fmt"
)

// flagStringSlice is a flag.Value implementation which allows collecting
// multiple instances of a single flag into a slice. This is used for flags
// such as -target=aws_instance.foo and -var x=y.
type flagStringSlice []string

var _ flag.Value = (*flagStringSlice)(nil)

func (v *flagStringSlice) String() string {
	return ""
}
func (v *flagStringSlice) Set(raw string) error {
	*v = append(*v, raw)

	return nil
}

// flagNameValueSlice is a flag.Value implementation that appends raw flag
// names and values to a slice. This is used to collect a sequence of flags
// with possibly different names, preserving the overall order.
//
// FIXME: this is a copy of rawFlags from command/meta_config.go, with the
// eventual aim of replacing it altogether by gathering variables in the
// arguments package.
type flagNameValueSlice struct {
	flagName string
	items    *[]FlagNameValue
}

var _ flag.Value = flagNameValueSlice{}

func newFlagNameValueSlice(flagName string) flagNameValueSlice {
	var items []FlagNameValue
	return flagNameValueSlice{
		flagName: flagName,
		items:    &items,
	}
}

func (f flagNameValueSlice) Empty() bool {
	if f.items == nil {
		return true
	}
	return len(*f.items) == 0
}

func (f flagNameValueSlice) AllItems() []FlagNameValue {
	if f.items == nil {
		return nil
	}
	return *f.items
}

func (f flagNameValueSlice) Alias(flagName string) flagNameValueSlice {
	return flagNameValueSlice{
		flagName: flagName,
		items:    f.items,
	}
}

func (f flagNameValueSlice) String() string {
	return ""
}

func (f flagNameValueSlice) Set(str string) error {
	*f.items = append(*f.items, FlagNameValue{
		Name:  f.flagName,
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
