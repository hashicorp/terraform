package customdiff

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
)

func TestComputedIf(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		var condCalls int
		var gotOld, gotNew string

		provider := testProvider(
			map[string]*schema.Schema{
				"foo": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"comp": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
			ComputedIf("comp", func(d *schema.ResourceDiff, meta interface{}) bool {
				// When we set "ForceNew", our CustomizeDiff function is actually
				// called a second time to construct the "create" portion of
				// the replace diff. On the second call, the old value is masked
				// as "" to suggest that the object is being created rather than
				// updated.

				condCalls++
				old, new := d.GetChange("foo")
				gotOld = old.(string)
				gotNew = new.(string)

				return true
			}),
		)

		diff, err := testDiff(
			provider,
			map[string]string{
				"foo":  "bar",
				"comp": "old",
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

		if !diff.Attributes["comp"].NewComputed {
			t.Error("Attribute 'comp' is not marked as NewComputed")
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
				"comp": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
			ComputedIf("comp", func(d *schema.ResourceDiff, meta interface{}) bool {
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
				"foo":  "bar",
				"comp": "old",
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

		if diff.Attributes["comp"] != nil && diff.Attributes["comp"].NewComputed {
			t.Error("Attribute 'foo' is marked as NewComputed, but should not be")
		}
	})
}
