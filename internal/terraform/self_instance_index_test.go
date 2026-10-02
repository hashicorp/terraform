// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strconv"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestSelfInstanceEveryIndexMatchesAggregate(t *testing.T) {
	for _, kind := range []string{"count", "for_each"} {
		d, first := selfInstanceTestData(16, kind, false)
		all, diags := d.GetResource(first.Resource, tfdiags.SourceRange{})
		if diags.HasErrors() {
			t.Fatal(diags.Err())
		}
		for i := 0; i < 16; i++ {
			var key addrs.InstanceKey = addrs.IntKey(i)
			var index = cty.NumberIntVal(int64(i))
			if kind == "for_each" {
				key = addrs.StringKey("instance-" + strconv.Itoa(i))
				index = cty.StringVal(string(key.(addrs.StringKey)))
			}
			want, errors := hcl.Index(all, index, nil)
			if errors.HasErrors() {
				t.Fatal(errors)
			}
			got, diags := d.GetResourceInstance(first.Resource.Instance(key), tfdiags.SourceRange{})
			if diags.HasErrors() || !got.RawEquals(want) {
				t.Fatalf("%s index %d differs from aggregate", kind, i)
			}
		}
	}
}
