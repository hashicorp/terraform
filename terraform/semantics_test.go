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
	errs = smcUserVariables(c, map[string]string{"foo": "bar"})
	if len(errs) != 0 {
		t.Fatalf("err: %#v", errs)
	}

	// Mapping element override
	errs = smcUserVariables(c, map[string]string{
		"foo":     "bar",
		"map.foo": "baz",
	})
	if len(errs) != 0 {
		t.Fatalf("err: %#v", errs)
	}

	// Mapping complete override
	errs = smcUserVariables(c, map[string]string{
		"foo": "bar",
		"map": "baz",
	})
	if len(errs) == 0 {
		t.Fatal("should have errors")
	}

}
