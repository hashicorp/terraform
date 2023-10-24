package refactoring

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

type Moves struct {
	Statements []MoveStatement

	Changes addrs.Map[addrs.AbsResourceInstance, MoveSuccess]
	Blocked addrs.Map[addrs.AbsMoveable, MoveBlocked]
}

type MoveSuccess struct {
	From addrs.AbsResourceInstance
	To   addrs.AbsResourceInstance
}

type MoveBlocked struct {
	Wanted addrs.AbsMoveable
	Actual addrs.AbsMoveable
}

func NewMoves(stmts []MoveStatement) *Moves {
	return &Moves{
		Statements: stmts,
		Changes:    addrs.MakeMap[addrs.AbsResourceInstance, MoveSuccess](),
		Blocked:    addrs.MakeMap[addrs.AbsMoveable, MoveBlocked](),
	}
}

func (moves *Moves) RecordMove(oldAddr, newAddr addrs.AbsResourceInstance) {
	if prevMove, exists := moves.Changes.GetOk(oldAddr); exists {
		// If the old address was _already_ the result of a move then
		// we'll replace that entry so that our results summarize a chain
		// of moves into a single entry.
		moves.Changes.Remove(oldAddr)
		oldAddr = prevMove.From
	}
	moves.Changes.Put(newAddr, MoveSuccess{
		From: oldAddr,
		To:   newAddr,
	})
}

func (moves *Moves) RecordBlockage(newAddr, wantedAddr addrs.AbsMoveable) {
	moves.Blocked.Put(newAddr, MoveBlocked{
		Wanted: wantedAddr,
		Actual: newAddr,
	})
}

// AddrMoved returns true if and only if the given resource instance moved to
// a new address in the ApplyMoves call that the receiver is describing.
//
// If AddrMoved returns true, you can pass the same address to method OldAddr
// to find its original address prior to moving.
func (moves *Moves) AddrMoved(newAddr addrs.AbsResourceInstance) bool {
	return moves.Changes.Has(newAddr)
}

// OldAddr returns the old address of the given resource instance address, or
// just returns back the same address if the given instance wasn't affected by
// any move statements.
func (moves *Moves) OldAddr(newAddr addrs.AbsResourceInstance) addrs.AbsResourceInstance {
	change, ok := moves.Changes.GetOk(newAddr)
	if !ok {
		return newAddr
	}
	return change.From
}
