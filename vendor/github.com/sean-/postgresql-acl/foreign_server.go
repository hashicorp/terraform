package acl

import "fmt"

// ForeignServer models the privileges of a foreign server aclitem
type ForeignServer struct {
	ACL
}

// NewForeignServer parses an ACL object and returns a ForeignServer object.
func NewForeignServer(acl ACL) (ForeignServer, error) {
	if !validRights(acl, validForeignServerPrivs) {
		return ForeignServer{}, fmt.Errorf("invalid flags set for foreign server (%+q), only %+q allowed", permString(acl.Privileges, acl.GrantOptions), validForeignServerPrivs)
	}

	return ForeignServer{ACL: acl}, nil
}
