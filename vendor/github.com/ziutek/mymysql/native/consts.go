package native

import "strconv"

// Client caps - borrowed from GoMySQL
const (
	_CLIENT_LONG_PASSWORD    = 1 << iota // new more secure passwords
	_CLIENT_FOUND_ROWS                   // Found instead of affected rows
	_CLIENT_LONG_FLAG                    // Get all column flags
	_CLIENT_CONNECT_WITH_DB              // One can specify db on connect
	_CLIENT_NO_SCHEMA                    // Don't allow database.table.column
	_CLIENT_COMPRESS                     // Can use compression protocol
	_CLIENT_ODBC                         // Odbc client
	_CLIENT_LOCAL_FILES                  // Can use LOAD DATA LOCAL
	_CLIENT_IGNORE_SPACE                 // Ignore spaces before '('
	_CLIENT_PROTOCOL_41                  // New 4.1 protocol
	_CLIENT_INTERACTIVE                  // This is an interactive client
	_CLIENT_SSL                          // Switch to SSL after handshake
	_CLIENT_IGNORE_SIGPIPE               // IGNORE sigpipes
	_CLIENT_TRANSACTIONS                 // Client knows about transactions
	_CLIENT_RESERVED                     // Old flag for 4.1 protocol
	_CLIENT_SECURE_CONN                  // New 4.1 authentication
	_CLIENT_MULTI_STATEMENTS             // Enable/disable multi-stmt support
	_CLIENT_MULTI_RESULTS                // Enable/disable multi-results
)

// Commands - borrowed from GoMySQL
const (
	_COM_QUIT                = 0x01
	_COM_INIT_DB             = 0x02
	_COM_QUERY               = 0x03
	_COM_FIELD_LIST          = 0x04
	_COM_CREATE_DB           = 0x05
	_COM_DROP_DB             = 0x06
	_COM_REFRESH             = 0x07
	_COM_SHUTDOWN            = 0x08
	_COM_STATISTICS          = 0x09
	_COM_PROCESS_INFO        = 0x0a
	_COM_CONNECT             = 0x0b
	_COM_PROCESS_KILL        = 0x0c
	_COM_DEBUG               = 0x0d
	_COM_PING                = 0x0e
	_COM_TIME                = 0x0f
	_COM_DELAYED_INSERT      = 0x10
	_COM_CHANGE_USER         = 0x11
	_COM_BINLOG_DUMP         = 0x12
	_COM_TABLE_DUMP          = 0x13
	_COM_CONNECT_OUT         = 0x14
	_COM_REGISTER_SLAVE      = 0x15
	_COM_STMT_PREPARE        = 0x16
	_COM_STMT_EXECUTE        = 0x17
	_COM_STMT_SEND_LONG_DATA = 0x18
	_COM_STMT_CLOSE          = 0x19
	_COM_STMT_RESET          = 0x1a
	_COM_SET_OPTION          = 0x1b
	_COM_STMT_FETCH          = 0x1c
)

// MySQL protocol types.
//
// mymysql uses only some of them for send data to the MySQL server. Used
// MySQL types are marked with a comment contains mymysql type that uses it.
const (
	MYSQL_TYPE_DECIMAL     = 0x00
	MYSQL_TYPE_TINY        = 0x01 // int8, uint8, bool
	MYSQL_TYPE_SHORT       = 0x02 // int16, uint16
	MYSQL_TYPE_LONG        = 0x03 // int32, uint32
	MYSQL_TYPE_FLOAT       = 0x04 // float32
	MYSQL_TYPE_DOUBLE      = 0x05 // float64
	MYSQL_TYPE_NULL        = 0x06 // nil
	MYSQL_TYPE_TIMESTAMP   = 0x07 // Timestamp
	MYSQL_TYPE_LONGLONG    = 0x08 // int64, uint64
	MYSQL_TYPE_INT24       = 0x09
	MYSQL_TYPE_DATE        = 0x0a // Date
	MYSQL_TYPE_TIME        = 0x0b // Time
	MYSQL_TYPE_DATETIME    = 0x0c // time.Time
	MYSQL_TYPE_YEAR        = 0x0d
	MYSQL_TYPE_NEWDATE     = 0x0e
	MYSQL_TYPE_VARCHAR     = 0x0f
	MYSQL_TYPE_BIT         = 0x10
	MYSQL_TYPE_NEWDECIMAL  = 0xf6
	MYSQL_TYPE_ENUM        = 0xf7
	MYSQL_TYPE_SET         = 0xf8
	MYSQL_TYPE_TINY_BLOB   = 0xf9
	MYSQL_TYPE_MEDIUM_BLOB = 0xfa
	MYSQL_TYPE_LONG_BLOB   = 0xfb
	MYSQL_TYPE_BLOB        = 0xfc // Blob
	MYSQL_TYPE_VAR_STRING  = 0xfd // []byte
	MYSQL_TYPE_STRING      = 0xfe // string
	MYSQL_TYPE_GEOMETRY    = 0xff

	MYSQL_UNSIGNED_MASK = uint16(1 << 15)
)

