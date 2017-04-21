package acl

import "fmt"

// Type models the privileges of a type aclitem
type Type struct {
	ACL
}

// NewType parses an ACL object and returns a Type object.
func NewType(acl ACL) (Type, error) {
	if !validRights(acl, validTypePrivs) {
		return Type{}, fmt.Errorf("invalid flags set for type (%+q), only %+q allowed", permString(acl.Privileges, acl.GrantOptions), validTypePrivs)
	}

	return Type{ACL: acl}, nil
}
