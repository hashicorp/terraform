package native

import (
	"bytes"
	"fmt"
	"github.com/ziutek/mymysql/mysql"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"
)

var (
	my     mysql.Conn
	user   = "testuser"
	passwd = "TestPasswd9"
	dbname = "test"
	//conn   = []string{"unix", "", "/var/run/mysqld/mysqld.sock"}
	conn  = []string{"", "", "127.0.0.1:3306"}
	debug = false
)

type RowsResErr struct {
	rows []mysql.Row
	res  mysql.Result
	err  error
}

func query(sql string, params ...interface{}) *RowsResErr {
	rows, res, err := my.Query(sql, params...)
	return &RowsResErr{rows, res, err}
}

func exec(stmt *Stmt, params ...interface{}) *RowsResErr {
	rows, res, err := stmt.Exec(params...)
	return &RowsResErr{rows, res, err}
}

func checkErr(t *testing.T, err error, exp_err error) {
	if err != exp_err {
		if exp_err == nil {
			t.Fatalf("Error: %v", err)
		} else {
			t.Fatalf("Error: %v\nExpected error: %v", err, exp_err)
		}
	}
}

func checkWarnCount(t *testing.T, res_cnt, exp_cnt int) {
	if res_cnt != exp_cnt {
		t.Errorf("Warning count: res=%d exp=%d", res_cnt, exp_cnt)
		rows, res, err := my.Query("show warnings")
		if err != nil {
			t.Fatal("Can't get warrnings from MySQL", err)
		}
		for _, row := range rows {
			t.Errorf("%s: \"%s\"", row.Str(res.Map("Level")),
				row.Str(res.Map("Message")))
		}
		t.FailNow()
	}
}

func checkErrWarn(t *testing.T, res, exp *RowsResErr) {
	checkErr(t, res.err, exp.err)
	checkWarnCount(t, res.res.WarnCount(), exp.res.WarnCount())
}

func types(row mysql.Row) (tt []reflect.Type) {
	tt = make([]reflect.Type, len(row))
	for ii, val := range row {
		tt[ii] = reflect.TypeOf(val)
	}
	return
}

func checkErrWarnRows(t *testing.T, res, exp *RowsResErr) {
	checkErrWarn(t, res, exp)
	if !reflect.DeepEqual(res.rows, exp.rows) {
		rlen := len(res.rows)
		elen := len(exp.rows)
		t.Error("Rows are different!")
		t.Errorf("len/cap: res=%d/%d exp=%d/%d",
			rlen, cap(res.rows), elen, cap(exp.rows))
		max := rlen
		if elen > max {
			max = elen
		}
		for ii := 0; ii < max; ii++ {
			if ii < len(res.rows) {
				t.Errorf("%d: res type: %s", ii, types(res.rows[ii]))
			} else {
				t.Errorf("%d: res: ------", ii)
			}
			if ii < len(exp.rows) {
				t.Errorf("%d: exp type: %s", ii, types(exp.rows[ii]))
			} else {
				t.Errorf("%d: exp: ------", ii)
			}
			if ii < len(res.rows) {
				t.Error(" res: ", res.rows[ii])
			}
			if ii < len(exp.rows) {
				t.Error(" exp: ", exp.rows[ii])
			}
			if ii < len(res.rows) {
				t.Errorf(" res: %#v", res.rows[ii][2])
			}
			if ii < len(exp.rows) {
				t.Errorf(" exp: %#v", exp.rows[ii][2])
			}
		}
		t.FailNow()
	}
}

func checkResult(t *testing.T, res, exp *RowsResErr) {
	checkErrWarnRows(t, res, exp)
	r, e := res.res.(*Result), exp.res.(*Result)
	if r.my != e.my || r.binary != e.binary || r.status_only != e.status_only ||
		r.status&0xdf != e.status || !bytes.Equal(r.message, e.message) ||
		r.affected_rows != e.affected_rows ||
		r.eor_returned != e.eor_returned ||
		!reflect.DeepEqual(res.rows, exp.rows) || res.err != exp.err {
		t.Fatalf("Bad result:\nres=%+v\nexp=%+v", res.res, exp.res)
	}
}

func cmdOK(affected uint64, binary, eor bool) *RowsResErr {
	return &RowsResErr{
		res: &Result{
			my:            my.(*Conn),
			binary:        binary,
			status_only:   true,
			status:        0x2,
			message:       []byte{},
			affected_rows: affected,
			eor_returned:  eor,
		},
	}
}

