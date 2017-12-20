package customdiff

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
)

func TestIf(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		var condCalled, customCalled bool
		var gotOld, gotNew string

		provider := testProvider(
			map[string]*schema.Schema{
				"foo": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
			If(
				func(d *schema.ResourceDiff, meta interface{}) bool {
					condCalled = true
					old, new := d.GetChange("foo")
					gotOld = old.(string)
					gotNew = new.(string)
					return true
				},
				func(d *schema.ResourceDiff, meta interface{}) error {
					customCalled = true
					return errors.New("bad")
				},
			),
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
			t.Fatal("Diff succeeded; want error")
		}
		if got, want := err.Error(), "bad"; got != want {
			t.Fatalf("wrong error message %q; want %q", got, want)
		}

		if !condCalled {
			t.Error("condition callback was not called")
		} else {
			if got, want := gotOld, "bar"; got != want {
				t.Errorf("wrong old value %q; want %q", got, want)
			}
			if got, want := gotNew, "baz"; got != want {
				t.Errorf("wrong new value %q; want %q", got, want)
			}
		}

		if !customCalled {
			t.Error("customize callback was not called")
		}
	})
	t.Run("false", func(t *testing.T) {
		var condCalled, customCalled bool
		var gotOld, gotNew string

		provider := testProvider(
			map[string]*schema.Schema{
				"foo": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
			If(
				func(d *schema.ResourceDiff, meta interface{}) bool {
					condCalled = true
					old, new := d.GetChange("foo")
					gotOld = old.(string)
					gotNew = new.(string)
					return false
				},
				func(d *schema.ResourceDiff, meta interface{}) error {
					customCalled = true
					return errors.New("bad")
				},
			),
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

		if err != nil {
			t.Fatalf("Diff error %q; want success", err.Error())
		}

		if !condCalled {
			t.Error("condition callback was not called")
		} else {
			if got, want := gotOld, "bar"; got != want {
				t.Errorf("wrong old value %q; want %q", got, want)
			}
			if got, want := gotNew, "baz"; got != want {
				t.Errorf("wrong new value %q; want %q", got, want)
			}
		}

		if customCalled {
			t.Error("customize callback was called (should not have been)")
		}
	})
}

func TestIfValueChange(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		var condCalled, customCalled bool
		var gotOld, gotNew string

		provider := testProvider(
			map[string]*schema.Schema{
				"foo": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
			IfValueChange(
				"foo",
				func(old, new, meta interface{}) bool {
					condCalled = true
					gotOld = old.(string)
					gotNew = new.(string)
					return true
				},
				func(d *schema.ResourceDiff, meta interface{}) error {
					customCalled = true
					return errors.New("bad")
				},
			),
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
			t.Fatal("Diff succeeded; want error")
		}
		if got, want := err.Error(), "bad"; got != want {
			t.Fatalf("wrong error message %q; want %q", got, want)
		}

		if !condCalled {
			t.Error("condition callback was not called")
		} else {
			if got, want := gotOld, "bar"; got != want {
				t.Errorf("wrong old value %q; want %q", got, want)
			}
			if got, want := gotNew, "baz"; got != want {
				t.Errorf("wrong new value %q; want %q", got, want)
			}
		}

		if !customCalled {
			t.Error("customize callback was not called")
		}
	})
	t.Run("false", func(t *testing.T) {
		var condCalled, customCalled bool
		var gotOld, gotNew string

		provider := testProvider(
			map[string]*schema.Schema{
				"foo": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
			IfValueChange(
				"foo",
				func(old, new, meta interface{}) bool {
					condCalled = true
					gotOld = old.(string)
					gotNew = new.(string)
					return false
				},
				func(d *schema.ResourceDiff, meta interface{}) error {
					customCalled = true
					return errors.New("bad")
				},
			),
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

		if err != nil {
			t.Fatalf("Diff error %q; want success", err.Error())
		}

		if !condCalled {
			t.Error("condition callback was not called")
		} else {
			if got, want := gotOld, "bar"; got != want {
				t.Errorf("wrong old value %q; want %q", got, want)
			}
			if got, want := gotNew, "baz"; got != want {
				t.Errorf("wrong new value %q; want %q", got, want)
			}
		}

		if customCalled {
			t.Error("customize callback was called (should not have been)")
		}
	})
}

func TestIfValue(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		var condCalled, customCalled bool
		var gotValue string

		provider := testProvider(
			map[string]*schema.Schema{
				"foo": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
			IfValue(
				"foo",
				func(value, meta interface{}) bool {
					condCalled = true
					gotValue = value.(string)
					return true
				},
				func(d *schema.ResourceDiff, meta interface{}) error {
					customCalled = true
					return errors.New("bad")
				},
			),
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
			t.Fatal("Diff succeeded; want error")
		}
		if got, want := err.Error(), "bad"; got != want {
			t.Fatalf("wrong error message %q; want %q", got, want)
		}

		if !condCalled {
			t.Error("condition callback was not called")
		} else {
			if got, want := gotValue, "baz"; got != want {
				t.Errorf("wrong value %q; want %q", got, want)
			}
		}

		if !customCalled {
			t.Error("customize callback was not called")
		}
	})
	t.Run("false", func(t *testing.T) {
		var condCalled, customCalled bool
		var gotValue string

		provider := testProvider(
			map[string]*schema.Schema{
				"foo": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
			IfValue(
				"foo",
				func(value, meta interface{}) bool {
					condCalled = true
					gotValue = value.(string)
					return false
				},
				func(d *schema.ResourceDiff, meta interface{}) error {
					customCalled = true
					return errors.New("bad")
				},
			),
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

		if err != nil {
			t.Fatalf("Diff error %q; want success", err.Error())
		}

		if !condCalled {
			t.Error("condition callback was not called")
		} else {
			if got, want := gotValue, "baz"; got != want {
				t.Errorf("wrong value %q; want %q", got, want)
			}
		}

		if customCalled {
			t.Error("customize callback was called (should not have been)")
		}
	})
}
