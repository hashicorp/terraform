package intfns

import (
	"fmt"

	"github.com/ChrisTrenkamp/goxpath/tree"
	"golang.org/x/text/language"
)

func boolean(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	if b, ok := args[0].(tree.IsBool); ok {
		return b.Bool(), nil
	}

	return nil, fmt.Errorf("Cannot convert object to a boolean")
}

func not(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	b, ok := args[0].(tree.IsBool)
	if !ok {
		return nil, fmt.Errorf("Cannot convert object to a boolean")
	}
	return !b.Bool(), nil
}

func _true(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	return tree.Bool(true), nil
}

func _false(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	return tree.Bool(false), nil
}

func lang(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	lStr := args[0].String()

	var n tree.Elem

	for _, i := range c.NodeSet {
		if i.GetNodeType() == tree.NtElem {
			n = i.(tree.Elem)
		} else {
			n = i.GetParent()
		}

		for n.GetNodeType() != tree.NtRoot {
			if attr, ok := tree.GetAttribute(n, "lang", tree.XMLSpace); ok {
				return checkLang(lStr, attr.Value), nil
			}
			n = n.GetParent()
		}
	}

	return tree.Bool(false), nil
}

func checkLang(srcStr, targStr string) tree.Bool {
	srcLang := language.Make(srcStr)
	srcRegion, srcRegionConf := srcLang.Region()

	targLang := language.Make(targStr)
	targRegion, targRegionConf := targLang.Region()

	if srcRegionConf == language.Exact && targRegionConf != language.Exact {
		return tree.Bool(false)
	}

	if srcRegion != targRegion && srcRegionConf == language.Exact && targRegionConf == language.Exact {
		return tree.Bool(false)
	}

	_, _, conf := language.NewMatcher([]language.Tag{srcLang}).Match(targLang)
	return tree.Bool(conf >= language.High)
}