func selectOK(rows []mysql.Row, binary bool) (exp *RowsResErr) {
	exp = cmdOK(0, binary, true)
	exp.rows = rows
	return
}

func myConnect(t *testing.T, with_dbname bool, max_pkt_size int) {
	if with_dbname {
		my = New(conn[0], conn[1], conn[2], user, passwd, dbname)
	} else {
		my = New(conn[0], conn[1], conn[2], user, passwd)
	}

	if max_pkt_size != 0 {
		my.SetMaxPktSize(max_pkt_size)
	}
	my.(*Conn).Debug = debug

	checkErr(t, my.Connect(), nil)
	checkResult(t, query("set names utf8"), cmdOK(0, false, true))
}

func myClose(t *testing.T) {
	checkErr(t, my.Close(), nil)
}

// Text queries tests

func TestUse(t *testing.T) {
	myConnect(t, false, 0)
	checkErr(t, my.Use(dbname), nil)
	myClose(t)
}

func TestPing(t *testing.T) {
	myConnect(t, false, 0)
	checkErr(t, my.Ping(), nil)
	myClose(t)
}

func TestQuery(t *testing.T) {
	myConnect(t, true, 0)
	query("drop table t") // Drop test table if exists
	checkResult(t, query("create table t (s varchar(40))"),
		cmdOK(0, false, true))

	exp := &RowsResErr{
		res: &Result{
			my:          my.(*Conn),
			field_count: 1,
			fields: []*mysql.Field{
				&mysql.Field{
					Catalog:  "def",
					Db:       "test",
					Table:    "Test",
					OrgTable: "T",
					Name:     "Str",
					OrgName:  "s",
					DispLen:  3 * 40, //varchar(40)
					Flags:    0,
					Type:     MYSQL_TYPE_VAR_STRING,
					Scale:    0,
				},
			},
			status:       mysql.SERVER_STATUS_AUTOCOMMIT,
			eor_returned: true,
		},
	}

	for ii := 0; ii > 10000; ii += 3 {
		var val interface{}
		if ii%10 == 0 {
			checkResult(t, query("insert t values (null)"),
				cmdOK(1, false, true))
			val = nil
		} else {
			txt := []byte(fmt.Sprintf("%d %d %d %d %d", ii, ii, ii, ii, ii))
			checkResult(t,
				query("insert t values ('%s')", txt), cmdOK(1, false, true))
			val = txt
		}
		exp.rows = append(exp.rows, mysql.Row{val})
	}

	checkResult(t, query("select s as Str from t as Test"), exp)
	checkResult(t, query("drop table t"), cmdOK(0, false, true))
	myClose(t)
}

// Prepared statements tests

type StmtErr struct {
	stmt *Stmt
	err  error
}

func prepare(sql string) *StmtErr {
	stmt, err := my.Prepare(sql)
	return &StmtErr{stmt.(*Stmt), err}
}

func checkStmt(t *testing.T, res, exp *StmtErr) {
	ok := res.err == exp.err &&
		// Skipping id
		reflect.DeepEqual(res.stmt.fields, exp.stmt.fields) &&
		res.stmt.field_count == exp.stmt.field_count &&
		res.stmt.param_count == exp.stmt.param_count &&
		res.stmt.warning_count == exp.stmt.warning_count &&
		res.stmt.status == exp.stmt.status

	if !ok {
		if exp.err == nil {
			checkErr(t, res.err, nil)
			checkWarnCount(t, res.stmt.warning_count, exp.stmt.warning_count)
			for _, v := range res.stmt.fields {
				fmt.Printf("%+v\n", v)
			}
			t.Fatalf("Bad result statement: res=%v exp=%v", res.stmt, exp.stmt)
		}
	}
}

