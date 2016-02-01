package thrsafe

import (
	"github.com/ziutek/mymysql/mysql"
	"github.com/ziutek/mymysql/native"
	"testing"
)

const (
	user   = "testuser"
	passwd = "TestPasswd9"
	dbname = "test"
	proto  = "tcp"
	daddr  = "127.0.0.1:3306"
	//proto = "unix"
	//daddr = "/var/run/mysqld/mysqld.sock"
	debug = false
)

var db mysql.Conn

func checkErr(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
}

func connect(t *testing.T) mysql.Conn {
	db := New(proto, "", daddr, user, passwd, dbname)
	db.(*Conn).Conn.(*native.Conn).Debug = debug
	checkErr(t, db.Connect())
	return db
}

func TestS(t *testing.T) {
	db := connect(t)
	res, err := db.Start("SET @a=1")
	checkErr(t, err)
	if !res.StatusOnly() {
		t.Fatalf("'SET @a' statement returns result with rows")
	}
	err = db.Close()
	checkErr(t, err)
}

func TestSS(t *testing.T) {
	db := connect(t)

	res, err := db.Start("SET @a=1; SET @b=2")
	checkErr(t, err)
	if !res.StatusOnly() {
		t.Fatalf("'SET @a' statement returns result with rows")
	}

	res, err = res.NextResult()
	checkErr(t, err)
	if !res.StatusOnly() {
		t.Fatalf("'SET @b' statement returns result with rows")
	}

	err = db.Close()
	checkErr(t, err)
}

func TestSDS(t *testing.T) {
	db := connect(t)

	res, err := db.Start("SET @a=1; SELECT @a; SET @b=2")
	checkErr(t, err)
	if !res.StatusOnly() {
		t.Fatalf("'SET @a' statement returns result with rows")
	}

	res, err = res.NextResult()
	checkErr(t, err)
	rows, err := res.GetRows()
	checkErr(t, err)
	if rows[0].Int(0) != 1 {
		t.Fatalf("First query doesn't return '1'")
	}

	res, err = res.NextResult()
	checkErr(t, err)
	if !res.StatusOnly() {
		t.Fatalf("'SET @b' statement returns result with rows")
	}

	err = db.Close()
	checkErr(t, err)
}

func TestSSDDD(t *testing.T) {
	db := connect(t)

	res, err := db.Start("SET @a=1; SET @b=2; SELECT @a; SELECT @b; SELECT 3")
	checkErr(t, err)
	if !res.StatusOnly() {
		t.Fatalf("'SET @a' statement returns result with rows")
	}

	res, err = res.NextResult()
	checkErr(t, err)
	if !res.StatusOnly() {
		t.Fatalf("'SET @b' statement returns result with rows")
	}

	res, err = res.NextResult()
	checkErr(t, err)
	rows, err := res.GetRows()
	checkErr(t, err)
	if rows[0].Int(0) != 1 {
		t.Fatalf("First query doesn't return '1'")
	}

	res, err = res.NextResult()
	checkErr(t, err)
	rows, err = res.GetRows()
	checkErr(t, err)
	if rows[0].Int(0) != 2 {
		t.Fatalf("Second query doesn't return '2'")
	}

	res, err = res.NextResult()
	checkErr(t, err)
	rows, err = res.GetRows()
	checkErr(t, err)
	if rows[0].Int(0) != 3 {
		t.Fatalf("Thrid query doesn't return '3'")
	}
	if res.MoreResults() {
		t.Fatalf("There is unexpected one more result")
	}

	err = db.Close()
	checkErr(t, err)
}
