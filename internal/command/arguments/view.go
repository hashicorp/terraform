// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package arguments

// View represents the global command-line arguments which configure the view.
type View struct {
	// NoColor is used to disable the use of terminal color codes in all
	// output.
	NoColor bool

	// CompactWarnings is used to coalesce duplicate warnings, to reduce the
	// level of noise when multiple instances of the same warning are raised
	// for a configuration.
	CompactWarnings bool
}

// ParseView processes CLI arguments, returning a View value and a
// possibly-modified slice of arguments. If any of the supported flags are
// found, they will be removed from the slice.
func ParseView(args []string) (*View, []string) {
	common := &View{}

	// Keep track of the length of the returned slice. When we find an
	// argument we support, i will not be incremented.
	i := 0
	for _, v := range args {
		switch v {
		case "-no-color":
			common.NoColor = true
		case "-compact-warnings":
			common.CompactWarnings = true
		default:
			// Unsupported argument: move left to the current position, and
			// increment the index.
			args[i] = v
			i++
		}
	}

	// Reduce the slice to the number of unsupported arguments. Any remaining
	// to the right of i have already been moved left.
	args = args[:i]

	return common, args
}