func TestPrepared(t *testing.T) {
	myConnect(t, true, 0)
	query("drop table p") // Drop test table if exists
	checkResult(t,
		query(
			"create table p ("+
				"   ii int not null, ss varchar(20), dd datetime"+
				") default charset=utf8",
		),
		cmdOK(0, false, true),
	)

	exp := Stmt{
		fields: []*mysql.Field{
			&mysql.Field{
				Catalog: "def", Db: "test", Table: "p", OrgTable: "p",
				Name:    "i",
				OrgName: "ii",
				DispLen: 11,
				Flags:   _FLAG_NO_DEFAULT_VALUE | _FLAG_NOT_NULL,
				Type:    MYSQL_TYPE_LONG,
				Scale:   0,
			},
			&mysql.Field{
				Catalog: "def", Db: "test", Table: "p", OrgTable: "p",
				Name:    "s",
				OrgName: "ss",
				DispLen: 3 * 20, // varchar(20)
				Flags:   0,
				Type:    MYSQL_TYPE_VAR_STRING,
				Scale:   0,
			},
			&mysql.Field{
				Catalog: "def", Db: "test", Table: "p", OrgTable: "p",
				Name:    "d",
				OrgName: "dd",
				DispLen: 19,
				Flags:   _FLAG_BINARY,
				Type:    MYSQL_TYPE_DATETIME,
				Scale:   0,
			},
		},
		field_count:   3,
		param_count:   2,
		warning_count: 0,
		status:        0x2,
	}

	sel := prepare("select ii i, ss s, dd d from p where ii = ? and ss = ?")
	checkStmt(t, sel, &StmtErr{&exp, nil})

	all := prepare("select * from p")
	checkErr(t, all.err, nil)

	ins := prepare("insert p values (?, ?, ?)")
	checkErr(t, ins.err, nil)

	parsed, err := mysql.ParseTime("2012-01-17 01:10:10", time.Local)
	checkErr(t, err, nil)
	parsedZero, err := mysql.ParseTime("0000-00-00 00:00:00", time.Local)
	checkErr(t, err, nil)
	if !parsedZero.IsZero() {
		t.Fatalf("time '%s' isn't zero", parsedZero)
	}
	exp_rows := []mysql.Row{
		mysql.Row{
			2, "Taki tekst", time.Unix(123456789, 0),
		},
		mysql.Row{
			5, "Pąk róży", parsed,
		},
		mysql.Row{
			-3, "基础体温", parsed,
		},
		mysql.Row{
			11, "Zero UTC datetime", time.Unix(0, 0),
		},
		mysql.Row{
			17, mysql.Blob([]byte("Zero datetime")), parsedZero,
		},
		mysql.Row{
			23, []byte("NULL datetime"), (*time.Time)(nil),
		},
		mysql.Row{
			23, "NULL", nil,
		},
	}

	for _, row := range exp_rows {
		checkErrWarn(t,
			exec(ins.stmt, row[0], row[1], row[2]),
			cmdOK(1, true, true),
		)
	}

	// Convert values to expected result types
	for _, row := range exp_rows {
		for ii, col := range row {
			val := reflect.ValueOf(col)
			// Dereference pointers
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			switch val.Kind() {
			case reflect.Invalid:
				row[ii] = nil

			case reflect.String:
				row[ii] = []byte(val.String())

			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
				reflect.Int64:
				row[ii] = int32(val.Int())

			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
				reflect.Uint64:
				row[ii] = int32(val.Uint())

			case reflect.Slice:
				if val.Type().Elem().Kind() == reflect.Uint8 {
					bytes := make([]byte, val.Len())
					for ii := range bytes {
						bytes[ii] = val.Index(ii).Interface().(uint8)
					}
					row[ii] = bytes
				}
			}
		}
	}

	checkErrWarn(t, exec(sel.stmt, 2, "Taki tekst"), selectOK(exp_rows, true))
	checkErrWarnRows(t, exec(all.stmt), selectOK(exp_rows, true))

	checkResult(t, query("drop table p"), cmdOK(0, false, true))

	checkErr(t, sel.stmt.Delete(), nil)
	checkErr(t, all.stmt.Delete(), nil)
	checkErr(t, ins.stmt.Delete(), nil)

	myClose(t)
}

// Bind testing

