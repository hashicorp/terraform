package terraform

import (
	"bytes"
	"fmt"
	"sort"
)

func testDiffStr(d *Diff) string {
	var buf bytes.Buffer

	names := make([]string, len(d.Resources))
	for n, _ := range d.Resources {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, n := range names {
		r := d.Resources[n]
		buf.WriteString(fmt.Sprintf("%s\n", n))
		for attr, attrDiff := range r {
			buf.WriteString(fmt.Sprintf(
				"  %s: %#v => %#v\n",
				attr,
				attrDiff.Old,
				attrDiff.New,
			))
		}
	}

	return buf.String()
}
