# `postgresql-acl`

## `acl` Library

`acl` parses
[PostgreSQL's ACL syntax](https://www.postgresql.org/docs/current/static/sql-grant.html#SQL-GRANT-NOTES)
and returns a usable structure.  Library documentation is available at
[https://godoc.org/github.com/sean-/postgresql-acl](https://godoc.org/github.com/sean-/postgresql-acl).


```go
package main

import (
	"fmt"

	"github.com/sean-/postgresql-acl"
)

func structToString() acl.ACL {
	return acl.ACL{
		Role:         "foo",
		GrantedBy:    "bar",
		Privileges:   acl.Usage | acl.Create,
		GrantOptions: acl.Create,
	}
}

func stringToStruct() acl.Schema {
	// Parse an aclitem string
	aclitem, err := acl.Parse("foo=C*U/bar")
	if err != nil {
		panic(fmt.Sprintf("bad: %v", err))
	}

	// Verify that ACL permissions are appropriate for a schema type
	schema, err := acl.NewSchema(aclitem)
	if err != nil {
		panic(fmt.Sprintf("bad: %v", err))
	}

	return schema
}

func main() {
	fmt.Printf("ACL Struct to String: %+q\n", structToString().String())
	fmt.Printf("ACL String to Struct: %#v\n", stringToStruct().String())
}
```

```text
ACL Struct to String: "foo=UC*/bar"
ACL String to Struct: "foo=UC*/bar"
```

## Supported PostgreSQL `aclitem` Types

- column permissions
- database
- domain
- foreign data wrappers
- foreign server
- function
- language
- large object
- schema
- sequences
- table
- table space
- type

## Notes

The output from `String()` should match the ordering of characters in `aclitem`.

The target of each of these ACLs (e.g. schema name, table name, etc) is not
contained within PostgreSQLs `aclitem` and it is expected this value is managed
elsewhere in your object model.

Arrays of `aclitem` are supposed to be iterated over by the caller.  For
example:

```go
const schema = "public"
var name, owner string
var acls []string
err := conn.QueryRow("SELECT n.nspname, pg_catalog.pg_get_userbyid(n.nspowner), COALESCE(n.nspacl, '{}'::aclitem[])::TEXT[] FROM pg_catalog.pg_namespace n WHERE n.nspname = $1", schema).Scan(&name, &owner, pq.Array(&acls))
if err == nil {
    for _, acl := range acls {
        acl, err = pgacl.NewSchema(acl)
        if err != nil {
            return err
        }
        // ...
    }
}
```