func TestVarBinding(t *testing.T) {
	myConnect(t, true, 0)
	query("drop table t") // Drop test table if exists
	checkResult(t,
		query("create table t (id int primary key, str varchar(20))"),
		cmdOK(0, false, true),
	)

	ins, err := my.Prepare("insert t values (?, ?)")
	checkErr(t, err, nil)

	var (
		rre RowsResErr
		id  *int
		str *string
		ii  int
		ss  string
	)
	ins.Bind(&id, &str)

	i1 := 1
	s1 := "Ala"
	id = &i1
	str = &s1
	rre.res, rre.err = ins.Run()
	checkResult(t, &rre, cmdOK(1, true, false))

	i2 := 2
	s2 := "Ma kota!"
	id = &i2
	str = &s2

	rre.res, rre.err = ins.Run()
	checkResult(t, &rre, cmdOK(1, true, false))

	ins.Bind(&ii, &ss)
	ii = 3
	ss = "A kot ma Ale!"

	rre.res, rre.err = ins.Run()
	checkResult(t, &rre, cmdOK(1, true, false))

	sel, err := my.Prepare("select str from t where id = ?")
	checkErr(t, err, nil)

	rows, _, err := sel.Exec(1)
	checkErr(t, err, nil)
	if len(rows) != 1 || bytes.Compare([]byte(s1), rows[0].Bin(0)) != 0 {
		t.Fatal("First string don't match")
	}

	rows, _, err = sel.Exec(2)
	checkErr(t, err, nil)
	if len(rows) != 1 || bytes.Compare([]byte(s2), rows[0].Bin(0)) != 0 {
		t.Fatal("Second string don't match")
	}

	rows, _, err = sel.Exec(3)
	checkErr(t, err, nil)
	if len(rows) != 1 || bytes.Compare([]byte(ss), rows[0].Bin(0)) != 0 {
		t.Fatal("Thrid string don't match")
	}

	checkResult(t, query("drop table t"), cmdOK(0, false, true))
	myClose(t)
}

func TestBindStruct(t *testing.T) {
	myConnect(t, true, 0)
	query("drop table t") // Drop test table if exists
	checkResult(t,
		query("create table t (id int primary key, txt varchar(20), b bool)"),
		cmdOK(0, false, true),
	)

	ins, err := my.Prepare("insert t values (?, ?, ?)")
	checkErr(t, err, nil)
	sel, err := my.Prepare("select txt, b from t where id = ?")
	checkErr(t, err, nil)

	var (
		s struct {
			Id  int
			Txt string
			B   bool
		}
		rre RowsResErr
	)

	ins.Bind(&s)

	s.Id = 2
	s.Txt = "Ala ma kota."
	s.B = true

	rre.res, rre.err = ins.Run()
	checkResult(t, &rre, cmdOK(1, true, false))

	rows, _, err := sel.Exec(s.Id)
	checkErr(t, err, nil)
	if len(rows) != 1 || rows[0].Str(0) != s.Txt || rows[0].Bool(1) != s.B {
		t.Fatal("selected data don't match inserted data")
	}

	checkResult(t, query("drop table t"), cmdOK(0, false, true))
	myClose(t)
}

func TestDate(t *testing.T) {
	myConnect(t, true, 0)
	query("drop table d") // Drop test table if exists
	checkResult(t,
		query("create table d (id int, dd date, dt datetime, tt time)"),
		cmdOK(0, false, true),
	)

	test := []struct {
		dd, dt string
		tt     time.Duration
	}{
		{
			"2011-11-13",
			"2010-12-12 11:24:00",
			-time.Duration((128*3600 + 3*60 + 2) * 1e9),
		}, {
			"0000-00-00",
			"0000-00-00 00:00:00",
			time.Duration(0),
		},
	}

	ins, err := my.Prepare("insert d values (?, ?, ?, ?)")
	checkErr(t, err, nil)

	sel, err := my.Prepare("select id, tt from d where dd = ? && dt = ?")
	checkErr(t, err, nil)

	for i, r := range test {
		_, err = ins.Run(i, r.dd, r.dt, r.tt)
		checkErr(t, err, nil)

		sdt, err := mysql.ParseTime(r.dt, time.Local)
		checkErr(t, err, nil)
		sdd, err := mysql.ParseDate(r.dd)
		checkErr(t, err, nil)

		rows, _, err := sel.Exec(sdd, sdt)
		checkErr(t, err, nil)
		if rows == nil {
			t.Fatal("nil result")
		}
		if rows[0].Int(0) != i {
			t.Fatal("Bad id", rows[0].Int(1))
		}
		if rows[0][1].(time.Duration) != r.tt {
			t.Fatal("Bad tt", rows[0].Duration(1))
		}
	}

	checkResult(t, query("drop table d"), cmdOK(0, false, true))
	myClose(t)
}

