package acl

import (
	"bytes"
	"fmt"

	"github.com/lib/pq"
)

// Schema models the privileges of a schema aclitem
type Schema struct {
	ACL
}

// NewSchema parses an ACL object and returns a Schema object.
func NewSchema(acl ACL) (Schema, error) {
	if !validRights(acl, validSchemaPrivs) {
		return Schema{}, fmt.Errorf("invalid flags set for schema (%+q), only %+q allowed", permString(acl.Privileges, acl.GrantOptions), validSchemaPrivs)
	}

	return Schema{ACL: acl}, nil
}

// Merge adds the argument's attributes to the receiver for values that are
// composable or not set and returns a new Schema object with the resulting
// values.  Be careful with the role "" which is implicitly interpreted as the
// PUBLIC role.
func (s Schema) Merge(x Schema) Schema {
	role := s.Role
	if role == "" {
		role = x.Role
	}

	grantedBy := s.GrantedBy
	if grantedBy == "" {
		grantedBy = x.GrantedBy
	}

	return Schema{
		ACL{
			Privileges:   s.Privileges | x.Privileges,
			GrantOptions: s.GrantOptions | x.GrantOptions,
			Role:         role,
			GrantedBy:    grantedBy,
		},
	}
}

// Grants returns a list of SQL queries that constitute the privileges specified
// in the receiver for the target schema.
func (s Schema) Grants(target string) []string {
	const maxQueries = 2
	queries := make([]string, 0, maxQueries)

	if s.GetPrivilege(Create) {
		b := bytes.NewBufferString("GRANT CREATE ON SCHEMA ")
		fmt.Fprint(b, pq.QuoteIdentifier(target), " TO ", quoteRole(s.Role))

		if s.GetGrantOption(Create) {
			fmt.Fprint(b, " WITH GRANT OPTION")
		}

		queries = append(queries, b.String())
	}

	if s.GetPrivilege(Usage) {
		b := bytes.NewBufferString("GRANT USAGE ON SCHEMA ")
		fmt.Fprint(b, pq.QuoteIdentifier(target), " TO ", quoteRole(s.Role))

		if s.GetGrantOption(Usage) {
			fmt.Fprint(b, " WITH GRANT OPTION")
		}

		queries = append(queries, b.String())
	}

	return queries
}

// Revokes returns a list of SQL queries that remove the privileges specified
// in the receiver from the target schema.
func (s Schema) Revokes(target string) []string {
	const maxQueries = 2
	queries := make([]string, 0, maxQueries)

	if s.GetPrivilege(Create) {
		b := bytes.NewBufferString("REVOKE")
		if s.GetGrantOption(Create) {
			fmt.Fprint(b, " GRANT OPTION FOR")
		}

		fmt.Fprint(b, " CREATE ON SCHEMA ")
		fmt.Fprint(b, pq.QuoteIdentifier(target), " FROM ", quoteRole(s.Role))
		queries = append(queries, b.String())
	}

	if s.GetPrivilege(Usage) {
		b := bytes.NewBufferString("REVOKE")
		if s.GetGrantOption(Usage) {
			fmt.Fprint(b, " GRANT OPTION FOR")
		}

		fmt.Fprint(b, " USAGE ON SCHEMA ")
		fmt.Fprint(b, pq.QuoteIdentifier(target), " FROM ", quoteRole(s.Role))
		queries = append(queries, b.String())
	}

	return queries
}
