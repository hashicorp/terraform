// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package chunks

const (
	// DefaultStateStoreChunkSize is the default chunk size proposed
	// to the provider.
	// This can be tweaked but should provide reasonable performance
	// trade-offs for average network conditions and state file sizes.
	DefaultStateStoreChunkSize int64 = 8 << 20 // 8 MB

	// MaxStateStoreChunkSize is the highest chunk size provider may choose
	// which we still consider reasonable/safe.
	// This reflects terraform-plugin-go's max. RPC message size of 256MB
	// and leaves plenty of space for other variable data like diagnostics.
	MaxStateStoreChunkSize int64 = 128 << 20 // 128 MB
)
