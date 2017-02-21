package terraform

import (
	"testing"
)

func TestSMCUserVariables(t *testing.T) {
	c := testConfig(t, "smc-uservars")

	// Required variables not set
	errs := smcUserVariables(c, nil)
	if len(errs) == 0 {
		t.Fatal("should have errors")
	}

	// Required variables set, optional variables unset
	errs = smcUserVariables(c, map[string]interface{}{"foo": "bar"})
	if len(errs) != 0 {
		t.Fatalf("err: %#v", errs)
	}

	// Mapping element override
	errs = smcUserVariables(c, map[string]interface{}{
		"foo":     "bar",
		"map.foo": "baz",
	})
	if len(errs) == 0 {
		t.Fatalf("err: %#v", errs)
	}

	// Mapping complete override
	errs = smcUserVariables(c, map[string]interface{}{
		"foo": "bar",
		"map": "baz",
	})
	if len(errs) == 0 {
		t.Fatal("should have errors")
	}

}

func TestSMCUserVariables_mapFromJSON(t *testing.T) {
	c := testConfig(t, "uservars-map")

	// ensure that a single map in a list can satisfy a map variable, since it
	// will be coerced later to a map
	err := smcUserVariables(c, map[string]interface{}{
		"test_map": []map[string]interface{}{
			map[string]interface{}{
				"foo": "bar",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
