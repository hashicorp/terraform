// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package statefile

// looksLikeVersion0 sniffs for the signature indicating a version 0 state
// file.
//
// Version 0 was the number retroactively assigned to Terraform's initial
// (unversioned) binary state file format, which was later superseded by the
// version 1 format in JSON.
//
// Version 0 is no longer supported, so this is used only to detect it and
// return a nice error to the user.
func looksLikeVersion0(src []byte) bool {
	// Version 0 files begin with the magic prefix "tfstate".
	const magic = "tfstate"
	if len(src) < len(magic) {
		// Not even long enough to have the magic prefix
		return false
	}
	if string(src[0:len(magic)]) == magic {
		return true
	}
	return false
}
