package tree

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

//Boolean strings
const (
	True  = "true"
	False = "false"
)

//Bool is a boolean XPath type
type Bool bool

//ResValue satisfies the Res interface for Bool
func (b Bool) String() string {
	if b {
		return True
	}

	return False
}

//Bool satisfies the HasBool interface for Bool's
func (b Bool) Bool() Bool {
	return b
}

//Num satisfies the HasNum interface for Bool's
func (b Bool) Num() Num {
	if b {
		return Num(1)
	}

	return Num(0)
}

//Num is a number XPath type
type Num float64

//ResValue satisfies the Res interface for Num
func (n Num) String() string {
	if math.IsInf(float64(n), 0) {
		if math.IsInf(float64(n), 1) {
			return "Infinity"
		}
		return "-Infinity"
	}
	return fmt.Sprintf("%g", float64(n))
}

//Bool satisfies the HasBool interface for Num's
func (n Num) Bool() Bool {
	return n != 0
}

//Num satisfies the HasNum interface for Num's
func (n Num) Num() Num {
	return n
}

//String is string XPath type
type String string

//ResValue satisfies the Res interface for String
func (s String) String() string {
	return string(s)
}

//Bool satisfies the HasBool interface for String's
func (s String) Bool() Bool {
	return Bool(len(s) > 0)
}

//Num satisfies the HasNum interface for String's
func (s String) Num() Num {
	num, err := strconv.ParseFloat(strings.TrimSpace(string(s)), 64)
	if err != nil {
		return Num(math.NaN())
	}
	return Num(num)
}

//NodeSet is a node-set XPath type
type NodeSet []Node

//GetNodeNum converts the node to a string-value and to a number
func GetNodeNum(n Node) Num {
	return String(n.ResValue()).Num()
}

//String satisfies the Res interface for NodeSet
func (n NodeSet) String() string {
	if len(n) == 0 {
		return ""
	}

	return n[0].ResValue()
}

//Bool satisfies the HasBool interface for node-set's
func (n NodeSet) Bool() Bool {
	return Bool(len(n) > 0)
}

//Num satisfies the HasNum interface for NodeSet's
func (n NodeSet) Num() Num {
	return String(n.String()).Num()
}
