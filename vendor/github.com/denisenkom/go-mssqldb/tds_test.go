package mssql

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"
)

type MockTransport struct {
	bytes.Buffer
}

func (t *MockTransport) Close() error {
	return nil
}

func TestSendLogin(t *testing.T) {
	buf := newTdsBuffer(1024, new(MockTransport))
	login := login{
		TDSVersion:     verTDS73,
		PacketSize:     0x1000,
		ClientProgVer:  0x01060100,
		ClientPID:      100,
		ClientTimeZone: -4 * 60,
		ClientID:       [6]byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab},
		OptionFlags1:   0xe0,
		OptionFlags3:   8,
		HostName:       "subdev1",
		UserName:       "test",
		Password:       "testpwd",
		AppName:        "appname",
		ServerName:     "servername",
		CtlIntName:     "library",
		Language:       "en",
		Database:       "database",
		ClientLCID:     0x204,
		AtchDBFile:     "filepath",
	}
	err := sendLogin(buf, login)
	if err != nil {
		t.Error("sendLogin should succeed")
	}
	ref := []byte{
		16, 1, 0, 222, 0, 0, 1, 0, 198 + 16, 0, 0, 0, 3, 0, 10, 115, 0, 16, 0, 0, 0, 1,
		6, 1, 100, 0, 0, 0, 0, 0, 0, 0, 224, 0, 0, 8, 16, 255, 255, 255, 4, 2, 0,
		0, 94, 0, 7, 0, 108, 0, 4, 0, 116, 0, 7, 0, 130, 0, 7, 0, 144, 0, 10, 0, 0,
		0, 0, 0, 164, 0, 7, 0, 178, 0, 2, 0, 182, 0, 8, 0, 18, 52, 86, 120, 144, 171,
		198, 0, 0, 0, 198, 0, 8, 0, 214, 0, 0, 0, 0, 0, 0, 0, 115, 0, 117, 0, 98,
		0, 100, 0, 101, 0, 118, 0, 49, 0, 116, 0, 101, 0, 115, 0, 116, 0, 226, 165,
		243, 165, 146, 165, 226, 165, 162, 165, 210, 165, 227, 165, 97, 0, 112,
		0, 112, 0, 110, 0, 97, 0, 109, 0, 101, 0, 115, 0, 101, 0, 114, 0, 118, 0,
		101, 0, 114, 0, 110, 0, 97, 0, 109, 0, 101, 0, 108, 0, 105, 0, 98, 0, 114,
		0, 97, 0, 114, 0, 121, 0, 101, 0, 110, 0, 100, 0, 97, 0, 116, 0, 97, 0, 98,
		0, 97, 0, 115, 0, 101, 0, 102, 0, 105, 0, 108, 0, 101, 0, 112, 0, 97, 0,
		116, 0, 104, 0}
	out := buf.buf[:buf.pos]
	if !bytes.Equal(ref, out) {
		t.Error("input output don't match")
		fmt.Print(hex.Dump(ref))
		fmt.Print(hex.Dump(out))
	}
}

func TestSendSqlBatch(t *testing.T) {
	addr := os.Getenv("HOST")
	instance := os.Getenv("INSTANCE")

	conn, err := connect(map[string]string{
		"server":   fmt.Sprintf("%s\\%s", addr, instance),
		"user id":  os.Getenv("SQLUSER"),
		"password": os.Getenv("SQLPASSWORD"),
		"database": os.Getenv("DATABASE"),
	})
	if err != nil {
		t.Error("Open connection failed:", err.Error())
		return
	}
	defer conn.buf.transport.Close()

	headers := []headerStruct{
		{hdrtype: dataStmHdrTransDescr,
			data: transDescrHdr{0, 1}.pack()},
	}
	err = sendSqlBatch72(conn.buf, "select 1", headers)
	if err != nil {
		t.Error("Sending sql batch failed", err.Error())
		return
	}

	ch := make(chan tokenStruct, 5)
	go processResponse(conn, ch)

	var lastRow []interface{}
loop:
	for tok := range ch {
		switch token := tok.(type) {
		case doneStruct:
			break loop
		case []columnStruct:
			conn.columns = token
		case []interface{}:
			lastRow = token
		default:
			fmt.Println("unknown token", tok)
		}
	}

	switch value := lastRow[0].(type) {
	case int32:
		if value != 1 {
			t.Error("Invalid value returned, should be 1", value)
			return
		}
	}
}

func makeConnStr() string {
	addr := os.Getenv("HOST")
	instance := os.Getenv("INSTANCE")
	user := os.Getenv("SQLUSER")
	password := os.Getenv("SQLPASSWORD")
	database := os.Getenv("DATABASE")
	return fmt.Sprintf(
		"Server=%s\\%s;User Id=%s;Password=%s;Database=%s;log=63",
		addr, instance, user, password, database)
}

