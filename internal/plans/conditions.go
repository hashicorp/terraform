package plans

import (
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

// Conditions describes a set of results for condition blocks evaluated during
// the planning process. In normal operation, each result will either represent
// a passed check (Result is cty.True) or a deferred check (Result is
// cty.UnknownVal(cty.Bool)). Failing checks result in errors, except in
// refresh-only mode.
//
// The map key is a string representation of the check rule address, which is
// globally unique. Condition blocks can be evaluated multiple times during the
// planning operation, so we must be able to update an existing result value.
type Conditions map[string]*ConditionResult

type ConditionResult struct {
	Address      addrs.Checkable
	Result       cty.Value
	Type         addrs.CheckType
	ErrorMessage string
}

func NewConditions() Conditions {
	return make(Conditions)
}

func (c Conditions) SyncWrapper() *ConditionsSync {
	return &ConditionsSync{
		results: c,
	}
}

// ConditionsSync is a wrapper around a Conditions that provides a
// concurrency-safe interface to add or update a condition result value.
type ConditionsSync struct {
	lock    sync.Mutex
	results Conditions
}

func (cs *ConditionsSync) SetResult(addr addrs.Check, result *ConditionResult) {
	if cs == nil {
		panic("SetResult on nil Conditions")
	}
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.results[addr.String()] = result
}
