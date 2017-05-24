package acl

import "fmt"

// Database models the privileges of a database aclitem
type Database struct {
	ACL
}

// NewDatabase parses an ACL object and returns a Database object.
func NewDatabase(acl ACL) (Database, error) {
	if !validRights(acl, validDatabasePrivs) {
		return Database{}, fmt.Errorf("invalid flags set for database (%+q), only %+q allowed", permString(acl.Privileges, acl.GrantOptions), validDatabasePrivs)
	}

	return Database{ACL: acl}, nil
}
