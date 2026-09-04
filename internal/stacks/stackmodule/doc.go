// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package stackmodule deals with decoding and some static validation of the
// Terraform Language. It lives in the stacks package, because it still
// utilises the old way `BuildConfig` based way of loading Terraform
// configurations. The newer init-graph based approach doesn't work with Stacks
// yet, because it doesn't support an easy of suppling values for const vars.
package stackmodule
