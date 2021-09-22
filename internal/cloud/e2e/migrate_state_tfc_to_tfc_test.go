package main

import (
	"testing"
)

/*

	If org to org, treat it like a new backend. Then go through the multi/single logic

	If same org, but name/tag changes
		config name -> config name
		-- straight copy
		config name -> config tags
		-- jsut add tag to workspace.
		config tags -> config name
		-- straight copy
*/
func Test_migrate_tfc_to_tfc(t *testing.T) {
	t.Skip("todo: see comments")
}
