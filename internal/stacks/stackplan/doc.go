// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package stackplan contains the models and some business logic for stack-wide
// "meta-plans", which in practice are equivalent to multiple of what we
// traditionally think of as a "plan" in the non-stacks Terraform workflow,
// typically represented as a [plans.Plan] object.
//
// The stack plan model is intentionally slightly different from the original
// plan model because in the stack runtime we need to be able to split a
// traditional plan into smaller parts that we stream out to the caller as
// events, but the model here should be isomorphic so that we can translate
// to and from the models expected by the main Terraform language runtime.
package stackplan
