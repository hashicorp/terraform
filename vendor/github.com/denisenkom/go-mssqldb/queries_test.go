package mssql

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSelect(t *testing.T) {
	conn := open(t)
	defer conn.Close()

	type testStruct struct {
		sql string
		val interface{}
	}

	longstr := strings.Repeat("x", 10000)

	values := []testStruct{
		{"1", int64(1)},
		{"-1", int64(-1)},
		{"cast(1 as int)", int64(1)},
		{"cast(-1 as int)", int64(-1)},
		{"cast(1 as tinyint)", int64(1)},
		{"cast(1 as smallint)", int64(1)},
		{"cast(-1 as smallint)", int64(-1)},
		{"cast(1 as bigint)", int64(1)},
		{"cast(-1 as bigint)", int64(-1)},
		{"cast(1 as bit)", true},
		{"cast(0 as bit)", false},
		{"'abc'", string("abc")},
		{"cast(0.5 as float)", float64(0.5)},
		{"cast(0.5 as real)", float64(0.5)},
		{"cast(1 as decimal)", []byte("1")},
		{"cast(1.2345 as money)", []byte("1.2345")},
		{"cast(-1.2345 as money)", []byte("-1.2345")},
		{"cast(1.2345 as smallmoney)", []byte("1.2345")},
		{"cast(-1.2345 as smallmoney)", []byte("-1.2345")},
		{"cast(0.5 as decimal(18,1))", []byte("0.5")},
		{"cast(-0.5 as decimal(18,1))", []byte("-0.5")},
		{"cast(-0.5 as numeric(18,1))", []byte("-0.5")},
		{"cast(4294967296 as numeric(20,0))", []byte("4294967296")},
		{"cast(-0.5 as numeric(18,2))", []byte("-0.50")},
		{"N'abc'", string("abc")},
		{"cast(null as nvarchar(3))", nil},
		{"NULL", nil},
		{"cast('2000-01-01' as datetime)", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"cast('2000-01-01T12:13:14.12' as datetime)",
			time.Date(2000, 1, 1, 12, 13, 14, 120000000, time.UTC)},
		{"cast('2014-06-26 11:08:09.673' as datetime)", time.Date(2014, 06, 26, 11, 8, 9, 673000000, time.UTC)},
		{"cast(NULL as datetime)", nil},
		{"cast('2000-01-01T12:13:00' as smalldatetime)",
			time.Date(2000, 1, 1, 12, 13, 0, 0, time.UTC)},
		{"cast(0x6F9619FF8B86D011B42D00C04FC964FF as uniqueidentifier)",
			[]byte{0x6F, 0x96, 0x19, 0xFF, 0x8B, 0x86, 0xD0, 0x11, 0xB4, 0x2D, 0x00, 0xC0, 0x4F, 0xC9, 0x64, 0xFF}},
		{"cast(NULL as uniqueidentifier)", nil},
		{"cast(0x1234 as varbinary(2))", []byte{0x12, 0x34}},
		{"cast(N'abc' as nvarchar(max))", "abc"},
		{"cast(null as nvarchar(max))", nil},
		{"cast('<root/>' as xml)", "<root/>"},
		{"cast('abc' as text)", "abc"},
		{"cast(null as text)", nil},
		{"cast(N'abc' as ntext)", "abc"},
		{"cast(0x1234 as image)", []byte{0x12, 0x34}},
		{"cast(N'проверка' as nvarchar(max))", "проверка"},
		{"cast(N'Δοκιμή' as nvarchar(max))", "Δοκιμή"},
		{"cast(cast(N'สวัสดี' as nvarchar(max)) collate Thai_CI_AI as varchar(max))", "สวัสดี"},                // cp874
		{"cast(cast(N'你好' as nvarchar(max)) collate Chinese_PRC_CI_AI as varchar(max))", "你好"},                 // cp936
		{"cast(cast(N'こんにちは' as nvarchar(max)) collate Japanese_CI_AI as varchar(max))", "こんにちは"},              // cp939
		{"cast(cast(N'안녕하세요.' as nvarchar(max)) collate Korean_90_CI_AI as varchar(max))", "안녕하세요."},           // cp949
		{"cast(cast(N'你好' as nvarchar(max)) collate Chinese_Hong_Kong_Stroke_90_CI_AI as varchar(max))", "你好"}, // cp950
		{"cast(cast(N'cześć' as nvarchar(max)) collate Polish_CI_AI as varchar(max))", "cześć"},                // cp1250
		{"cast(cast(N'Алло' as nvarchar(max)) collate Cyrillic_General_CI_AI as varchar(max))", "Алло"},        // cp1251
		{"cast(cast(N'Bonjour' as nvarchar(max)) collate French_CI_AI as varchar(max))", "Bonjour"},            // cp1252
		{"cast(cast(N'Γεια σας' as nvarchar(max)) collate Greek_CI_AI as varchar(max))", "Γεια σας"},           // cp1253
		{"cast(cast(N'Merhaba' as nvarchar(max)) collate Turkish_CI_AI as varchar(max))", "Merhaba"},           // cp1254
		{"cast(cast(N'שלום' as nvarchar(max)) collate Hebrew_CI_AI as varchar(max))", "שלום"},                  // cp1255
		{"cast(cast(N'مرحبا' as nvarchar(max)) collate Arabic_CI_AI as varchar(max))", "مرحبا"},                // cp1256
		{"cast(cast(N'Sveiki' as nvarchar(max)) collate Lithuanian_CI_AI as varchar(max))", "Sveiki"},          // cp1257
		{"cast(cast(N'chào' as nvarchar(max)) collate Vietnamese_CI_AI as varchar(max))", "chào"},              // cp1258
		{fmt.Sprintf("cast(N'%s' as nvarchar(max))", longstr), longstr},
		{"cast(NULL as sql_variant)", nil},
		{"cast(cast(0x6F9619FF8B86D011B42D00C04FC964FF as uniqueidentifier) as sql_variant)",
			[]byte{0x6F, 0x96, 0x19, 0xFF, 0x8B, 0x86, 0xD0, 0x11, 0xB4, 0x2D, 0x00, 0xC0, 0x4F, 0xC9, 0x64, 0xFF}},
		{"cast(cast(1 as bit) as sql_variant)", true},
		{"cast(cast(10 as tinyint) as sql_variant)", int64(10)},
		{"cast(cast(-10 as smallint) as sql_variant)", int64(-10)},
		{"cast(cast(-20 as int) as sql_variant)", int64(-20)},
		{"cast(cast(-20 as bigint) as sql_variant)", int64(-20)},
		{"cast(cast('2000-01-01' as datetime) as sql_variant)", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"cast(cast('2000-01-01T12:13:00' as smalldatetime) as sql_variant)",
			time.Date(2000, 1, 1, 12, 13, 0, 0, time.UTC)},
		{"cast(cast(0.125 as real) as sql_variant)", float64(0.125)},
		{"cast(cast(0.125 as float) as sql_variant)", float64(0.125)},
		{"cast(cast(1.2345 as smallmoney) as sql_variant)", []byte("1.2345")},
		{"cast(cast(1.2345 as money) as sql_variant)", []byte("1.2345")},
		{"cast(cast(0x1234 as varbinary(2)) as sql_variant)", []byte{0x12, 0x34}},
		{"cast(cast(0x1234 as binary(2)) as sql_variant)", []byte{0x12, 0x34}},
		{"cast(cast(-0.5 as decimal(18,1)) as sql_variant)", []byte("-0.5")},
		{"cast(cast(-0.5 as numeric(18,1)) as sql_variant)", []byte("-0.5")},
		{"cast(cast('abc' as varchar(3)) as sql_variant)", "abc"},
		{"cast(cast('abc' as char(3)) as sql_variant)", "abc"},
		{"cast(N'abc' as sql_variant)", "abc"},
	}

	for _, test := range values {
		stmt, err := conn.Prepare("select " + test.sql)
		if err != nil {
			t.Error("Prepare failed:", test.sql, err.Error())
			return
		}
		defer stmt.Close()

		row := stmt.QueryRow()
		var retval interface{}
		err = row.Scan(&retval)
		if err != nil {
			t.Error("Scan failed:", test.sql, err.Error())
			continue
		}
		var same bool
		switch decodedval := retval.(type) {
		case []byte:
			switch decodedvaltest := test.val.(type) {
			case []byte:
				same = bytes.Equal(decodedval, decodedvaltest)
			default:
				same = false
			}
		default:
			same = retval == test.val
		}
		if !same {
			t.Errorf("Values don't match '%s' '%s' for test: %s", retval, test.val, test.sql)
			continue
		}
	}
}

