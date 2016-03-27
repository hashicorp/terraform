// Thread safe engine for MyMySQL
//
// In contrast to native engine:
// - one connection can be used by multiple gorutines,
// - if connection is idle pings are sent to the server (once per minute) to
//   avoid timeout.
//
// See documentation of mymysql/native for details
package thrsafe

import (
	"github.com/ziutek/mymysql/mysql"
	_ "github.com/ziutek/mymysql/native"
	"io"
	"sync"
	"time"
)

type Conn struct {
	mysql.Conn
	mutex *sync.Mutex

	stopPinger chan struct{}
	lastUsed   time.Time
}

func (c *Conn) lock() {
	//log.Println(c, ":: lock @", c.mutex)
	c.mutex.Lock()
}

func (c *Conn) unlock() {
	//log.Println(c, ":: unlock @", c.mutex)
	c.lastUsed = time.Now()
	c.mutex.Unlock()
}

type Result struct {
	mysql.Result
	conn *Conn
}

type Stmt struct {
	mysql.Stmt
	conn *Conn
}

type Transaction struct {
	*Conn
	conn *Conn
}

func New(proto, laddr, raddr, user, passwd string, db ...string) mysql.Conn {
	return &Conn{
		Conn:  orgNew(proto, laddr, raddr, user, passwd, db...),
		mutex: new(sync.Mutex),
	}
}

func (c *Conn) Clone() mysql.Conn {
	return &Conn{
		Conn:  c.Conn.Clone(),
		mutex: new(sync.Mutex),
	}
}

func (c *Conn) pinger() {
	const to = 60 * time.Second
	sleep := to
	for {
		timer := time.After(sleep)
		select {
		case <-c.stopPinger:
			return
		case t := <-timer:
			c.mutex.Lock()
			lastUsed := c.lastUsed
			c.mutex.Unlock()
			sleep := to - t.Sub(lastUsed)
			if sleep <= 0 {
				if c.Ping() != nil {
					return
				}
				sleep = to
			}
		}
	}
}

func (c *Conn) Connect() error {
	//log.Println("Connect")
	c.lock()
	defer c.unlock()
	c.stopPinger = make(chan struct{})
	go c.pinger()
	return c.Conn.Connect()
}

func (c *Conn) Close() error {
	//log.Println("Close")
	close(c.stopPinger) // Stop pinger before lock connection
	c.lock()
	defer c.unlock()
	return c.Conn.Close()
}

func (c *Conn) Reconnect() error {
	//log.Println("Reconnect")
	c.lock()
	defer c.unlock()
	if c.stopPinger == nil {
		go c.pinger()
	}
	return c.Conn.Reconnect()
}

func (c *Conn) Use(dbname string) error {
	//log.Println("Use")
	c.lock()
	defer c.unlock()
	return c.Conn.Use(dbname)
}

func (c *Conn) Start(sql string, params ...interface{}) (mysql.Result, error) {
	//log.Println("Start")
	c.lock()
	res, err := c.Conn.Start(sql, params...)
	// Unlock if error or OK result (which doesn't provide any fields)
	if err != nil {
		c.unlock()
		return nil, err
	}
	if res.StatusOnly() && !res.MoreResults() {
		c.unlock()
	}
	return &Result{Result: res, conn: c}, err
}

func (c *Conn) Status() mysql.ConnStatus {
	c.lock()
	defer c.unlock()
	return c.Conn.Status()
}

func (c *Conn) Escape(txt string) string {
	return mysql.Escape(c, txt)
}

func (res *Result) ScanRow(row mysql.Row) error {
	//log.Println("ScanRow")
	err := res.Result.ScanRow(row)
	if err == nil {
		// There are more rows to read
		return nil
	}
	if err == mysql.ErrReadAfterEOR {
		// Trying read after EOR - connection unlocked before
		return err
	}
	if err != io.EOF || !res.StatusOnly() && !res.MoreResults() {
		// Error or no more rows in not empty result set and no more resutls.
		// In case if empty result set and no more resutls Start has unlocked
		// it before.
		res.conn.unlock()
	}
	return err
}

func (res *Result) GetRow() (mysql.Row, error) {
	return mysql.GetRow(res)
}

func (res *Result) NextResult() (mysql.Result, error) {
	//log.Println("NextResult")
	next, err := res.Result.NextResult()
	if err != nil {
		return nil, err
	}
	if next == nil {
		return nil, nil
	}
	if next.StatusOnly() && !next.MoreResults() {
		res.conn.unlock()
	}
	return &Result{next, res.conn}, nil
}

func (c *Conn) Ping() error {
	c.lock()
	defer c.unlock()
	return c.Conn.Ping()
}

