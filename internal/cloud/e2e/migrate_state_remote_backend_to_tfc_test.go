package main

import (
	"testing"
)

// REMOTE BACKEND
/*
	- RB name -> TFC name
	-- straight copy if only if different name, or same WS name in diff org
	-- other
	--  ensure that the local workspace, after migration, is the new name (in the tfc config block)
	- RB name -> TFC tags
	-- just add tag, if in same org
	-- If new org, if WS exists, just add tag
	-- If new org, if WS not exists, create and add tag
	- RB prefix -> TFC name
	-- create if not exists
	-- migrate the current worksapce state to ws name
	- RB prefix -> TFC tags
	-- update previous workspaces (prefix + local) with cloud config tag
	-- Rename the local workspaces to match the TFC workspaces (prefix + former local, ie app-prod). inform user

*/
func Test_migrate_remote_backend_name_to_tfc(t *testing.T) {
	t.Skip("todo: see comments")
}

func Test_migrate_remote_backend_prefix_to_tfc(t *testing.T) {
	t.Skip("todo: see comments")
}
