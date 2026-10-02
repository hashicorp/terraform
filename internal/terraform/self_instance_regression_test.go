// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func selfInstanceTestData(n int, kind string, badSibling bool) (*evaluationStateData, addrs.ResourceInstance) {
	addr := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_resource", Name: "each"}
	provider := addrs.NewDefaultProvider("test")
	rc := &configs.Resource{Mode: addr.Mode, Type: addr.Type, Name: addr.Name, Provider: provider}
	key := func(i int) addrs.InstanceKey { return addrs.NoKey }
	switch kind {
	case "count":
		rc.Count = hcl.StaticExpr(cty.NumberIntVal(int64(n)), hcl.Range{})
		key = func(i int) addrs.InstanceKey { return addrs.IntKey(i) }
	case "for_each":
		rc.ForEach = hcl.StaticExpr(cty.EmptyObjectVal, hcl.Range{})
		key = func(i int) addrs.InstanceKey { return addrs.StringKey(fmt.Sprintf("instance-%d", i)) }
	}
	state := states.BuildState(func(ss *states.SyncState) {
		for i := 0; i < n; i++ {
			raw := []byte(fmt.Sprintf(`{"id":"instance-%d","secret":"private"}`, i))
			if badSibling && i != 0 {
				raw = []byte(`malformed`)
			}
			ss.SetResourceInstanceCurrent(addr.Instance(key(i)).Absolute(addrs.RootModuleInstance), &states.ResourceInstanceObjectSrc{Status: states.ObjectReady, AttrsJSON: raw, AttrSensitivePaths: []cty.Path{cty.GetAttrPath("secret")}}, addrs.AbsProviderConfig{Provider: provider, Module: addrs.RootModule})
		}
	}).SyncWrapper()
	schema := providers.Schema{Body: &configschema.Block{Attributes: map[string]*configschema.Attribute{"id": {Type: cty.String, Computed: true}, "secret": {Type: cty.String, Computed: true, Sensitive: true}}}}
	evaluator := &Evaluator{Operation: walkEval, Config: &configs.Config{Module: &configs.Module{ManagedResources: map[string]*configs.Resource{addr.String(): rc}}}, State: state, Changes: plans.NewChanges().SyncWrapper(), Deferrals: deferring.NewDeferred(false), Plugins: schemaOnlyProvidersForTesting(map[addrs.Provider]providers.ProviderSchema{provider: {ResourceTypes: map[string]providers.Schema{addr.Type: schema}}})}
	return &evaluationStateData{evaluationData: &evaluationData{Evaluator: evaluator}, Operation: walkEval}, addr.Instance(key(0))
}

func TestSelfInstanceResolutionMatchesAggregate(t *testing.T) {
	for _, kind := range []string{"single", "count", "for_each"} {
		t.Run(kind, func(t *testing.T) {
			n := 8
			if kind == "single" {
				n = 1
			}
			d, addr := selfInstanceTestData(n, kind, false)
			want, diags := d.GetResource(addr.Resource, tfdiags.SourceRange{})
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}
			switch key := addr.Key.(type) {
			case addrs.IntKey:
				want = want.Index(cty.NumberIntVal(int64(key)))
			case addrs.StringKey:
				want = want.GetAttr(string(key))
			}
			got, diags := d.GetResourceInstance(addr, tfdiags.SourceRange{})
			if diags.HasErrors() || !got.RawEquals(want) {
				t.Fatalf("instance differs: %s", diags.Err())
			}
			if !got.GetAttr("secret").HasMark(marks.Sensitive) {
				t.Fatal("sensitive mark lost")
			}
		})
	}
}
func TestSelfInstanceDoesNotDecodeUnreferencedSiblings(t *testing.T) {
	d, addr := selfInstanceTestData(128, "for_each", true)
	value, diags := d.GetResourceInstance(addr, tfdiags.SourceRange{})
	if diags.HasErrors() || !value.GetAttr("id").RawEquals(cty.StringVal("instance-0")) {
		t.Fatalf("self decoded unreferenced sibling: %s", diags.Err())
	}
}
func BenchmarkSelfInstanceResolution(b *testing.B) {
	for _, n := range []int{1, 16, 128} {
		for _, aggregate := range []bool{true, false} {
			name := "instance"
			if aggregate {
				name = "aggregate"
			}
			b.Run(fmt.Sprintf("%s/%d", name, n), func(b *testing.B) {
				d, addr := selfInstanceTestData(n, "for_each", false)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var diags tfdiags.Diagnostics
					if aggregate {
						_, diags = d.GetResource(addr.Resource, tfdiags.SourceRange{})
					} else {
						_, diags = d.GetResourceInstance(addr, tfdiags.SourceRange{})
					}
					if diags.HasErrors() {
						b.Fatal(diags.Err())
					}
				}
			})
		}
	}
}