func (c *Conn) Prepare(sql string) (mysql.Stmt, error) {
	//log.Println("Prepare")
	c.lock()
	defer c.unlock()
	stmt, err := c.Conn.Prepare(sql)
	if err != nil {
		return nil, err
	}
	return &Stmt{Stmt: stmt, conn: c}, nil
}

func (stmt *Stmt) Run(params ...interface{}) (mysql.Result, error) {
	//log.Println("Run")
	stmt.conn.lock()
	res, err := stmt.Stmt.Run(params...)
	// Unlock if error or OK result (which doesn't provide any fields)
	if err != nil {
		stmt.conn.unlock()
		return nil, err
	}
	if res.StatusOnly() && !res.MoreResults() {
		stmt.conn.unlock()
	}
	return &Result{Result: res, conn: stmt.conn}, nil
}

func (stmt *Stmt) Delete() error {
	//log.Println("Delete")
	stmt.conn.lock()
	defer stmt.conn.unlock()
	return stmt.Stmt.Delete()
}

func (stmt *Stmt) Reset() error {
	//log.Println("Reset")
	stmt.conn.lock()
	defer stmt.conn.unlock()
	return stmt.Stmt.Reset()
}

func (stmt *Stmt) SendLongData(pnum int, data interface{}, pkt_size int) error {
	//log.Println("SendLongData")
	stmt.conn.lock()
	defer stmt.conn.unlock()
	return stmt.Stmt.SendLongData(pnum, data, pkt_size)
}

// See mysql.Query
func (c *Conn) Query(sql string, params ...interface{}) ([]mysql.Row, mysql.Result, error) {
	return mysql.Query(c, sql, params...)
}

// See mysql.QueryFirst
func (my *Conn) QueryFirst(sql string, params ...interface{}) (mysql.Row, mysql.Result, error) {
	return mysql.QueryFirst(my, sql, params...)
}

// See mysql.QueryLast
func (my *Conn) QueryLast(sql string, params ...interface{}) (mysql.Row, mysql.Result, error) {
	return mysql.QueryLast(my, sql, params...)
}

// See mysql.Exec
func (stmt *Stmt) Exec(params ...interface{}) ([]mysql.Row, mysql.Result, error) {
	return mysql.Exec(stmt, params...)
}

// See mysql.ExecFirst
func (stmt *Stmt) ExecFirst(params ...interface{}) (mysql.Row, mysql.Result, error) {
	return mysql.ExecFirst(stmt, params...)
}

// See mysql.ExecLast
func (stmt *Stmt) ExecLast(params ...interface{}) (mysql.Row, mysql.Result, error) {
	return mysql.ExecLast(stmt, params...)
}

// See mysql.End
func (res *Result) End() error {
	return mysql.End(res)
}

// See mysql.GetFirstRow
func (res *Result) GetFirstRow() (mysql.Row, error) {
	return mysql.GetFirstRow(res)
}

// See mysql.GetLastRow
func (res *Result) GetLastRow() (mysql.Row, error) {
	return mysql.GetLastRow(res)
}

// See mysql.GetRows
func (res *Result) GetRows() ([]mysql.Row, error) {
	return mysql.GetRows(res)
}

// Begins a new transaction. No any other thread can send command on this
// connection until Commit or Rollback will be called.
// Periodical pinging the server is disabled during transaction.

func (c *Conn) Begin() (mysql.Transaction, error) {
	//log.Println("Begin")
	c.lock()
	tr := Transaction{
		&Conn{Conn: c.Conn, mutex: new(sync.Mutex)},
		c,
	}
	_, err := c.Conn.Start("START TRANSACTION")
	if err != nil {
		c.unlock()
		return nil, err
	}
	return &tr, nil
}

func (tr *Transaction) end(cr string) error {
	tr.lock()
	_, err := tr.conn.Conn.Start(cr)
	tr.conn.unlock()
	// Invalidate this transaction
	m := tr.Conn.mutex
	tr.Conn = nil
	tr.conn = nil
	m.Unlock() // One goorutine which still uses this transaction will panic
	return err
}

func (tr *Transaction) Commit() error {
	//log.Println("Commit")
	return tr.end("COMMIT")
}

func (tr *Transaction) Rollback() error {
	//log.Println("Rollback")
	return tr.end("ROLLBACK")
}

func (tr *Transaction) IsValid() bool {
	return tr.Conn != nil
}

func (tr *Transaction) Do(st mysql.Stmt) mysql.Stmt {
	if s, ok := st.(*Stmt); ok && s.conn == tr.conn {
		// Returns new statement which uses statement mutexes
		return &Stmt{s.Stmt, tr.Conn}
	}
	panic("Transaction and statement doesn't belong to the same connection")
}

var orgNew func(proto, laddr, raddr, user, passwd string, db ...string) mysql.Conn

func init() {
	orgNew = mysql.New
	mysql.New = New
}
