package acl

import "fmt"

// Table models the privileges of a table aclitem
type Table struct {
	ACL
}

// NewTable parses a PostgreSQL ACL string for a table and returns a Table
// object
func NewTable(acl ACL) (Table, error) {
	if !validRights(acl, validTablePrivs) {
		return Table{}, fmt.Errorf("invalid flags set for table (%+q), only %+q allowed", permString(acl.Privileges, acl.GrantOptions), validTablePrivs)
	}

	return Table{ACL: acl}, nil
}