func TestSelectNewTypes(t *testing.T) {
	conn := open(t)
	defer conn.Close()
	var ver string
	err := conn.QueryRow("select SERVERPROPERTY('productversion')").Scan(&ver)
	if err != nil {
		t.Fatalf("cannot select productversion: %s", err)
	}
	var n int
	_, err = fmt.Sscanf(ver, "%d", &n)
	if err != nil {
		t.Fatalf("cannot parse productversion: %s", err)
	}
	// 8 is SQL 2000, 9 is SQL 2005, 10 is SQL 2008, 11 is SQL 2012
	if n < 10 {
		return
	}
	// run tests for new data types available only in SQL Server 2008 and later
	type testStruct struct {
		sql string
		val interface{}
	}
	values := []testStruct{
		{"cast('2000-01-01' as date)",
			time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"cast(NULL as date)", nil},
		{"cast('00:00:45.123' as time(3))",
			time.Date(1, 1, 1, 00, 00, 45, 123000000, time.UTC)},
		{"cast('11:56:45.123' as time(3))",
			time.Date(1, 1, 1, 11, 56, 45, 123000000, time.UTC)},
		{"cast('11:56:45' as time(0))",
			time.Date(1, 1, 1, 11, 56, 45, 0, time.UTC)},
		{"cast('2010-11-15T11:56:45.123' as datetime2(3))",
			time.Date(2010, 11, 15, 11, 56, 45, 123000000, time.UTC)},
		{"cast('2010-11-15T11:56:45' as datetime2(0))",
			time.Date(2010, 11, 15, 11, 56, 45, 0, time.UTC)},
		//{"cast('2010-11-15T11:56:45.123+10:00' as datetimeoffset(3))",
		// time.Date(2010, 11, 15, 11, 56, 45, 123000000, time.FixedZone("", 10*60*60)) },
		{"cast(cast('2000-01-01' as date) as sql_variant)",
			time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"cast(cast('00:00:45.123' as time(3)) as sql_variant)",
			time.Date(1, 1, 1, 00, 00, 45, 123000000, time.UTC)},
		{"cast(cast('2010-11-15T11:56:45.123' as datetime2(3)) as sql_variant)",
			time.Date(2010, 11, 15, 11, 56, 45, 123000000, time.UTC)},
		//{"cast(cast('2010-11-15T11:56:45.123+10:00' as datetimeoffset(3)) as sql_variant)",
		// time.Date(2010, 11, 15, 11, 56, 45, 123000000, time.FixedZone("", 10*60*60)) },
	}
	for _, test := range values {
		stmt, err := conn.Prepare("select " + test.sql)
		if err != nil {
			t.Error("Prepare failed:", test.sql, err.Error())
			return
		}
		defer stmt.Close()

		row := stmt.QueryRow()
		var retval interface{}
		err = row.Scan(&retval)
		if err != nil {
			t.Error("Scan failed:", test.sql, err.Error())
			continue
		}
		if retval != test.val {
			t.Errorf("Values don't match '%s' '%s' for test: %s", retval, test.val, test.sql)
			continue
		}
	}
}

