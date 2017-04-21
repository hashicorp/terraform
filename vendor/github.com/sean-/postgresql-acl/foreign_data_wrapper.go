package acl

import "fmt"

// ForeignDataWrapper models the privileges of a domain aclitem
type ForeignDataWrapper struct {
	ACL
}

// NewForeignDataWrapper parses an ACL object and returns a ForeignDataWrapper object.
func NewForeignDataWrapper(acl ACL) (ForeignDataWrapper, error) {
	if !validRights(acl, validForeignDataWrapperPrivs) {
		return ForeignDataWrapper{}, fmt.Errorf("invalid flags set for domain (%+q), only %+q allowed", permString(acl.Privileges, acl.GrantOptions), validForeignDataWrapperPrivs)
	}

	return ForeignDataWrapper{ACL: acl}, nil
}
