// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package format

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zclconf/go-cty/cty"
)

// CtyPath is a helper function to produce a user-friendly string
// representation of a cty.Path. The result uses a syntax similar to the
// HCL expression language in the hope of it being familiar to users.
func CtyPath(path cty.Path) string {
	var buf strings.Builder
	for _, step := range path {
		switch ts := step.(type) {
		case cty.GetAttrStep:
			fmt.Fprintf(&buf, ".%s", ts.Name)
		case cty.IndexStep:
			buf.WriteByte('[')
			key := ts.Key
			keyTy := key.Type()
			switch {
			case key.IsNull():
				buf.WriteString("null")
			case !key.IsKnown():
				buf.WriteString("(not yet known)")
			case keyTy == cty.Number:
				bf := key.AsBigFloat()
				buf.WriteString(bf.Text('g', -1))
			case keyTy == cty.String:
				buf.WriteString(strconv.Quote(key.AsString()))
			default:
				buf.WriteString("...")
			}
			buf.WriteByte(']')
		}
	}
	return buf.String()
}

// ErrorDiag is a helper function to produce a user-friendly string
// representation of certain special error types that we might want to
// include in diagnostic messages.
func ErrorDiag(err error) string {
	perr, ok := err.(cty.PathError)
	if !ok || len(perr.Path) == 0 {
		return err.Error()
	}

	return fmt.Sprintf("%s: %s", CtyPath(perr.Path), perr.Error())
}

// ErrorDiagPrefixed is like Error except that it presents any path
// information after the given prefix string, which is assumed to contain
// an HCL syntax representation of the value that errors are relative to.
func ErrorDiagPrefixed(err error, prefix string) string {
	perr, ok := err.(cty.PathError)
	if !ok || len(perr.Path) == 0 {
		return fmt.Sprintf("%s: %s", prefix, err.Error())
	}

	return fmt.Sprintf("%s%s: %s", prefix, CtyPath(perr.Path), perr.Error())
}