func TestTrans(t *testing.T) {
	conn := open(t)
	defer conn.Close()

	var tx *sql.Tx
	var err error
	if tx, err = conn.Begin(); err != nil {
		t.Fatal("Begin failed", err.Error())
	}
	if err = tx.Commit(); err != nil {
		t.Fatal("Commit failed", err.Error())
	}

	if tx, err = conn.Begin(); err != nil {
		t.Fatal("Begin failed", err.Error())
	}
	if _, err = tx.Exec("create table #abc (fld int)"); err != nil {
		t.Fatal("Create table failed", err.Error())
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal("Rollback failed", err.Error())
	}
}

func TestParams(t *testing.T) {
	longstr := strings.Repeat("x", 10000)
	longbytes := make([]byte, 10000)
	values := []interface{}{
		int64(5),
		"hello",
		"",
		[]byte{1, 2, 3},
		[]byte{},
		float64(1.12313554),
		true,
		false,
		nil,
		longstr,
		longbytes,
	}

	conn := open(t)
	defer conn.Close()

	for _, val := range values {
		row := conn.QueryRow("select ?", val)
		var retval interface{}
		err := row.Scan(&retval)
		if err != nil {
			t.Error("Scan failed", err.Error())
			return
		}
		var same bool
		switch decodedval := retval.(type) {
		case []byte:
			switch decodedvaltest := val.(type) {
			case []byte:
				same = bytes.Equal(decodedval, decodedvaltest)
			default:
				same = false
			}
		default:
			same = retval == val
		}
		if !same {
			t.Error("Value don't match", retval, val)
			return
		}
	}
}

