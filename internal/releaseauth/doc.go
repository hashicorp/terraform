// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package releaseauth helps authenticates archives downloaded from a service
// like releases.hashicorp.com by providing some simple authentication tools:
//
//  1. Matching reported SHA-256 hash against a standard SHA256SUMS file.
//  2. Calculates the SHA-256 checksum of an archive and compares it against a reported hash.
//  3. Ensures the checksums were signed by HashiCorp.
package releaseauth
