package refactoring

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestImpliedMoveStatements(t *testing.T) {
	resourceAddr := func(name string) addrs.AbsResource {
		return addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: name,
		}.Absolute(addrs.RootModuleInstance)
	}
	instObjState := func() *states.ResourceInstanceObjectSrc {
		return &states.ResourceInstanceObjectSrc{}
	}
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.MustParseProviderSourceString("hashicorp/foo"),
	}

	rootCfg, _ := loadRefactoringFixture(t, "testdata/move-statement-implied")
	prevRunState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			resourceAddr("formerly_count").Instance(addrs.IntKey(0)),
			instObjState(),
			providerAddr,
		)
		s.SetResourceInstanceCurrent(
			resourceAddr("formerly_count").Instance(addrs.IntKey(1)),
			instObjState(),
			providerAddr,
		)
		s.SetResourceInstanceCurrent(
			resourceAddr("now_count").Instance(addrs.NoKey),
			instObjState(),
			providerAddr,
		)
		s.SetResourceInstanceCurrent(
			resourceAddr("formerly_count_explicit").Instance(addrs.IntKey(0)),
			instObjState(),
			providerAddr,
		)
		s.SetResourceInstanceCurrent(
			resourceAddr("formerly_count_explicit").Instance(addrs.IntKey(1)),
			instObjState(),
			providerAddr,
		)
		s.SetResourceInstanceCurrent(
			resourceAddr("now_count_explicit").Instance(addrs.NoKey),
			instObjState(),
			providerAddr,
		)
		s.SetResourceInstanceCurrent(
			resourceAddr("now_for_each_formerly_count").Instance(addrs.IntKey(0)),
			instObjState(),
			providerAddr,
		)
		s.SetResourceInstanceCurrent(
			resourceAddr("now_for_each_formerly_no_count").Instance(addrs.NoKey),
			instObjState(),
			providerAddr,
		)

		// This "ambiguous" resource is representing a rare but possible
		// situation where we end up having a mixture of different index
		// types in the state at the same time. The main way to get into
		// this state would be to remove "count = 1" and then have the
		// provider fail to destroy the zero-key instance even though we
		// already created the no-key instance. Users can also get here
		// by using "terraform state mv" in weird ways.
		s.SetResourceInstanceCurrent(
			resourceAddr("ambiguous").Instance(addrs.NoKey),
			instObjState(),
			providerAddr,
		)
		s.SetResourceInstanceCurrent(
			resourceAddr("ambiguous").Instance(addrs.IntKey(0)),
			instObjState(),
			providerAddr,
		)
	})

	explicitStmts := FindMoveStatements(rootCfg)
	got := ImpliedMoveStatements(rootCfg, prevRunState, explicitStmts)
	want := []MoveStatement{
		{
			From:    addrs.ImpliedMoveStatementEndpoint(resourceAddr("formerly_count").Instance(addrs.IntKey(0)), tfdiags.SourceRange{}),
			To:      addrs.ImpliedMoveStatementEndpoint(resourceAddr("formerly_count").Instance(addrs.NoKey), tfdiags.SourceRange{}),
			Implied: true,
			DeclRange: tfdiags.SourceRange{
				Filename: "testdata/move-statement-implied/move-statement-implied.tf",
				Start:    tfdiags.SourcePos{Line: 5, Column: 1, Byte: 180},
				End:      tfdiags.SourcePos{Line: 5, Column: 32, Byte: 211},
			},
		},
		{
			From:    addrs.ImpliedMoveStatementEndpoint(resourceAddr("now_count").Instance(addrs.NoKey), tfdiags.SourceRange{}),
			To:      addrs.ImpliedMoveStatementEndpoint(resourceAddr("now_count").Instance(addrs.IntKey(0)), tfdiags.SourceRange{}),
			Implied: true,
			DeclRange: tfdiags.SourceRange{
				Filename: "testdata/move-statement-implied/move-statement-implied.tf",
				Start:    tfdiags.SourcePos{Line: 10, Column: 11, Byte: 282},
				End:      tfdiags.SourcePos{Line: 10, Column: 12, Byte: 283},
			},
		},

		// We generate foo.ambiguous[0] to foo.ambiguous here, even though
		// there's already a foo.ambiguous in the state, because it's the
		// responsibility of the later ApplyMoves step to deal with the
		// situation where an object wants to move into an address already
		// occupied by another object.
		{
			From:    addrs.ImpliedMoveStatementEndpoint(resourceAddr("ambiguous").Instance(addrs.IntKey(0)), tfdiags.SourceRange{}),
			To:      addrs.ImpliedMoveStatementEndpoint(resourceAddr("ambiguous").Instance(addrs.NoKey), tfdiags.SourceRange{}),
			Implied: true,
			DeclRange: tfdiags.SourceRange{
				Filename: "testdata/move-statement-implied/move-statement-implied.tf",
				Start:    tfdiags.SourcePos{Line: 46, Column: 1, Byte: 806},
				End:      tfdiags.SourcePos{Line: 46, Column: 27, Byte: 832},
			},
		},
	}

	sort.Slice(got, func(i, j int) bool {
		// This is just an arbitrary sort to make the result consistent
		// regardless of what order the ImpliedMoveStatements function
		// visits the entries in the state/config.
		return got[i].DeclRange.Start.Line < got[j].DeclRange.Start.Line
	})

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}
