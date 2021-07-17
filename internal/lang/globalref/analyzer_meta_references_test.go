package globalref

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
)

func TestAnalyzerMetaReferences(t *testing.T) {
	tests := []struct {
		InputContainer string
		InputRef       string
		WantRefs       []string
	}{
		{
			``,
			`local.a`,
			nil,
		},
		{
			``,
			`test_thing.single`,
			[]string{
				"::local.a",
				"::local.b",
			},
		},
		{
			``,
			`test_thing.single.string`,
			[]string{
				"::local.a",
			},
		},
		{
			``,
			`test_thing.for_each`,
			[]string{
				"::local.a",
				"::test_thing.single.string",
			},
		},
		{
			``,
			`test_thing.for_each["whatever"]`,
			[]string{
				"::local.a",
				"::test_thing.single.string",
			},
		},
		{
			``,
			`test_thing.for_each["whatever"].single`,
			[]string{
				"::test_thing.single.string",
			},
		},
		{
			``,
			`test_thing.for_each["whatever"].single.z`,
			[]string{
				"::test_thing.single.string",
			},
		},
		{
			``,
			`test_thing.count`,
			[]string{
				"::local.a",
			},
		},
		{
			``,
			`test_thing.count[0]`,
			[]string{
				"::local.a",
			},
		},
		{
			``,
			`module.single.a`,
			[]string{
				"module.single::test_thing.foo",
				"module.single::var.a",
			},
		},
		{
			``,
			`module.for_each["whatever"].a`,
			[]string{
				`module.for_each["whatever"]::test_thing.foo`,
				`module.for_each["whatever"]::var.a`,
			},
		},
		{
			``,
			`module.count[0].a`,
			[]string{
				`module.count[0]::test_thing.foo`,
				`module.count[0]::var.a`,
			},
		},
		{
			`module.single`,
			`var.a`,
			[]string{
				"::test_thing.single",
			},
		},
		{
			`module.single`,
			`test_thing.foo`,
			[]string{
				"module.single::var.a",
			},
		},
	}

	azr := testAnalyzer(t, "assorted")

	for _, test := range tests {
		name := test.InputRef
		if test.InputContainer != "" {
			name = test.InputContainer + " " + test.InputRef
		}
		t.Run(name, func(t *testing.T) {
			t.Logf("testing %s", name)
			var containerAddr addrs.Targetable
			containerAddr = addrs.RootModuleInstance
			if test.InputContainer != "" {
				moduleAddrTarget, diags := addrs.ParseTargetStr(test.InputContainer)
				if diags.HasErrors() {
					t.Fatalf("input module address is invalid: %s", diags.Err())
				}
				containerAddr = moduleAddrTarget.Subject
			}

			localRef, diags := addrs.ParseRefStr(test.InputRef)
			if diags.HasErrors() {
				t.Fatalf("input reference is invalid: %s", diags.Err())
			}

			ref := Reference{
				ContainerAddr: containerAddr,
				LocalRef:      localRef,
			}

			refs := azr.MetaReferences(ref)

			want := test.WantRefs
			var got []string
			for _, ref := range refs {
				got = append(got, ref.DebugString())
			}
			sort.Strings(got)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong references\n%s", diff)
			}
		})
	}
}
