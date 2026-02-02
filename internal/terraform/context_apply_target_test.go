package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	provider "github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestContextApply_TargetInstance(t *testing.T) {
	testConfig := testModuleInline(t, map[string]string{
		"main.tf": `
		
	locals {
		numbers = {
			a = resource.test_object.baz[*].test_number
			b = 0
		}
	}
	
	resource "test_object" "foo" {
		count = 2
		test_string = join("_", ["foo", count.index])
	}	
	
	resource "test_object" "baz" {
		count = 2
		test_number = 0
	}	

	resource "test_object" "bar" {
		count = 2
    	test_string = resource.test_object.foo[local.numbers.a[local.numbers.b]].test_string
	}
`,
	})

	p := simpleMockProvider()
	ctx := testContext2(
		t,
		&ContextOpts{
			Providers: map[addrs.Provider]provider.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			},
		},
	)

	testTarget := mustResourceInstanceAddr("test_object.bar[0]")
	planOptions := DefaultPlanOpts
	planOptions.Targets = []addrs.Targetable{
		testTarget,
	}
	planOptions.SafeTargeting = true
	pr, diags := ctx.Plan(testConfig, states.NewState(), planOptions)
	tfdiags.AssertNoErrors(t, diags)

	rs := pr.Changes.Resources
	for _, r := range rs {
		fmt.Printf("Resource: %s\n", r.Addr.String())
	}
	if len(rs) != 4 {
		t.Fatalf("expected 4 resources, got %d", len(rs))
	}

	t.Log("test")
}