// Mapping of MySQL types to (prefered) protocol types. Use it if you create
// your own Raw value.
//
// Comments contains corresponding types used by mymysql. string type may be
// replaced by []byte type and vice versa. []byte type is native for sending
// on a network, so any string is converted to it before sending. Than for
// better preformance use []byte. 
const (
	// Client send and receive, mymysql representation for send / receive
	TINYINT   = MYSQL_TYPE_TINY      // int8 / int8
	SMALLINT  = MYSQL_TYPE_SHORT     // int16 / int16
	INT       = MYSQL_TYPE_LONG      // int32 / int32
	BIGINT    = MYSQL_TYPE_LONGLONG  // int64 / int64
	FLOAT     = MYSQL_TYPE_FLOAT     // float32 / float32
	DOUBLE    = MYSQL_TYPE_DOUBLE    // float64 / float32
	TIME      = MYSQL_TYPE_TIME      // Time / Time
	DATE      = MYSQL_TYPE_DATE      // Date / Date
	DATETIME  = MYSQL_TYPE_DATETIME  // time.Time / time.Time
	TIMESTAMP = MYSQL_TYPE_TIMESTAMP // Timestamp / time.Time
	CHAR      = MYSQL_TYPE_STRING    // string / []byte
	BLOB      = MYSQL_TYPE_BLOB      // Blob / []byte
	NULL      = MYSQL_TYPE_NULL      // nil

	// Client send only, mymysql representation for send
	OUT_TEXT      = MYSQL_TYPE_STRING // string
	OUT_VARCHAR   = MYSQL_TYPE_STRING // string
	OUT_BINARY    = MYSQL_TYPE_BLOB   // Blob
	OUT_VARBINARY = MYSQL_TYPE_BLOB   // Blob

	// Client receive only, mymysql representation for receive
	IN_MEDIUMINT  = MYSQL_TYPE_LONG        // int32
	IN_YEAR       = MYSQL_TYPE_SHORT       // int16
	IN_BINARY     = MYSQL_TYPE_STRING      // []byte
	IN_VARCHAR    = MYSQL_TYPE_VAR_STRING  // []byte
	IN_VARBINARY  = MYSQL_TYPE_VAR_STRING  // []byte
	IN_TINYBLOB   = MYSQL_TYPE_TINY_BLOB   // []byte
	IN_TINYTEXT   = MYSQL_TYPE_TINY_BLOB   // []byte
	IN_TEXT       = MYSQL_TYPE_BLOB        // []byte
	IN_MEDIUMBLOB = MYSQL_TYPE_MEDIUM_BLOB // []byte
	IN_MEDIUMTEXT = MYSQL_TYPE_MEDIUM_BLOB // []byte
	IN_LONGBLOB   = MYSQL_TYPE_LONG_BLOB   // []byte
	IN_LONGTEXT   = MYSQL_TYPE_LONG_BLOB   // []byte

	// MySQL 5.x specific
	IN_DECIMAL = MYSQL_TYPE_NEWDECIMAL // TODO
	IN_BIT     = MYSQL_TYPE_BIT        // []byte
)

// Flags - borrowed from GoMySQL
const (
	_FLAG_NOT_NULL = 1 << iota
	_FLAG_PRI_KEY
	_FLAG_UNIQUE_KEY
	_FLAG_MULTIPLE_KEY
	_FLAG_BLOB
	_FLAG_UNSIGNED
	_FLAG_ZEROFILL
	_FLAG_BINARY
	_FLAG_ENUM
	_FLAG_AUTO_INCREMENT
	_FLAG_TIMESTAMP
	_FLAG_SET
	_FLAG_NO_DEFAULT_VALUE
)

var (
	_SIZE_OF_INT int
	_INT_TYPE    uint16
)

func init() {
	switch strconv.IntSize {
	case 32:
		_INT_TYPE = MYSQL_TYPE_LONG
		_SIZE_OF_INT = 4
	case 64:
		_INT_TYPE = MYSQL_TYPE_LONGLONG
		_SIZE_OF_INT = 8
	default:
		panic("bad int size")
	}
}