func TestExec(t *testing.T) {
	conn := open(t)
	defer conn.Close()

	res, err := conn.Exec("create table #abc (fld int)")
	if err != nil {
		t.Fatal("Exec failed", err.Error())
	}
	_ = res
}

func TestDefaultTimeout(t *testing.T) {
	if testing.Short() {
		return
	}
	conn := open(t)
	defer conn.Close()

	res, err := conn.Exec("waitfor delay '00:31'")
	if err == nil {
		t.Fatal("Exec should fail with timeout")
	}
	if neterr, ok := err.(net.Error); !ok || !neterr.Timeout() {
		t.Fatal("Exec should fail with timeout, failed with", err)
	}
	_ = res
}

func TestShortTimeout(t *testing.T) {
	if testing.Short() {
		return
	}
	dsn := makeConnStr() + ";Connection Timeout=2"
	conn, err := sql.Open("mssql", dsn)
	if err != nil {
		t.Fatal("Open connection failed:", err.Error())
	}
	defer conn.Close()

	res, err := conn.Exec("waitfor delay '00:03'")
	if err == nil {
		t.Fatal("Exec should fail with timeout")
	}
	if neterr, ok := err.(net.Error); !ok || !neterr.Timeout() {
		t.Fatal("Exec should fail with timeout, failed with", err)
	}
	_ = res
}

func TestTwoQueries(t *testing.T) {
	conn := open(t)
	defer conn.Close()

	rows, err := conn.Query("select 1")
	if err != nil {
		t.Fatal("First exec failed", err)
	}
	if !rows.Next() {
		t.Fatal("First query didn't return row")
	}
	var i int
	if err = rows.Scan(&i); err != nil {
		t.Fatal("Scan failed", err)
	}
	if i != 1 {
		t.Fatalf("Wrong value returned %d, should be 1", i)
	}

	if rows, err = conn.Query("select 2"); err != nil {
		t.Fatal("Second query failed", err)
	}
	if !rows.Next() {
		t.Fatal("Second query didn't return row")
	}
	if err = rows.Scan(&i); err != nil {
		t.Fatal("Scan failed", err)
	}
	if i != 2 {
		t.Fatalf("Wrong value returned %d, should be 2", i)
	}
}

func TestError(t *testing.T) {
	conn := open(t)
	defer conn.Close()

	_, err := conn.Query("exec bad")
	if err == nil {
		t.Fatal("Query should fail")
	}

	if sqlerr, ok := err.(Error); !ok {
		t.Fatalf("Should be sql error, actually %T, %v", err, err)
	} else {
		if sqlerr.Number != 2812 { // Could not find stored procedure 'bad'
			t.Fatalf("Should be specific error code 2812, actually %d %s", sqlerr.Number, sqlerr)
		}
	}
}

