package flaghelper

import (
	"strconv"
	"strings"
	"time"
)

// StringFlag implements the flag.Value interface and allows multiple
// calls to the same variable to append a list.
type StringFlag []string

func (s *StringFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *StringFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// FuncVar is a type of flag that accepts a function that is the string
// given
// by the user.
type FuncVar func(s string) error

func (f FuncVar) Set(s string) error { return f(s) }
func (f FuncVar) String() string     { return "" }
func (f FuncVar) IsBoolFlag() bool   { return false }

// FuncBoolVar is a type of flag that accepts a function, converts the
// user's
// value to a bool, and then calls the given function.
type FuncBoolVar func(b bool) error

func (f FuncBoolVar) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	return f(v)
}
func (f FuncBoolVar) String() string   { return "" }
func (f FuncBoolVar) IsBoolFlag() bool { return true }

// FuncDurationVar is a type of flag that
// accepts a function, converts the
// user's value to a duration, and then
// calls the given function.
type FuncDurationVar func(d time.Duration) error

func (f FuncDurationVar) Set(s string) error {
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	return f(v)
}
func (f FuncDurationVar) String() string   { return "" }
func (f FuncDurationVar) IsBoolFlag() bool { return false }