func TestDateTimeZone(t *testing.T) {
	myConnect(t, true, 0)
	query("drop table d") // Drop test table if exists
	checkResult(t,
		query("create table d (dt datetime)"),
		cmdOK(0, false, true),
	)

	ins, err := my.Prepare("insert d values (?)")
	checkErr(t, err, nil)

	sel, err := my.Prepare("select dt from d")
	checkErr(t, err, nil)

	tstr := "2013-05-10 15:26:00.000000000"

	_, err = ins.Run(tstr)
	checkErr(t, err, nil)

	tt := make([]time.Time, 4)

	row, _, err := sel.ExecFirst()
	checkErr(t, err, nil)
	tt[0] = row.Time(0, time.UTC)
	tt[1] = row.Time(0, time.Local)
	row, _, err = my.QueryFirst("select dt from d")
	checkErr(t, err, nil)
	tt[2] = row.Time(0, time.UTC)
	tt[3] = row.Time(0, time.Local)
	for _, v := range tt {
		if v.Format(mysql.TimeFormat) != tstr {
			t.Fatal("Timezone problem:", tstr, "!=", v)
		}
	}

	checkResult(t, query("drop table d"), cmdOK(0, false, true))
	myClose(t)
}

// Big blob
func TestBigBlob(t *testing.T) {
	myConnect(t, true, 34*1024*1024)
	query("drop table p") // Drop test table if exists
	checkResult(t,
		query("create table p (id int primary key, bb longblob)"),
		cmdOK(0, false, true),
	)

	ins, err := my.Prepare("insert p values (?, ?)")
	checkErr(t, err, nil)

	sel, err := my.Prepare("select bb from p where id = ?")
	checkErr(t, err, nil)

	big_blob := make(mysql.Blob, 33*1024*1024)
	for ii := range big_blob {
		big_blob[ii] = byte(ii)
	}

	var (
		rre RowsResErr
		bb  mysql.Blob
		id  int
	)
	data := struct {
		Id int
		Bb mysql.Blob
	}{}

	// Individual parameters binding
	ins.Bind(&id, &bb)
	id = 1
	bb = big_blob

	// Insert full blob. Three packets are sended. First two has maximum length
	rre.res, rre.err = ins.Run()
	checkResult(t, &rre, cmdOK(1, true, false))

	// Struct binding
	ins.Bind(&data)
	data.Id = 2
	data.Bb = big_blob[0 : 32*1024*1024-31]

	// Insert part of blob - Two packets are sended. All has maximum length.
	rre.res, rre.err = ins.Run()
	checkResult(t, &rre, cmdOK(1, true, false))

	sel.Bind(&id)

	// Check first insert.
	tmr := "Too many rows"

	id = 1
	res, err := sel.Run()
	checkErr(t, err, nil)

	row, err := res.GetRow()
	checkErr(t, err, nil)
	end, err := res.GetRow()
	checkErr(t, err, nil)
	if end != nil {
		t.Fatal(tmr)
	}

	if bytes.Compare(row[0].([]byte), big_blob) != 0 {
		t.Fatal("Full blob data don't match")
	}

	// Check second insert.
	id = 2
	res, err = sel.Run()
	checkErr(t, err, nil)

	row, err = res.GetRow()
	checkErr(t, err, nil)
	end, err = res.GetRow()
	checkErr(t, err, nil)
	if end != nil {
		t.Fatal(tmr)
	}

	if bytes.Compare(row.Bin(res.Map("bb")), data.Bb) != 0 {
		t.Fatal("Partial blob data don't match")
	}

	checkResult(t, query("drop table p"), cmdOK(0, false, true))
	myClose(t)
}

// Test for empty result
func TestEmpty(t *testing.T) {
	checkNil := func(r mysql.Row) {
		if r != nil {
			t.Error("Not empty result")
		}
	}
	myConnect(t, true, 0)
	query("drop table e") // Drop test table if exists
	// Create table
	checkResult(t,
		query("create table e (id int)"),
		cmdOK(0, false, true),
	)
	// Text query
	res, err := my.Start("select * from e")
	checkErr(t, err, nil)
	row, err := res.GetRow()
	checkErr(t, err, nil)
	checkNil(row)
	row, err = res.GetRow()
	checkErr(t, err, mysql.ErrReadAfterEOR)
	checkNil(row)
	// Prepared statement
	sel, err := my.Prepare("select * from e")
	checkErr(t, err, nil)
	res, err = sel.Run()
	checkErr(t, err, nil)
	row, err = res.GetRow()
	checkErr(t, err, nil)
	checkNil(row)
	row, err = res.GetRow()
	checkErr(t, err, mysql.ErrReadAfterEOR)
	checkNil(row)
	// Drop test table
	checkResult(t, query("drop table e"), cmdOK(0, false, true))
}

