package customdiff

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
)

func TestForceNewIf(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		var condCalls int
		var gotOld1, gotNew1, gotOld2, gotNew2 string

		provider := testProvider(
			map[string]*schema.Schema{
				"foo": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
			ForceNewIf("foo", func(d *schema.ResourceDiff, meta interface{}) bool {
				// When we set "ForceNew", our CustomizeDiff function is actually
				// called a second time to construct the "create" portion of
				// the replace diff. On the second call, the old value is masked
				// as "" to suggest that the object is being created rather than
				// updated.

				condCalls++
				old, new := d.GetChange("foo")

				switch condCalls {
				case 1:
					gotOld1 = old.(string)
					gotNew1 = new.(string)
				case 2:
					gotOld2 = old.(string)
					gotNew2 = new.(string)
				}

				return true
			}),
		)

		diff, err := testDiff(
			provider,
			map[string]string{
				"foo": "bar",
			},
			map[string]string{
				"foo": "baz",
			},
		)

		if err != nil {
			t.Fatalf("Diff failed with error: %s", err)
		}

		if condCalls != 2 {
			t.Fatalf("Wrong number of conditional callback calls %d; want %d", condCalls, 2)
		} else {
			if got, want := gotOld1, "bar"; got != want {
				t.Errorf("wrong old value %q on first call; want %q", got, want)
			}
			if got, want := gotNew1, "baz"; got != want {
				t.Errorf("wrong new value %q on first call; want %q", got, want)
			}
			if got, want := gotOld2, ""; got != want {
				t.Errorf("wrong old value %q on first call; want %q", got, want)
			}
			if got, want := gotNew2, "baz"; got != want {
				t.Errorf("wrong new value %q on first call; want %q", got, want)
			}
		}

		if !diff.Attributes["foo"].RequiresNew {
			t.Error("Attribute 'foo' is not marked as RequiresNew")
		}
	})
	t.Run("false", func(t *testing.T) {
		var condCalls int
		var gotOld, gotNew string

		provider := testProvider(
			map[string]*schema.Schema{
				"foo": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
			ForceNewIf("foo", func(d *schema.ResourceDiff, meta interface{}) bool {
				condCalls++
				old, new := d.GetChange("foo")
				gotOld = old.(string)
				gotNew = new.(string)

				return false
			}),
		)

		diff, err := testDiff(
			provider,
			map[string]string{
				"foo": "bar",
			},
			map[string]string{
				"foo": "baz",
			},
		)

		if err != nil {
			t.Fatalf("Diff failed with error: %s", err)
		}

		if condCalls != 1 {
			t.Fatalf("Wrong number of conditional callback calls %d; want %d", condCalls, 1)
		} else {
			if got, want := gotOld, "bar"; got != want {
				t.Errorf("wrong old value %q on first call; want %q", got, want)
			}
			if got, want := gotNew, "baz"; got != want {
				t.Errorf("wrong new value %q on first call; want %q", got, want)
			}
		}

		if diff.Attributes["foo"].RequiresNew {
			t.Error("Attribute 'foo' is marked as RequiresNew, but should not be")
		}
	})
}

func TestForceNewIfChange(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		var condCalls int
		var gotOld1, gotNew1, gotOld2, gotNew2 string

		provider := testProvider(
			map[string]*schema.Schema{
				"foo": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
			ForceNewIfChange("foo", func(old, new, meta interface{}) bool {
				// When we set "ForceNew", our CustomizeDiff function is actually
				// called a second time to construct the "create" portion of
				// the replace diff. On the second call, the old value is masked
				// as "" to suggest that the object is being created rather than
				// updated.

				condCalls++

				switch condCalls {
				case 1:
					gotOld1 = old.(string)
					gotNew1 = new.(string)
				case 2:
					gotOld2 = old.(string)
					gotNew2 = new.(string)
				}

				return true
			}),
		)

		diff, err := testDiff(
			provider,
			map[string]string{
				"foo": "bar",
			},
			map[string]string{
				"foo": "baz",
			},
		)

		if err != nil {
			t.Fatalf("Diff failed with error: %s", err)
		}

		if condCalls != 2 {
			t.Fatalf("Wrong number of conditional callback calls %d; want %d", condCalls, 2)
		} else {
			if got, want := gotOld1, "bar"; got != want {
				t.Errorf("wrong old value %q on first call; want %q", got, want)
			}
			if got, want := gotNew1, "baz"; got != want {
				t.Errorf("wrong new value %q on first call; want %q", got, want)
			}
			if got, want := gotOld2, ""; got != want {
				t.Errorf("wrong old value %q on first call; want %q", got, want)
			}
			if got, want := gotNew2, "baz"; got != want {
				t.Errorf("wrong new value %q on first call; want %q", got, want)
			}
		}

		if !diff.Attributes["foo"].RequiresNew {
			t.Error("Attribute 'foo' is not marked as RequiresNew")
		}
	})
	t.Run("false", func(t *testing.T) {
		var condCalls int
		var gotOld, gotNew string

		provider := testProvider(
			map[string]*schema.Schema{
				"foo": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
			ForceNewIfChange("foo", func(old, new, meta interface{}) bool {
				condCalls++
				gotOld = old.(string)
				gotNew = new.(string)

				return false
			}),
		)

		diff, err := testDiff(
			provider,
			map[string]string{
				"foo": "bar",
			},
			map[string]string{
				"foo": "baz",
			},
		)

		if err != nil {
			t.Fatalf("Diff failed with error: %s", err)
		}

		if condCalls != 1 {
			t.Fatalf("Wrong number of conditional callback calls %d; want %d", condCalls, 1)
		} else {
			if got, want := gotOld, "bar"; got != want {
				t.Errorf("wrong old value %q on first call; want %q", got, want)
			}
			if got, want := gotNew, "baz"; got != want {
				t.Errorf("wrong new value %q on first call; want %q", got, want)
			}
		}

		if diff.Attributes["foo"].RequiresNew {
			t.Error("Attribute 'foo' is marked as RequiresNew, but should not be")
		}
	})
}
