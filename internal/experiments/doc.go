// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package experiments contains the models and logic for opt-in experiments
// that can be activated for a particular Terraform module.
//
// We use experiments to get feedback on new configuration language features
// in a way that permits breaking changes without waiting for a future minor
// release. Any feature behind an experiment flag is subject to change in any
// way in even a patch release, until we have enough confidence about the
// design of the feature to make compatibility commitments about it.
package experiments
