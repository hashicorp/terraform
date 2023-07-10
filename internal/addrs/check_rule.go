// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import (
	"fmt"
)

// CheckRule is the address of a check rule within a checkable object.
//
// This represents the check rule globally within a configuration, and is used
// during graph evaluation to identify a condition result object to update with
// the result of check rule evaluation.
//
// The check address is not distinct from resource traversals, and check rule
// values are not intended to be available to the language, so the address is
// not Referenceable.
//
// Note also that the check address is only relevant within the scope of a run,
// as reordering check blocks between runs will result in their addresses
// changing. CheckRule is therefore for internal use only and should not be
// exposed in durable artifacts such as state snapshots.
type CheckRule struct {
	Container Checkable
	Type      CheckRuleType
	Index     int
}

func NewCheckRule(container Checkable, typ CheckRuleType, index int) CheckRule {
	return CheckRule{
		Container: container,
		Type:      typ,
		Index:     index,
	}
}

func (c CheckRule) String() string {
	container := c.Container.String()
	switch c.Type {
	case ResourcePrecondition:
		return fmt.Sprintf("%s.precondition[%d]", container, c.Index)
	case ResourcePostcondition:
		return fmt.Sprintf("%s.postcondition[%d]", container, c.Index)
	case OutputPrecondition:
		return fmt.Sprintf("%s.precondition[%d]", container, c.Index)
	case CheckDataResource:
		return fmt.Sprintf("%s.data[%d]", container, c.Index)
	case CheckAssertion:
		return fmt.Sprintf("%s.assert[%d]", container, c.Index)
	case InputValidation:
		return fmt.Sprintf("%s.validation[%d]", container, c.Index)
	default:
		// This should not happen
		return fmt.Sprintf("%s.condition[%d]", container, c.Index)
	}
}

func (c CheckRule) UniqueKey() UniqueKey {
	return checkRuleKey{
		ContainerKey: c.Container.UniqueKey(),
		Type:         c.Type,
		Index:        c.Index,
	}
}

type checkRuleKey struct {
	ContainerKey UniqueKey
	Type         CheckRuleType
	Index        int
}

func (k checkRuleKey) uniqueKeySigil() {}

// CheckRuleType describes a category of check. We use this only to establish
// uniqueness for Check values, and do not expose this concept of "check types"
// (which is subject to change in future) in any durable artifacts such as
// state snapshots.
//
// (See [CheckableKind] for an enumeration that we _do_ use externally, to
// describe the type of object being checked rather than the type of the check
// itself.)
type CheckRuleType int

//go:generate go run golang.org/x/tools/cmd/stringer -type=CheckRuleType check_rule.go

const (
	InvalidCondition      CheckRuleType = 0
	ResourcePrecondition  CheckRuleType = 1
	ResourcePostcondition CheckRuleType = 2
	OutputPrecondition    CheckRuleType = 3
	CheckDataResource     CheckRuleType = 4
	CheckAssertion        CheckRuleType = 5
	InputValidation       CheckRuleType = 6
)

// Description returns a human-readable description of the check type. This is
// presented in the user interface through a diagnostic summary.
func (c CheckRuleType) Description() string {
	switch c {
	case ResourcePrecondition:
		return "Resource precondition"
	case ResourcePostcondition:
		return "Resource postcondition"
	case OutputPrecondition:
		return "Module output value precondition"
	case CheckDataResource:
		return "Check block data resource"
	case CheckAssertion:
		return "Check block assertion"
	case InputValidation:
		return "Input variable validation"
	default:
		// This should not happen
		return "Condition"
	}
}