func TestQueryNoRows(t *testing.T) {
	conn := open(t)
	defer conn.Close()

	var rows *sql.Rows
	var err error
	if rows, err = conn.Query("create table #abc (fld int)"); err != nil {
		t.Fatal("Query failed", err)
	}
	if rows.Next() {
		t.Fatal("Query shoulnd't return any rows")
	}
}

func TestQueryManyNullsRow(t *testing.T) {
	conn := open(t)
	defer conn.Close()

	var row *sql.Row
	var err error
	if row = conn.QueryRow("select null, null, null, null, null, null, null, null"); err != nil {
		t.Fatal("Query failed", err)
	}
	var v [8]sql.NullInt64
	if err = row.Scan(&v[0], &v[1], &v[2], &v[3], &v[4], &v[5], &v[6], &v[7]); err != nil {
		t.Fatal("Scan failed", err)
	}
}

func TestOrderBy(t *testing.T) {
	conn := open(t)
	defer conn.Close()

	tx, err := conn.Begin()
	if err != nil {
		t.Fatal("Begin tran failed", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("if (exists(select * from INFORMATION_SCHEMA.TABLES where TABLE_NAME='tbl')) drop table tbl")
	if err != nil {
		t.Fatal("Drop table failed", err)
	}

	_, err = tx.Exec("create table tbl (fld1 int primary key, fld2 int)")
	if err != nil {
		t.Fatal("Create table failed", err)
	}
	_, err = tx.Exec("insert into tbl (fld1, fld2) values (1, 2)")
	if err != nil {
		t.Fatal("Insert failed", err)
	}
	_, err = tx.Exec("insert into tbl (fld1, fld2) values (2, 1)")
	if err != nil {
		t.Fatal("Insert failed", err)
	}

	rows, err := tx.Query("select * from tbl order by fld1")
	if err != nil {
		t.Fatal("Query failed", err)
	}

	for rows.Next() {
		var fld1 int32
		var fld2 int32
		err = rows.Scan(&fld1, &fld2)
		if err != nil {
			t.Fatal("Scan failed", err)
		}
	}

	err = rows.Err()
	if err != nil {
		t.Fatal("Rows have errors", err)
	}
}

func TestScanDecimal(t *testing.T) {
	conn := open(t)
	defer conn.Close()

	var f float64
	err := conn.QueryRow("select cast(0.5 as numeric(25,1))").Scan(&f)
	if err != nil {
		t.Error("query row / scan failed:", err.Error())
		return
	}
	if math.Abs(f-0.5) > 0.000001 {
		t.Error("Value is not 0.5:", f)
		return
	}

	var s string
	err = conn.QueryRow("select cast(-0.05 as numeric(25,2))").Scan(&s)
	if err != nil {
		t.Error("query row / scan failed:", err.Error())
		return
	}
	if s != "-0.05" {
		t.Error("Value is not -0.05:", s)
		return
	}
}

func TestAffectedRows(t *testing.T) {
	conn := open(t)
	defer conn.Close()

	tx, err := conn.Begin()
	if err != nil {
		t.Fatal("Begin tran failed", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec("create table #foo (bar int)")
	if err != nil {
		t.Fatal("create table failed")
	}
	n, err := res.RowsAffected()
	if err != nil {
		t.Fatal("rows affected failed")
	}
	if n != 0 {
		t.Error("Expected 0 rows affected, got ", n)
	}

	res, err = tx.Exec("insert into #foo (bar) values (1)")
	if err != nil {
		t.Fatal("insert failed")
	}
	n, err = res.RowsAffected()
	if err != nil {
		t.Fatal("rows affected failed")
	}
	if n != 1 {
		t.Error("Expected 1 row affected, got ", n)
	}

	res, err = tx.Exec("insert into #foo (bar) values (?)", 2)
	if err != nil {
		t.Fatal("insert failed")
	}
	n, err = res.RowsAffected()
	if err != nil {
		t.Fatal("rows affected failed")
	}
	if n != 1 {
		t.Error("Expected 1 row affected, got ", n)
	}
}

func TestIdentity(t *testing.T) {
	conn := open(t)
	defer conn.Close()

	tx, err := conn.Begin()
	if err != nil {
		t.Fatal("Begin tran failed", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec("create table #foo (bar int identity, baz int unique)")
	if err != nil {
		t.Fatal("create table failed")
	}

	res, err = tx.Exec("insert into #foo (baz) values (1)")
	if err != nil {
		t.Fatal("insert failed")
	}
	n, err := res.LastInsertId()
	if err != nil {
		t.Fatal("last insert id failed")
	}
	if n != 1 {
		t.Error("Expected 1 for identity, got ", n)
	}

	res, err = tx.Exec("insert into #foo (baz) values (20)")
	if err != nil {
		t.Fatal("insert failed")
	}
	n, err = res.LastInsertId()
	if err != nil {
		t.Fatal("last insert id failed")
	}
	if n != 2 {
		t.Error("Expected 2 for identity, got ", n)
	}

	res, err = tx.Exec("insert into #foo (baz) values (1)")
	if err == nil {
		t.Fatal("insert should fail")
	}

	res, err = tx.Exec("insert into #foo (baz) values (?)", 1)
	if err == nil {
		t.Fatal("insert should fail")
	}
}

func TestDateTimeParam(t *testing.T) {
	conn := open(t)
	defer conn.Close()

	t1, err := time.Parse("2006-01-02 15:04:05.99", "2004-06-03 12:13:14.15")
	if err != nil {
		t.Error("time parse failed", err.Error())
		return
	}
	var t2 time.Time
	err = conn.QueryRow("select ?", t1).Scan(&t2)
	if err != nil {
		t.Error("select / scan failed", err.Error())
		return
	}
	if t1.Sub(t2) != 0 {
		t.Errorf("datetime does not match: '%s' '%s' delta: %d", t1, t2, t1.Sub(t2))
		return
	}
}

func TestBigQuery(t *testing.T) {
	conn := open(t)
	defer conn.Close()
	rows, err := conn.Query(`WITH n(n) AS
		(
		    SELECT 1
		    UNION ALL
		    SELECT n+1 FROM n WHERE n < 10000
		)
		SELECT n, @@version FROM n ORDER BY n
		OPTION (MAXRECURSION 10000);`)
	if err != nil {
		t.Fatal("cannot exec query", err)
	}
	rows.Next()
	rows.Close()
	var res int
	err = conn.QueryRow("select 0").Scan(&res)
	if err != nil {
		t.Fatal("cannot scan value", err)
	}
	if res != 0 {
		t.Fatal("expected 0, got ", res)
	}
}

func TestBug32(t *testing.T) {
	conn := open(t)
	defer conn.Close()

	tx, err := conn.Begin()
	if err != nil {
		t.Fatal("Begin tran failed", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("if (exists(select * from INFORMATION_SCHEMA.TABLES where TABLE_NAME='tbl')) drop table tbl")
	if err != nil {
		t.Fatal("Drop table failed", err)
	}

	_, err = tx.Exec("create table tbl(a int primary key,fld bit null)")
	if err != nil {
		t.Fatal("Create table failed", err)
	}

	_, err = tx.Exec("insert into tbl (a,fld) values (1,nullif(?, ''))", "")
	if err != nil {
		t.Fatal("Insert failed", err)
	}
}

func TestLogging(t *testing.T) {
	flags := log.Flags()
	defer func() {
		log.SetFlags(flags)
		log.SetOutput(os.Stderr)
	}()
	log.SetFlags(0)
	var b bytes.Buffer
	log.SetOutput(&b)

	dsn := makeConnStr() + ";Log=2"
	conn, err := sql.Open("mssql", dsn)
	if err != nil {
		t.Fatal("Open connection failed:", err.Error())
	}
	defer conn.Close()
	_, err = conn.Exec("print 'test'")
	if err != nil {
		t.Fatal("Exec print failed", err.Error())
	}
	if b.String() != "test\n" {
		t.Fatal("logging test failed, got", b.String())
	}
}
