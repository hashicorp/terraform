package acl

import "fmt"

// Domain models the privileges of a domain aclitem
type Domain struct {
	ACL
}

// NewDomain parses an ACL object and returns a Domain object.
func NewDomain(acl ACL) (Domain, error) {
	if !validRights(acl, validDomainPrivs) {
		return Domain{}, fmt.Errorf("invalid flags set for domain (%+q), only %+q allowed", permString(acl.Privileges, acl.GrantOptions), validDomainPrivs)
	}

	return Domain{ACL: acl}, nil
}
