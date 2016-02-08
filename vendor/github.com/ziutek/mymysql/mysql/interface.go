// MySQL Client API written entirely in Go without any external dependences.
package mysql

import (
	"net"
	"time"
)

// ConCommon is a common interface to the connection.
// See mymysql/native for method documentation
type ConnCommon interface {
	Start(sql string, params ...interface{}) (Result, error)
	Prepare(sql string) (Stmt, error)

	Ping() error
	ThreadId() uint32
	Escape(txt string) string

	Query(sql string, params ...interface{}) ([]Row, Result, error)
	QueryFirst(sql string, params ...interface{}) (Row, Result, error)
	QueryLast(sql string, params ...interface{}) (Row, Result, error)
}

// Dialer can be used to dial connections to MySQL. If Dialer returns (nil, nil)
// the hook is skipped and normal dialing proceeds.
type Dialer func(proto, laddr, raddr string, timeout time.Duration) (net.Conn, error)

// Conn represnts connection to the MySQL server.
// See mymysql/native for method documentation
type Conn interface {
	ConnCommon

	Clone() Conn
	SetTimeout(time.Duration)
	Connect() error
	NetConn() net.Conn
	SetDialer(Dialer)
	Close() error
	IsConnected() bool
	Reconnect() error
	Use(dbname string) error
	Register(sql string)
	SetMaxPktSize(new_size int) int
	NarrowTypeSet(narrow bool)
	FullFieldInfo(full bool)
	Status() ConnStatus

	Begin() (Transaction, error)
}

// Transaction represents MySQL transaction
// See mymysql/native for method documentation
type Transaction interface {
	ConnCommon

	Commit() error
	Rollback() error
	Do(st Stmt) Stmt
	IsValid() bool
}

// Stmt represents MySQL prepared statement.
// See mymysql/native for method documentation
type Stmt interface {
	Bind(params ...interface{})
	Run(params ...interface{}) (Result, error)
	Delete() error
	Reset() error
	SendLongData(pnum int, data interface{}, pkt_size int) error

	Fields() []*Field
	NumParam() int
	WarnCount() int

	Exec(params ...interface{}) ([]Row, Result, error)
	ExecFirst(params ...interface{}) (Row, Result, error)
	ExecLast(params ...interface{}) (Row, Result, error)
}

// Result represents one MySQL result set.
// See mymysql/native for method documentation
type Result interface {
	StatusOnly() bool
	ScanRow(Row) error
	GetRow() (Row, error)

	MoreResults() bool
	NextResult() (Result, error)

	Fields() []*Field
	Map(string) int
	Message() string
	AffectedRows() uint64
	InsertId() uint64
	WarnCount() int

	MakeRow() Row
	GetRows() ([]Row, error)
	End() error
	GetFirstRow() (Row, error)
	GetLastRow() (Row, error)
}

// New can be used to establish a connection. It is set by imported engine
// (see mymysql/native, mymysql/thrsafe)
var New func(proto, laddr, raddr, user, passwd string, db ...string) Conn
