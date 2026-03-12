// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package ast provides a migration engine for Terraform HCL files built on
// top of hclwrite. It supports comment-preserving, roundtrip-safe mutations
// such as renaming attributes and blocks, changing values, and restructuring
// configuration to automate provider version migrations.
package ast
