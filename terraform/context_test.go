package terraform

import (
	"testing"
)

func TestContextValidate(t *testing.T) {
	config := testConfig(t, "validate-good")
	c := testContext(t, &ContextOpts{
		Config: config,
	})

	w, e := c.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) > 0 {
		t.Fatalf("bad: %#v", e)
	}
}

func TestContextValidate_badVar(t *testing.T) {
	config := testConfig(t, "validate-bad-var")
	c := testContext(t, &ContextOpts{
		Config: config,
	})

	w, e := c.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) == 0 {
		t.Fatalf("bad: %#v", e)
	}
}

func TestContextValidate_requiredVar(t *testing.T) {
	config := testConfig(t, "validate-required-var")
	c := testContext(t, &ContextOpts{
		Config: config,
	})

	w, e := c.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) == 0 {
		t.Fatalf("bad: %#v", e)
	}
}

func testContext(t *testing.T, opts *ContextOpts) *Context {
	return NewContext(opts)
}