func open(t *testing.T) *sql.DB {
	conn, err := sql.Open("mssql", makeConnStr())
	if err != nil {
		t.Error("Open connection failed:", err.Error())
		return nil
	}
	return conn
}

func TestConnect(t *testing.T) {
	conn, err := sql.Open("mssql", makeConnStr())
	if err != nil {
		t.Error("Open connection failed:", err.Error())
		return
	}
	defer conn.Close()
}

func TestBadConnect(t *testing.T) {
	badDsns := []string{
		//"Server=badhost",
		fmt.Sprintf("Server=%s\\%s;User ID=baduser;Password=badpwd",
			os.Getenv("HOST"), os.Getenv("INSTANCE")),
	}
	for _, badDsn := range badDsns {
		conn, err := sql.Open("mssql", badDsn)
		if err != nil {
			t.Error("Open connection failed:", err.Error())
		}
		defer conn.Close()
		err = conn.Ping()
		if err == nil {
			t.Error("Ping should fail for connection: ", badDsn)
		}
	}
}

func simpleQuery(conn *sql.DB, t *testing.T) (stmt *sql.Stmt) {
	stmt, err := conn.Prepare("select 1 as a")
	if err != nil {
		t.Error("Prepare failed:", err.Error())
		return nil
	}
	return stmt
}

func checkSimpleQuery(rows *sql.Rows, t *testing.T) {
	numrows := 0
	for rows.Next() {
		var val int
		err := rows.Scan(&val)
		if err != nil {
			t.Error("Scan failed:", err.Error())
		}
		if val != 1 {
			t.Error("query should return 1")
		}
		numrows++
	}
	if numrows != 1 {
		t.Error("query should return 1 row, returned", numrows)
	}
}

func TestQuery(t *testing.T) {
	conn := open(t)
	if conn == nil {
		return
	}
	defer conn.Close()

	stmt := simpleQuery(conn, t)
	if stmt == nil {
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		t.Error("Query failed:", err.Error())
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		t.Error("getting columns failed", err.Error())
	}
	if len(columns) != 1 && columns[0] != "a" {
		t.Error("returned incorrect columns (expected ['a']):", columns)
	}

	checkSimpleQuery(rows, t)
}

func TestMultipleQueriesSequentialy(t *testing.T) {

	conn := open(t)
	defer conn.Close()

	stmt, err := conn.Prepare("select 1 as a")
	if err != nil {
		t.Error("Prepare failed:", err.Error())
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		t.Error("Query failed:", err.Error())
		return
	}
	defer rows.Close()
	checkSimpleQuery(rows, t)

	rows, err = stmt.Query()
	if err != nil {
		t.Error("Query failed:", err.Error())
		return
	}
	defer rows.Close()
	checkSimpleQuery(rows, t)
}

func TestMultipleQueryClose(t *testing.T) {
	conn := open(t)
	defer conn.Close()

	stmt, err := conn.Prepare("select 1 as a")
	if err != nil {
		t.Error("Prepare failed:", err.Error())
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		t.Error("Query failed:", err.Error())
		return
	}
	rows.Close()

	rows, err = stmt.Query()
	if err != nil {
		t.Error("Query failed:", err.Error())
		return
	}
	defer rows.Close()
	checkSimpleQuery(rows, t)
}

func TestPing(t *testing.T) {
	conn := open(t)
	defer conn.Close()
	conn.Ping()
}

func TestSecureWithInvalidHostName(t *testing.T) {
	dsn := makeConnStr() + ";Encrypt=true;TrustServerCertificate=false;hostNameInCertificate=foo.bar"
	conn, err := sql.Open("mssql", dsn)
	if err != nil {
		t.Fatal("Open connection failed:", err.Error())
	}
	defer conn.Close()
	err = conn.Ping()
	if err == nil {
		t.Fatal("Connected to fake foo.bar server")
	}
}

func TestSecureConnection(t *testing.T) {
	dsn := makeConnStr() + ";Encrypt=true;TrustServerCertificate=true"
	conn, err := sql.Open("mssql", dsn)
	if err != nil {
		t.Fatal("Open connection failed:", err.Error())
	}
	defer conn.Close()
	var msg string
	err = conn.QueryRow("select 'secret'").Scan(&msg)
	if err != nil {
		t.Fatal("cannot scan value", err)
	}
	if msg != "secret" {
		t.Fatal("expected secret, got: ", msg)
	}
	var secure bool
	err = conn.QueryRow("select encrypt_option from sys.dm_exec_connections where session_id=@@SPID").Scan(&secure)
	if err != nil {
		t.Fatal("cannot scan value", err)
	}
	if !secure {
		t.Fatal("connection is not encrypted")
	}
}

func TestParseConnectParamsKeepAlive(t *testing.T) {
	params := parseConnectionString("keepAlive=60")
	parsedParams, err := parseConnectParams(params)
	if err != nil {
		t.Fatal("cannot parse params: ", err)
	}

	if parsedParams.keepAlive != time.Duration(60)*time.Second {
		t.Fail()
	}
}
