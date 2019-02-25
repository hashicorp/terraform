package intfns

import (
	"fmt"
	"math"

	"github.com/ChrisTrenkamp/goxpath/tree"
)

func number(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	if b, ok := args[0].(tree.IsNum); ok {
		return b.Num(), nil
	}

	return nil, fmt.Errorf("Cannot convert object to a number")
}

func sum(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	n, ok := args[0].(tree.NodeSet)
	if !ok {
		return nil, fmt.Errorf("Cannot convert object to a node-set")
	}

	ret := 0.0
	for _, i := range n {
		ret += float64(tree.GetNodeNum(i))
	}

	return tree.Num(ret), nil
}

func floor(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	n, ok := args[0].(tree.IsNum)
	if !ok {
		return nil, fmt.Errorf("Cannot convert object to a number")
	}

	return tree.Num(math.Floor(float64(n.Num()))), nil
}

func ceiling(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	n, ok := args[0].(tree.IsNum)
	if !ok {
		return nil, fmt.Errorf("Cannot convert object to a number")
	}

	return tree.Num(math.Ceil(float64(n.Num()))), nil
}

func round(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	isn, ok := args[0].(tree.IsNum)
	if !ok {
		return nil, fmt.Errorf("Cannot convert object to a number")
	}

	n := isn.Num()

	if math.IsNaN(float64(n)) || math.IsInf(float64(n), 0) {
		return n, nil
	}

	if n < -0.5 {
		n = tree.Num(int(n - 0.5))
	} else if n > 0.5 {
		n = tree.Num(int(n + 0.5))
	} else {
		n = 0
	}

	return n, nil
}
