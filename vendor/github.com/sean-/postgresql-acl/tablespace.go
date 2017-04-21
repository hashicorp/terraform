package acl

import "fmt"

// Tablespace models the privileges of a tablespace aclitem
type Tablespace struct {
	ACL
}

// NewTablespace parses an ACL object and returns a Tablespace object.
func NewTablespace(acl ACL) (Tablespace, error) {
	if !validRights(acl, validTablespacePrivs) {
		return Tablespace{}, fmt.Errorf("invalid flags set for tablespace (%+q), only %+q allowed", permString(acl.Privileges, acl.GrantOptions), validTablespacePrivs)
	}

	return Tablespace{ACL: acl}, nil
}
