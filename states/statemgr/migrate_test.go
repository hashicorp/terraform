package statemgr

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
)

func TestCheckValidImport(t *testing.T) {
	barState := states.BuildState(func(s *states.SyncState) {
		s.SetOutputValue(
			addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("bar"), false,
		)
	})
	notBarState := states.BuildState(func(s *states.SyncState) {
		s.SetOutputValue(
			addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
			cty.StringVal("not bar"), false,
		)
	})
	emptyState := states.NewState()

	tests := map[string]struct {
		New      *statefile.File
		Existing *statefile.File
		WantErr  string
	}{
		"exact match": {
			New:      statefile.New(barState, "lineage", 1),
			Existing: statefile.New(barState, "lineage", 1),
			WantErr:  ``,
		},
		"overwrite unrelated empty state": {
			New:      statefile.New(barState, "lineage1", 1),
			Existing: statefile.New(emptyState, "lineage2", 1),
			WantErr:  ``,
		},
		"different state with same serial": {
			New:      statefile.New(barState, "lineage", 1),
			Existing: statefile.New(notBarState, "lineage", 1),
			WantErr:  `cannot overwrite existing state with serial 1 with a different state that has the same serial`,
		},
		"different state with newer serial": {
			New:      statefile.New(barState, "lineage", 2),
			Existing: statefile.New(notBarState, "lineage", 1),
			WantErr:  ``,
		},
		"different state with older serial": {
			New:      statefile.New(barState, "lineage", 1),
			Existing: statefile.New(notBarState, "lineage", 2),
			WantErr:  `cannot import state with serial 1 over newer state with serial 2`,
		},
		"different lineage with same serial": {
			New:      statefile.New(barState, "lineage1", 2),
			Existing: statefile.New(notBarState, "lineage2", 2),
			WantErr:  `cannot import state with lineage "lineage1" over unrelated state with lineage "lineage2"`,
		},
		"different lineage with different serial": {
			New:      statefile.New(barState, "lineage1", 3),
			Existing: statefile.New(notBarState, "lineage2", 2),
			WantErr:  `cannot import state with lineage "lineage1" over unrelated state with lineage "lineage2"`,
		},
		"new state is legacy": {
			New:      statefile.New(barState, "", 2),
			Existing: statefile.New(notBarState, "lineage", 2),
			WantErr:  ``,
		},
		"old state is legacy": {
			New:      statefile.New(barState, "lineage", 2),
			Existing: statefile.New(notBarState, "", 2),
			WantErr:  ``,
		},
		"both states are legacy": {
			New:      statefile.New(barState, "", 2),
			Existing: statefile.New(notBarState, "", 2),
			WantErr:  ``,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotErr := CheckValidImport(test.New, test.Existing)

			if test.WantErr == "" {
				if gotErr != nil {
					t.Errorf("unexpected error: %s", gotErr)
				}
			} else {
				if gotErr == nil {
					t.Errorf("succeeded, but want error: %s", test.WantErr)
				} else if got, want := gotErr.Error(), test.WantErr; got != want {
					t.Errorf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
			}
		})
	}
}
