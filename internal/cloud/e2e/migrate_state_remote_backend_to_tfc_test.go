//go:build e2e
// +build e2e

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
	t.Skip("TODO: see comments")
	_ = map[string]struct {
		operations  []operationSets
		validations func(t *testing.T, orgName string)
	}{
		"single workspace with backend name strategy, to cloud with name strategy": {},
		"single workspace with backend name strategy, to cloud with tags strategy": {},
	}
}

func Test_migrate_remote_backend_prefix_to_tfc_name(t *testing.T) {
	t.Skip("TODO: see comments")
	_ = map[string]struct {
		operations  []operationSets
		validations func(t *testing.T, orgName string)
	}{
		"single workspace with backend prefix strategy, to cloud with name strategy":    {},
		"multiple workspaces with backend prefix strategy, to cloud with name strategy": {},
	}
}

func Test_migrate_remote_backend_prefix_to_tfc_tags(t *testing.T) {
	t.Skip("TODO: see comments")
	_ = map[string]struct {
		operations  []operationSets
		validations func(t *testing.T, orgName string)
	}{
		"single workspace with backend prefix strategy, to cloud with tags strategy":    {},
		"multiple workspaces with backend prefix strategy, to cloud with tags strategy": {},
	}
}
