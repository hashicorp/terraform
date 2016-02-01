package mysql

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"
)

// Version returns mymysql version string
func Version() string {
	return "1.5.3"
}

func syntaxError(ln int) error {
	return fmt.Errorf("syntax error at line: %d", ln)
}

// Creates new conneection handler using configuration in cfgFile. Returns
// connection handler and map contains unknown options.
//
// Config file format(example):
//
//	# mymysql options (if some option isn't specified it defaults to "")
//
//	DbRaddr	127.0.0.1:3306
//	# DbRaddr	/var/run/mysqld/mysqld.sock
//	DbUser	testuser
//	DbPass	TestPasswd9
//	# optional: DbName	test
//	# optional: DbEncd	utf8
//	# optional: DbLaddr	127.0.0.1:0
//	# optional: DbTimeout 15s
//
//	# Your options (returned in unk)
//
//	MyOpt	some text
func NewFromCF(cfgFile string) (con Conn, unk map[string]string, err error) {
	var cf *os.File
	cf, err = os.Open(cfgFile)
	if err != nil {
		return
	}
	br := bufio.NewReader(cf)
	um := make(map[string]string)
	var proto, laddr, raddr, user, pass, name, encd, to string
	for i := 1; ; i++ {
		buf, isPrefix, e := br.ReadLine()
		if e != nil {
			if e == io.EOF {
				break
			}
			err = e
			return
		}
		l := string(buf)
		if isPrefix {
			err = fmt.Errorf("line %d is too long", i)
			return
		}
		l = strings.TrimFunc(l, unicode.IsSpace)
		if len(l) == 0 || l[0] == '#' {
			continue
		}
		n := strings.IndexFunc(l, unicode.IsSpace)
		if n == -1 {
			err = fmt.Errorf("syntax error at line: %d", i)
			return
		}
		v := l[:n]
		l = strings.TrimLeftFunc(l[n:], unicode.IsSpace)
		switch v {
		case "DbLaddr":
			laddr = l
		case "DbRaddr":
			raddr = l
			proto = "tcp"
			if strings.IndexRune(l, ':') == -1 {
				proto = "unix"
			}
		case "DbUser":
			user = l
		case "DbPass":
			pass = l
		case "DbName":
			name = l
		case "DbEncd":
			encd = l
		case "DbTimeout":
			to = l
		default:
			um[v] = l
		}
	}
	if raddr == "" {
		err = errors.New("DbRaddr option is empty")
		return
	}
	unk = um
	if name != "" {
		con = New(proto, laddr, raddr, user, pass, name)
	} else {
		con = New(proto, laddr, raddr, user, pass)
	}
	if encd != "" {
		con.Register(fmt.Sprintf("SET NAMES %s", encd))
	}
	if to != "" {
		var timeout time.Duration
		timeout, err = time.ParseDuration(to)
		if err != nil {
			return
		}
		con.SetTimeout(timeout)
	}
	return
}

// Calls Start and next calls GetRow as long as it reads all rows from the
// result. Next it returns all readed rows as the slice of rows.
func Query(c Conn, sql string, params ...interface{}) (rows []Row, res Result, err error) {
	res, err = c.Start(sql, params...)
	if err != nil {
		return
	}
	rows, err = GetRows(res)
	return
}

// Calls Start and next calls GetFirstRow
func QueryFirst(c Conn, sql string, params ...interface{}) (row Row, res Result, err error) {
	res, err = c.Start(sql, params...)
	if err != nil {
		return
	}
	row, err = GetFirstRow(res)
	return
}

// Calls Start and next calls GetLastRow
func QueryLast(c Conn, sql string, params ...interface{}) (row Row, res Result, err error) {
	res, err = c.Start(sql, params...)
	if err != nil {
		return
	}
	row, err = GetLastRow(res)
	return
}

// Calls Run and next call GetRow as long as it reads all rows from the
// result. Next it returns all readed rows as the slice of rows.
func Exec(s Stmt, params ...interface{}) (rows []Row, res Result, err error) {
	res, err = s.Run(params...)
	if err != nil {
		return
	}
	rows, err = GetRows(res)
	return
}

// Calls Run and next call GetFirstRow
func ExecFirst(s Stmt, params ...interface{}) (row Row, res Result, err error) {
	res, err = s.Run(params...)
	if err != nil {
		return
	}
	row, err = GetFirstRow(res)
	return
}

// Calls Run and next call GetLastRow
func ExecLast(s Stmt, params ...interface{}) (row Row, res Result, err error) {
	res, err = s.Run(params...)
	if err != nil {
		return
	}
	row, err = GetLastRow(res)
	return
}

// Calls r.MakeRow and next r.ScanRow. Doesn't return io.EOF error (returns nil
// row insted).
func GetRow(r Result) (Row, error) {
	row := r.MakeRow()
	err := r.ScanRow(row)
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// Reads all rows from result and returns them as slice.
func GetRows(r Result) (rows []Row, err error) {
	var row Row
	for {
		row, err = r.GetRow()
		if err != nil || row == nil {
			break
		}
		rows = append(rows, row)
	}
	return
}

// Returns last row and discard others
func GetLastRow(r Result) (Row, error) {
	row := r.MakeRow()
	err := r.ScanRow(row)
	if err == io.EOF {
		return nil, nil
	}
	for err == nil {
		err = r.ScanRow(row)
	}
	if err == io.EOF {
		return row, nil
	}
	return nil, err
}

// Read all unreaded rows and discard them. This function is useful if you
// don't want to use the remaining rows. It has an impact only on current
// result. If there is multi result query, you must use NextResult method and
// read/discard all rows in this result, before use other method that sends
// data to the server. You can't use this function if last GetRow returned nil.
func End(r Result) error {
	_, err := GetLastRow(r)
	return err
}

// Returns first row and discard others
func GetFirstRow(r Result) (row Row, err error) {
	row, err = r.GetRow()
	if err == nil && row != nil {
		err = r.End()
	}
	return
}

func escapeString(txt string) string {
	var (
		esc string
		buf bytes.Buffer
	)
	last := 0
	for ii, bb := range txt {
		switch bb {
		case 0:
			esc = `\0`
		case '\n':
			esc = `\n`
		case '\r':
			esc = `\r`
		case '\\':
			esc = `\\`
		case '\'':
			esc = `\'`
		case '"':
			esc = `\"`
		case '\032':
			esc = `\Z`
		default:
			continue
		}
		io.WriteString(&buf, txt[last:ii])
		io.WriteString(&buf, esc)
		last = ii + 1
	}
	io.WriteString(&buf, txt[last:])
	return buf.String()
}

func escapeQuotes(txt string) string {
	var buf bytes.Buffer
	last := 0
	for ii, bb := range txt {
		if bb == '\'' {
			io.WriteString(&buf, txt[last:ii])
			io.WriteString(&buf, `''`)
			last = ii + 1
		}
	}
	io.WriteString(&buf, txt[last:])
	return buf.String()
}

// Escapes special characters in the txt, so it is safe to place returned string
// to Query method.
func Escape(c Conn, txt string) string {
	if c.Status()&SERVER_STATUS_NO_BACKSLASH_ESCAPES != 0 {
		return escapeQuotes(txt)
	}
	return escapeString(txt)
}
