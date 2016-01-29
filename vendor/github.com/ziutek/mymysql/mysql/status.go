package mysql

type ConnStatus uint16

// Status of server connection
const (
	SERVER_STATUS_IN_TRANS          ConnStatus = 0x01 // Transaction has started
	SERVER_STATUS_AUTOCOMMIT        ConnStatus = 0x02 // Server in auto_commit mode
	SERVER_STATUS_MORE_RESULTS      ConnStatus = 0x04
	SERVER_MORE_RESULTS_EXISTS      ConnStatus = 0x08 // Multi query - next query exists
	SERVER_QUERY_NO_GOOD_INDEX_USED ConnStatus = 0x10
	SERVER_QUERY_NO_INDEX_USED      ConnStatus = 0x20
	SERVER_STATUS_CURSOR_EXISTS     ConnStatus = 0x40 // Server opened a read-only non-scrollable cursor for a query
	SERVER_STATUS_LAST_ROW_SENT     ConnStatus = 0x80

	SERVER_STATUS_DB_DROPPED           ConnStatus = 0x100
	SERVER_STATUS_NO_BACKSLASH_ESCAPES ConnStatus = 0x200
)
