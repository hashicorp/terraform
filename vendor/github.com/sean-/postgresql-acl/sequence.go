package acl

import "fmt"

// Sequence models the privileges of a sequence aclitem
type Sequence struct {
	ACL
}

// NewSequence parses a PostgreSQL ACL string for a sequence and returns a Sequence
// object
func NewSequence(acl ACL) (Sequence, error) {
	if !validRights(acl, validSequencePrivs) {
		return Sequence{}, fmt.Errorf("invalid flags set for sequence (%+q), only %+q allowed", permString(acl.Privileges, acl.GrantOptions), validSequencePrivs)
	}

	return Sequence{ACL: acl}, nil
}
