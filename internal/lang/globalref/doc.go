// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package globalref is home to some analysis algorithms that aim to answer
// questions about references between objects and object attributes across
// an entire configuration.
//
// This is a different problem than references within a single module, which
// we handle using some relatively simpler functions in the "lang" package
// in the parent directory. The globalref algorithms are often implemented
// in terms of those module-local reference-checking functions.
package globalref
