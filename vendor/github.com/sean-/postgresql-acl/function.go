package acl

import "fmt"

// Function models the privileges of a function aclitem
type Function struct {
	ACL
}

// NewFunction parses an ACL object and returns a Function object.
func NewFunction(acl ACL) (Function, error) {
	if !validRights(acl, validFunctionPrivs) {
		return Function{}, fmt.Errorf("invalid flags set for function (%+q), only %+q allowed", permString(acl.Privileges, acl.GrantOptions), validFunctionPrivs)
	}

	return Function{ACL: acl}, nil
}
