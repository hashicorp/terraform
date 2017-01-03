package acl

import "fmt"

// Column models the privileges of a column aclitem
type Column struct {
	ACL
}

// NewColumn parses an ACL object and returns a Column object.
func NewColumn(acl ACL) (Column, error) {
	if !validRights(acl, validColumnPrivs) {
		return Column{}, fmt.Errorf("invalid flags set for column (%+q), only %+q allowed", permString(acl.Privileges, acl.GrantOptions), validColumnPrivs)
	}

	return Column{ACL: acl}, nil
}