// Reconnect test
func TestReconnect(t *testing.T) {
	myConnect(t, true, 0)
	query("drop table r") // Drop test table if exists
	checkResult(t,
		query("create table r (id int primary key, str varchar(20))"),
		cmdOK(0, false, true),
	)

	ins, err := my.Prepare("insert r values (?, ?)")
	checkErr(t, err, nil)
	sel, err := my.Prepare("select str from r where id = ?")
	checkErr(t, err, nil)

	params := struct {
		Id  int
		Str string
	}{}
	var sel_id int

	ins.Bind(&params)
	sel.Bind(&sel_id)

	checkErr(t, my.Reconnect(), nil)

	params.Id = 1
	params.Str = "Bla bla bla"
	_, err = ins.Run()
	checkErr(t, err, nil)

	checkErr(t, my.Reconnect(), nil)

	sel_id = 1
	res, err := sel.Run()
	checkErr(t, err, nil)

	row, err := res.GetRow()
	checkErr(t, err, nil)

	checkErr(t, res.End(), nil)

	if row == nil || row[0] == nil ||
		params.Str != row.Str(0) {
		t.Fatal("Bad result")
	}

	checkErr(t, my.Reconnect(), nil)

	checkResult(t, query("drop table r"), cmdOK(0, false, true))
	myClose(t)
}

// StmtSendLongData test

func TestSendLongData(t *testing.T) {
	myConnect(t, true, 64*1024*1024)
	query("drop table l") // Drop test table if exists
	checkResult(t,
		query("create table l (id int primary key, bb longblob)"),
		cmdOK(0, false, true),
	)
	ins, err := my.Prepare("insert l values (?, ?)")
	checkErr(t, err, nil)

	sel, err := my.Prepare("select bb from l where id = ?")
	checkErr(t, err, nil)

	var (
		rre RowsResErr
		id  int64
	)

	ins.Bind(&id, []byte(nil))
	sel.Bind(&id)

	// Prepare data
	data := make([]byte, 4*1024*1024)
	for ii := range data {
		data[ii] = byte(ii)
	}
	// Send long data twice
	checkErr(t, ins.SendLongData(1, data, 256*1024), nil)
	checkErr(t, ins.SendLongData(1, data, 512*1024), nil)

	id = 1
	rre.res, rre.err = ins.Run()
	checkResult(t, &rre, cmdOK(1, true, false))

	res, err := sel.Run()
	checkErr(t, err, nil)

	row, err := res.GetRow()
	checkErr(t, err, nil)

	checkErr(t, res.End(), nil)

	if row == nil || row[0] == nil ||
		bytes.Compare(append(data, data...), row.Bin(0)) != 0 {
		t.Fatal("Bad result")
	}

	file, err := ioutil.TempFile("", "mymysql_test-")
	checkErr(t, err, nil)
	filename := file.Name()
	defer os.Remove(filename)

	buf := make([]byte, 1024)
	for i := 0; i < 2048; i++ {
		_, err := file.Write(buf)
		checkErr(t, err, nil)
	}
	checkErr(t, file.Close(), nil)

	// Send long data from io.Reader twice
	file, err = os.Open(filename)
	checkErr(t, err, nil)
	checkErr(t, ins.SendLongData(1, file, 128*1024), nil)
	checkErr(t, file.Close(), nil)
	file, err = os.Open(filename)
	checkErr(t, err, nil)
	checkErr(t, ins.SendLongData(1, file, 1024*1024), nil)
	checkErr(t, file.Close(), nil)

	id = 2
	rre.res, rre.err = ins.Run()
	checkResult(t, &rre, cmdOK(1, true, false))

	res, err = sel.Run()
	checkErr(t, err, nil)

	row, err = res.GetRow()
	checkErr(t, err, nil)

	checkErr(t, res.End(), nil)

	// Read file for check result
	data, err = ioutil.ReadFile(filename)
	checkErr(t, err, nil)

	if row == nil || row[0] == nil ||
		bytes.Compare(append(data, data...), row.Bin(0)) != 0 {
		t.Fatal("Bad result")
	}

	checkResult(t, query("drop table l"), cmdOK(0, false, true))
	myClose(t)
}

