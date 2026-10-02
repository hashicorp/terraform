// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestSelfInstancePendingDeletionMatchesAggregate(t *testing.T) {
	d, first := selfInstanceTestData(4, "count", false)
	addr := first.Resource.Instance(addrs.IntKey(1))
	d.Evaluator.Changes.AppendResourceInstanceChange(&plans.ResourceInstanceChange{Addr: addr.Absolute(addrs.RootModuleInstance), Change: plans.Change{Action: plans.Delete}})
	all, wantDiags := d.GetResource(addr.Resource, tfdiags.SourceRange{})
	want, indexDiags := hcl.Index(all, cty.NumberIntVal(1), nil)
	wantDiags = wantDiags.Append(indexDiags)
	got, gotDiags := d.GetResourceInstance(addr, tfdiags.SourceRange{})
	if wantDiags.HasErrors() != gotDiags.HasErrors() || !got.RawEquals(want) {
		t.Fatalf("pending deletion differs: got %s %s; want %s %s", got.GoString(), gotDiags.Err(), want.GoString(), wantDiags.Err())
	}
}
