package acl

// Privileges represents a PostgreSQL ACL bitmask
type Privileges uint16

// See postgresql/src/include/utils/acl.h for inspiration.  Like PostgreSQL,
// "rights" refer to the combined grant option and privilege bits fields.
const (
	NoPrivs Privileges = 0

	// Ordering taken from postgresql/src/include/nodes/parsenodes.h
	Insert Privileges = 1 << iota
	Select
	Update
	Delete
	Truncate
	References
	Trigger
	Execute
	Usage
	Create
	Temporary
	Connect

	numPrivileges
)

const (
	validColumnPrivs             = Insert | Select | Update | References
	validDatabasePrivs           = Create | Temporary | Connect
	validDomainPrivs             = Usage
	validForeignDataWrapperPrivs = Usage
	validForeignServerPrivs      = Usage
	validFunctionPrivs           = Execute
	validLanguagePrivs           = Usage
	validLargeObjectPrivs        = Select | Update
	validSchemaPrivs             = Usage | Create
	validSequencePrivs           = Usage | Select | Update
	validTablePrivs              = Insert | Select | Update | Delete | Truncate | References | Trigger
	validTablespacePrivs         = Create
	validTypePrivs               = Usage
)
