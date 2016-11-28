package utils

import (
	"encoding/json"
)

type Nstring string

// NewNstring - create a new Nstring type
func NewNstring(s string) Nstring {
	return Nstring(s)
}

// UnmarshalJSON - convert byte stream to string
func (n *Nstring) UnmarshalJSON(b []byte) (err error) {
	if string(b) == "null" {
		return nil
	}
	return json.Unmarshal(b, (*string)(n))
}

func (n *Nstring) String() string {
	if n.IsNil() {
		return "null"
	} else {
		return string(*n)
	}
}

func (n *Nstring) Nil() {
	n = nil
}

func (n *Nstring) IsNil() bool {
	if len(*n) > 0 {
		return false
	} else {
		return true
	}
}
