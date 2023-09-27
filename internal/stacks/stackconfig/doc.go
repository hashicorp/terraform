// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package stackconfig deals with decoding and some static validation of the
// Terraform Stack language, which uses files with the suffixes .tfstack.hcl
// and .tfstack.json to describe a set of components to be planned and applied
// together.
//
// The Stack language has some elements that are intentionally similar to the
// main Terraform language (used to describe individual modules), but is
// currently implemented separately so they can evolve independently while
// the stacks language is still relatively new. Over time it might make sense
// to refactor so that there's only one implementation of each of the common
// elements, but we'll wait to see how similar things are once this language
// has been in real use for some time.
package stackconfig
