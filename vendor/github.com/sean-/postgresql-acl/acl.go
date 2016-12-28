package acl

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// ACL represents a single PostgreSQL `aclitem` entry.
type ACL struct {
	Privileges   Privileges
	GrantOptions Privileges
	Role         string
	GrantedBy    string
}

// GetGrantOption returns true if the acl has the grant option set for the
// specified priviledge.
func (a ACL) GetGrantOption(priv Privileges) bool {
	if a.GrantOptions&priv != 0 {
		return true
	}
	return false
}

// GetPriviledge returns true if the acl has the specified priviledge set.
func (a ACL) GetPrivilege(priv Privileges) bool {
	if a.Privileges&priv != 0 {
		return true
	}
	return false
}

// Parse parses a PostgreSQL aclitem string and returns an ACL
func Parse(aclStr string) (ACL, error) {
	acl := ACL{}
	idx := strings.IndexByte(aclStr, '=')
	if idx == -1 {
		return ACL{}, fmt.Errorf("invalid aclStr format: %+q", aclStr)
	}

	acl.Role = aclStr[:idx]

	aclLen := len(aclStr)
	var i int
	withGrant := func() bool {
		if i+1 >= aclLen {
			return false
		}

		if aclStr[i+1] == '*' {
			i++
			return true
		}

		return false
	}

SCAN:
	for i = idx + 1; i < aclLen; i++ {
		switch aclStr[i] {
		case 'w':
			acl.Privileges |= Update
			if withGrant() {
				acl.GrantOptions |= Update
			}
		case 'r':
			acl.Privileges |= Select
			if withGrant() {
				acl.GrantOptions |= Select
			}
		case 'a':
			acl.Privileges |= Insert
			if withGrant() {
				acl.GrantOptions |= Insert
			}
		case 'd':
			acl.Privileges |= Delete
			if withGrant() {
				acl.GrantOptions |= Delete
			}
		case 'D':
			acl.Privileges |= Truncate
			if withGrant() {
				acl.GrantOptions |= Truncate
			}
		case 'x':
			acl.Privileges |= References
			if withGrant() {
				acl.GrantOptions |= References
			}
		case 't':
			acl.Privileges |= Trigger
			if withGrant() {
				acl.GrantOptions |= Trigger
			}
		case 'X':
			acl.Privileges |= Execute
			if withGrant() {
				acl.GrantOptions |= Execute
			}
		case 'U':
			acl.Privileges |= Usage
			if withGrant() {
				acl.GrantOptions |= Usage
			}
		case 'C':
			acl.Privileges |= Create
			if withGrant() {
				acl.GrantOptions |= Create
			}
		case 'T':
			acl.Privileges |= Temporary
			if withGrant() {
				acl.GrantOptions |= Temporary
			}
		case 'c':
			acl.Privileges |= Connect
			if withGrant() {
				acl.GrantOptions |= Connect
			}
		case '/':
			if i+1 <= aclLen {
				acl.GrantedBy = aclStr[i+1:]
			}
			break SCAN
		default:
			return ACL{}, fmt.Errorf("invalid byte %c in aclitem at %d: %+q", aclStr[i], i, aclStr)
		}
	}

	return acl, nil
}

// String produces a PostgreSQL aclitem-compatible string
func (a ACL) String() string {
	b := new(bytes.Buffer)
	bitMaskStr := permString(a.Privileges, a.GrantOptions)
	role := a.Role
	grantedBy := a.GrantedBy

	b.Grow(len(role) + len("=") + len(bitMaskStr) + len("/") + len(grantedBy))

	fmt.Fprint(b, role, "=", bitMaskStr)

	if grantedBy != "" {
		fmt.Fprint(b, "/", grantedBy)
	}

	return b.String()
}

// permString is a small helper function that emits the permission bitmask as a
// string.
func permString(perms, grantOptions Privileges) string {
	b := new(bytes.Buffer)
	b.Grow(int(numPrivileges) * 2)

	// From postgresql/src/include/utils/acl.h:
	//
	// /* string holding all privilege code chars, in order by bitmask position */
	// #define ACL_ALL_RIGHTS_STR "arwdDxtXUCTc"
	if perms&Insert != 0 {
		fmt.Fprint(b, "a")
		if grantOptions&Insert != 0 {
			fmt.Fprint(b, "*")
		}
	}

	if perms&Select != 0 {
		fmt.Fprint(b, "r")
		if grantOptions&Select != 0 {
			fmt.Fprint(b, "*")
		}
	}

	if perms&Update != 0 {
		fmt.Fprint(b, "w")
		if grantOptions&Update != 0 {
			fmt.Fprint(b, "*")
		}
	}

	if perms&Delete != 0 {
		fmt.Fprint(b, "d")
		if grantOptions&Delete != 0 {
			fmt.Fprint(b, "*")
		}
	}

	if perms&Truncate != 0 {
		fmt.Fprint(b, "D")
		if grantOptions&Truncate != 0 {
			fmt.Fprint(b, "*")
		}
	}

	if perms&References != 0 {
		fmt.Fprint(b, "x")
		if grantOptions&References != 0 {
			fmt.Fprint(b, "*")
		}
	}

	if perms&Trigger != 0 {
		fmt.Fprint(b, "t")
		if grantOptions&Trigger != 0 {
			fmt.Fprint(b, "*")
		}
	}

	if perms&Execute != 0 {
		fmt.Fprint(b, "X")
		if grantOptions&Execute != 0 {
			fmt.Fprint(b, "*")
		}
	}

	if perms&Usage != 0 {
		fmt.Fprint(b, "U")
		if grantOptions&Usage != 0 {
			fmt.Fprint(b, "*")
		}
	}

	if perms&Create != 0 {
		fmt.Fprint(b, "C")
		if grantOptions&Create != 0 {
			fmt.Fprint(b, "*")
		}
	}

	if perms&Temporary != 0 {
		fmt.Fprint(b, "T")
		if grantOptions&Temporary != 0 {
			fmt.Fprint(b, "*")
		}
	}

	if perms&Connect != 0 {
		fmt.Fprint(b, "c")
		if grantOptions&Connect != 0 {
			fmt.Fprint(b, "*")
		}
	}

	return b.String()
}

// quoteRole is a small helper function that handles the quoting of a role name,
// or PUBLIC, if no role is specified.
func quoteRole(role string) string {
	if role == "" {
		return "PUBLIC"
	}

	return pq.QuoteIdentifier(role)
}

// validRights checks to make sure a given acl's permissions and grant options
// don't exceed the specified mask valid privileges.
func validRights(acl ACL, validPrivs Privileges) bool {
	if (acl.Privileges|validPrivs) == validPrivs &&
		(acl.GrantOptions|validPrivs) == validPrivs {
		return true
	}
	return false
}
