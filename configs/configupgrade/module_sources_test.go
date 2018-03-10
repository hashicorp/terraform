package configupgrade

import (
	"reflect"
	"testing"

	"github.com/hashicorp/hcl2/hcl"
)

func TestMaybeAlreadyUpgraded(t *testing.T) {
	t.Run("already upgraded", func(t *testing.T) {
		sources, err := LoadModule("test-fixtures/already-upgraded")
		if err != nil {
			t.Fatal(err)
		}

		got, rng := sources.MaybeAlreadyUpgraded()
		if !got {
			t.Fatal("result is false, but want true")
		}
		gotRange := rng.ToHCL()
		wantRange := hcl.Range{
			Filename: "versions.tf",
			Start:    hcl.Pos{Line: 3, Column: 3, Byte: 15},
			End:      hcl.Pos{Line: 3, Column: 33, Byte: 45},
		}
		if !reflect.DeepEqual(gotRange, wantRange) {
			t.Errorf("wrong range\ngot:  %#v\nwant: %#v", gotRange, wantRange)
		}
	})
	t.Run("not yet upgraded", func(t *testing.T) {
		sources, err := LoadModule("test-fixtures/valid/noop/input")
		if err != nil {
			t.Fatal(err)
		}

		got, _ := sources.MaybeAlreadyUpgraded()
		if got {
			t.Fatal("result is true, but want false")
		}
	})
}
