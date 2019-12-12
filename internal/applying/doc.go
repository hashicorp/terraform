// Package applying contains the core logic for applying a plan that was
// previously generated.
//
// This package encapsulates the apply graph walk, executing operations
// concurrently and in a suitable order before returning the resulting new
// state and any diagnostics.
package applying
