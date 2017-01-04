package acl

import "fmt"

// LargeObject models the privileges of a large object aclitem
type LargeObject struct {
	ACL
}

// NewLargeObject parses an ACL object and returns a LargeObject object.
func NewLargeObject(acl ACL) (LargeObject, error) {
	if !validRights(acl, validLargeObjectPrivs) {
		return LargeObject{}, fmt.Errorf("invalid flags set for large object (%+q), only %+q allowed", permString(acl.Privileges, acl.GrantOptions), validLargeObjectPrivs)
	}

	return LargeObject{ACL: acl}, nil
}
