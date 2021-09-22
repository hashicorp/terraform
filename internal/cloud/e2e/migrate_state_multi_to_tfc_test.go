package main

import (
	"testing"
)

/*
	"multi" == multi-backend, multiple workspaces
		-- when cloud config == name ->
		---- prompt -> do you want to ONLY migrate the current workspace

		-- when cloud config == tags
		-- If Default present, prompt to rename default.
		-- Then -> Prompt with *
*/
func Test_migrate_multi_to_tfc(t *testing.T) {
	t.Skip("todo: see comments")
}
