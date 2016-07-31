// Copyright 2014 Alvaro J. Genial. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package form

import (
	"net/url"
	"strconv"
	"strings"
)

type node map[string]interface{}

func (n node) values(d, e rune) url.Values {
	vs := url.Values{}
	n.merge(d, e, "", &vs)
	return vs
}

func (n node) merge(d, e rune, p string, vs *url.Values) {
	for k, x := range n {
		switch y := x.(type) {
		case string:
			vs.Add(p+escape(d, e, k), y)
		case node:
			y.merge(d, e, p+escape(d, e, k)+string(d), vs)
		default:
			panic("value is neither string nor node")
		}
	}
}

// TODO: Add tests for implicit indexing.
func parseValues(d, e rune, vs url.Values, canIndexFirstLevelOrdinally bool) node {
	// NOTE: Because of the flattening of potentially multiple strings to one key, implicit indexing works:
	//    i. At the first level;   e.g. Foo.Bar=A&Foo.Bar=B     becomes 0.Foo.Bar=A&1.Foo.Bar=B
	//   ii. At the last level;    e.g. Foo.Bar._=A&Foo.Bar._=B becomes Foo.Bar.0=A&Foo.Bar.1=B
	// TODO: At in-between levels; e.g. Foo._.Bar=A&Foo._.Bar=B becomes Foo.0.Bar=A&Foo.1.Bar=B
	//       (This last one requires that there only be one placeholder in order for it to be unambiguous.)

	m := map[string]string{}
	for k, ss := range vs {
		indexLastLevelOrdinally := strings.HasSuffix(k, string(d)+implicitKey)

		for i, s := range ss {
			if canIndexFirstLevelOrdinally {
				k = strconv.Itoa(i) + string(d) + k
			} else if indexLastLevelOrdinally {
				k = strings.TrimSuffix(k, implicitKey) + strconv.Itoa(i)
			}

			m[k] = s
		}
	}

	n := node{}
	for k, s := range m {
		n = n.split(d, e, k, s)
	}
	return n
}

func splitPath(d, e rune, path string) (k, rest string) {
	esc := false
	for i, r := range path {
		switch {
		case !esc && r == e:
			esc = true
		case !esc && r == d:
			return unescape(d, e, path[:i]), path[i+1:]
		default:
			esc = false
		}
	}
	return unescape(d, e, path), ""
}

func (n node) split(d, e rune, path, s string) node {
	k, rest := splitPath(d, e, path)
	if rest == "" {
		return add(n, k, s)
	}
	if _, ok := n[k]; !ok {
		n[k] = node{}
	}

	c := getNode(n[k])
	n[k] = c.split(d, e, rest, s)
	return n
}

func add(n node, k, s string) node {
	if n == nil {
		return node{k: s}
	}

	if _, ok := n[k]; ok {
		panic("key " + k + " already set")
	}

	n[k] = s
	return n
}

func isEmpty(x interface{}) bool {
	switch y := x.(type) {
	case string:
		return y == ""
	case node:
		if s, ok := y[""].(string); ok {
			return s == ""
		}
		return false
	}
	panic("value is neither string nor node")
}

func getNode(x interface{}) node {
	switch y := x.(type) {
	case string:
		return node{"": y}
	case node:
		return y
	}
	panic("value is neither string nor node")
}

func getString(x interface{}) string {
	switch y := x.(type) {
	case string:
		return y
	case node:
		if s, ok := y[""].(string); ok {
			return s
		}
		return ""
	}
	panic("value is neither string nor node")
}

func escape(d, e rune, s string) string {
	s = strings.Replace(s, string(e), string(e)+string(e), -1) // Escape the escape    (\ => \\)
	s = strings.Replace(s, string(d), string(e)+string(d), -1) // Escape the delimiter (. => \.)
	return s
}

func unescape(d, e rune, s string) string {
	s = strings.Replace(s, string(e)+string(d), string(d), -1) // Unescape the delimiter (\. => .)
	s = strings.Replace(s, string(e)+string(e), string(e), -1) // Unescape the escape    (\\ => \)
	return s
}
