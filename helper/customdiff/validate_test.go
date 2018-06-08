package customdiff

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
)

func TestValidateChange(t *testing.T) {
	var called bool
	var gotOld, gotNew string

	provider := testProvider(
		map[string]*schema.Schema{
			"foo": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
		ValidateChange("foo", func(old, new, meta interface{}) error {
			called = true
			gotOld = old.(string)
			gotNew = new.(string)
			return errors.New("bad")
		}),
	)

	_, err := testDiff(
		provider,
		map[string]string{
			"foo": "bar",
		},
		map[string]string{
			"foo": "baz",
		},
	)

	if err == nil {
		t.Fatalf("Diff succeeded; want error")
	}
	if got, want := err.Error(), "bad"; got != want {
		t.Fatalf("wrong error message %q; want %q", got, want)
	}

	if !called {
		t.Fatal("ValidateChange callback was not called")
	}
	if got, want := gotOld, "bar"; got != want {
		t.Errorf("wrong old value %q; want %q", got, want)
	}
	if got, want := gotNew, "baz"; got != want {
		t.Errorf("wrong new value %q; want %q", got, want)
	}
}

func TestValidateValue(t *testing.T) {
	var called bool
	var gotValue string

	provider := testProvider(
		map[string]*schema.Schema{
			"foo": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
		ValidateValue("foo", func(value, meta interface{}) error {
			called = true
			gotValue = value.(string)
			return errors.New("bad")
		}),
	)

	_, err := testDiff(
		provider,
		map[string]string{
			"foo": "bar",
		},
		map[string]string{
			"foo": "baz",
		},
	)

	if err == nil {
		t.Fatalf("Diff succeeded; want error")
	}
	if got, want := err.Error(), "bad"; got != want {
		t.Fatalf("wrong error message %q; want %q", got, want)
	}

	if !called {
		t.Fatal("ValidateValue callback was not called")
	}
	if got, want := gotValue, "baz"; got != want {
		t.Errorf("wrong value %q; want %q", got, want)
	}
}