func TestNull(t *testing.T) {
	myConnect(t, true, 0)
	query("drop table if exists n")
	checkResult(t,
		query("create table n (i int not null, n int)"),
		cmdOK(0, false, true),
	)
	ins, err := my.Prepare("insert n values (?, ?)")
	checkErr(t, err, nil)

	var (
		p   struct{ I, N *int }
		rre RowsResErr
	)
	ins.Bind(&p)

	p.I = new(int)
	p.N = new(int)

	*p.I = 0
	*p.N = 1
	rre.res, rre.err = ins.Run()
	checkResult(t, &rre, cmdOK(1, true, false))
	*p.I = 1
	p.N = nil
	rre.res, rre.err = ins.Run()
	checkResult(t, &rre, cmdOK(1, true, false))

	checkResult(t, query("insert n values (2, 1)"), cmdOK(1, false, true))
	checkResult(t, query("insert n values (3, NULL)"), cmdOK(1, false, true))

	rows, res, err := my.Query("select * from n")
	checkErr(t, err, nil)
	if len(rows) != 4 {
		t.Fatal("str: len(rows) != 4")
	}
	i := res.Map("i")
	n := res.Map("n")
	for k, row := range rows {
		switch {
		case row[i] == nil || row.Int(i) != k:
		case k%2 == 1 && row[n] != nil:
		case k%2 == 0 && (row[n] == nil || row.Int(n) != 1):
		default:
			continue
		}
		t.Fatalf("str row: %d = (%s, %s)", k, row[i], row[n])
	}

	sel, err := my.Prepare("select * from n")
	checkErr(t, err, nil)
	rows, res, err = sel.Exec()
	checkErr(t, err, nil)
	if len(rows) != 4 {
		t.Fatal("bin: len(rows) != 4")
	}
	i = res.Map("i")
	n = res.Map("n")
	for k, row := range rows {
		switch {
		case row[i] == nil || row.Int(i) != k:
		case k%2 == 1 && row[n] != nil:
		case k%2 == 0 && (row[n] == nil || row.Int(n) != 1):
		default:
			continue
		}
		t.Fatalf("bin row: %d = (%v, %v)", k, row[i], row[n])
	}

	checkResult(t, query("drop table n"), cmdOK(0, false, true))
}

func TestMultipleResults(t *testing.T) {
	myConnect(t, true, 0)
	query("drop table m") // Drop test table if exists
	checkResult(t,
		query("create table m (id int primary key, str varchar(20))"),
		cmdOK(0, false, true),
	)

	str := []string{"zero", "jeden", "dwa"}

	checkResult(t, query("insert m values (0, '%s')", str[0]),
		cmdOK(1, false, true))
	checkResult(t, query("insert m values (1, '%s')", str[1]),
		cmdOK(1, false, true))
	checkResult(t, query("insert m values (2, '%s')", str[2]),
		cmdOK(1, false, true))

	res, err := my.Start("select id from m; select str from m")
	checkErr(t, err, nil)

	for ii := 0; ; ii++ {
		row, err := res.GetRow()
		checkErr(t, err, nil)
		if row == nil {
			break
		}
		if row.Int(0) != ii {
			t.Fatal("Bad result")
		}
	}
	res, err = res.NextResult()
	checkErr(t, err, nil)
	for ii := 0; ; ii++ {
		row, err := res.GetRow()
		checkErr(t, err, nil)
		if row == nil {
			break
		}
		if row.Str(0) != str[ii] {
			t.Fatal("Bad result")
		}
	}

	checkResult(t, query("drop table m"), cmdOK(0, false, true))
	myClose(t)
}

func TestDecimal(t *testing.T) {
	myConnect(t, true, 0)

	query("drop table if exists d")
	checkResult(t,
		query("create table d (d decimal(4,2))"),
		cmdOK(0, false, true),
	)

	checkResult(t, query("insert d values (10.01)"), cmdOK(1, false, true))
	sql := "select * from d"
	sel, err := my.Prepare(sql)
	checkErr(t, err, nil)
	rows, res, err := sel.Exec()
	checkErr(t, err, nil)
	if len(rows) != 1 || rows[0][res.Map("d")].(float64) != 10.01 {
		t.Fatal(sql)
	}

	checkResult(t, query("drop table d"), cmdOK(0, false, true))
	myClose(t)
}

