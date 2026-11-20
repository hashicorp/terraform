// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

type EvaluationStage int

//go:generate go tool golang.org/x/tools/cmd/stringer -type=EvaluationStage

const (
	InvalidStage EvaluationStage = iota

	InitEvaluationStage
	PlanEvaluationStage
	ApplyEvaluationStage
	QueryEvaluationStage
)