func TestMediumInt(t *testing.T) {
	myConnect(t, true, 0)
	query("DROP TABLE mi")
	checkResult(t,
		query(
			`CREATE TABLE mi (
				id INT PRIMARY KEY AUTO_INCREMENT,
				m MEDIUMINT
			)`,
		),
		cmdOK(0, false, true),
	)

	const n = 9

	for i := 0; i < n; i++ {
		res, err := my.Start("INSERT mi VALUES (0, %d)", i)
		checkErr(t, err, nil)
		if res.InsertId() != uint64(i+1) {
			t.Fatalf("Wrong insert id: %d, expected: %d", res.InsertId(), i+1)
		}
	}

	sel, err := my.Prepare("SELECT * FROM mi")
	checkErr(t, err, nil)

	res, err := sel.Run()
	checkErr(t, err, nil)

	i := 0
	for {
		row, err := res.GetRow()
		checkErr(t, err, nil)
		if row == nil {
			break
		}
		id, m := row.Int(0), row.Int(1)
		if id != i+1 || m != i {
			t.Fatalf("i=%d id=%d m=%d", i, id, m)
		}
		i++
	}
	if i != n {
		t.Fatalf("%d rows read, %d expected", i, n)
	}
	checkResult(t, query("drop table mi"), cmdOK(0, false, true))
}

func TestStoredProcedures(t *testing.T) {
	myConnect(t, true, 0)
	query("DROP PROCEDURE pr")
	query("DROP TABLE p")
	checkResult(t,
		query(
			`CREATE TABLE p (
				id INT PRIMARY KEY AUTO_INCREMENT,
				txt VARCHAR(8)	
			)`,
		),
		cmdOK(0, false, true),
	)
	_, err := my.Start(
		`CREATE PROCEDURE pr (IN i INT)
		BEGIN
			INSERT p VALUES (0, "aaa");
			SELECT * FROM p;
			SELECT i * id FROM p;
		END`,
	)
	checkErr(t, err, nil)

	res, err := my.Start("CALL pr(3)")
	checkErr(t, err, nil)

	rows, err := res.GetRows()
	checkErr(t, err, nil)
	if len(rows) != 1 || len(rows[0]) != 2 || rows[0].Int(0) != 1 || rows[0].Str(1) != "aaa" {
		t.Fatalf("Bad result set: %+v", rows)
	}

	res, err = res.NextResult()
	checkErr(t, err, nil)

	rows, err = res.GetRows()
	checkErr(t, err, nil)
	if len(rows) != 1 || len(rows[0]) != 1 || rows[0].Int(0) != 3 {
		t.Fatalf("Bad result set: %+v", rows)
	}

	res, err = res.NextResult()
	checkErr(t, err, nil)
	if !res.StatusOnly() {
		t.Fatalf("Result includes resultset at end of procedure: %+v", res)
	}

	_, err = my.Start("DROP PROCEDURE pr")
	checkErr(t, err, nil)

	checkResult(t, query("DROP TABLE p"), cmdOK(0, false, true))
}

// Benchamrks

func check(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func BenchmarkInsertSelect(b *testing.B) {
	b.StopTimer()

	my := New(conn[0], conn[1], conn[2], user, passwd, dbname)
	check(my.Connect())

	my.Start("drop table b") // Drop test table if exists

	_, err := my.Start("create table b (s varchar(40), i int)")
	check(err)

	for ii := 0; ii < 10000; ii++ {
		_, err := my.Start("insert b values ('%d-%d-%d', %d)", ii, ii, ii, ii)
		check(err)
	}

	b.StartTimer()

	for ii := 0; ii < b.N; ii++ {
		res, err := my.Start("select * from b")
		check(err)
		for {
			row, err := res.GetRow()
			check(err)
			if row == nil {
				break
			}
		}
	}

	b.StopTimer()

	_, err = my.Start("drop table b")
	check(err)
	check(my.Close())
}

func BenchmarkPreparedInsertSelect(b *testing.B) {
	b.StopTimer()

	my := New(conn[0], conn[1], conn[2], user, passwd, dbname)
	check(my.Connect())

	my.Start("drop table b") // Drop test table if exists

	_, err := my.Start("create table b (s varchar(40), i int)")
	check(err)

	ins, err := my.Prepare("insert b values (?, ?)")
	check(err)

	sel, err := my.Prepare("select * from b")
	check(err)

	for ii := 0; ii < 10000; ii++ {
		_, err := ins.Run(fmt.Sprintf("%d-%d-%d", ii, ii, ii), ii)
		check(err)
	}

	b.StartTimer()

	for ii := 0; ii < b.N; ii++ {
		res, err := sel.Run()
		check(err)
		for {
			row, err := res.GetRow()
			check(err)
			if row == nil {
				break
			}
		}
	}

	b.StopTimer()

	_, err = my.Start("drop table b")
	check(err)
	check(my.Close())
}
