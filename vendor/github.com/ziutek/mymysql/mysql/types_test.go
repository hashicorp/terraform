package mysql

import (
	"testing"
	"time"
)

type sio struct {
	in, out string
}

func checkRow(t *testing.T, examples []sio, conv func(string) interface{}) {
	row := make(Row, 1)
	for _, ex := range examples {
		row[0] = conv(ex.in)
		str := row.Str(0)
		if str != ex.out {
			t.Fatalf("Wrong conversion: '%s' != '%s'", str, ex.out)
		}
	}
}

var dates = []sio{
	sio{"2121-11-22", "2121-11-22"},
	sio{"0000-00-00", "0000-00-00"},
	sio{" 1234-12-18  ", "1234-12-18"},
	sio{"\t1234-12-18 \r\n", "1234-12-18"},
}

func TestConvDate(t *testing.T) {
	conv := func(str string) interface{} {
		d, err := ParseDate(str)
		if err != nil {
			return err
		}
		return d
	}
	checkRow(t, dates, conv)
}

var datetimes = []sio{
	sio{"2121-11-22 11:22:32", "2121-11-22 11:22:32"},
	sio{"  1234-12-18 22:11:22 ", "1234-12-18 22:11:22"},
	sio{"\t 1234-12-18 22:11:22 \r\n", "1234-12-18 22:11:22"},
	sio{"2000-11-11", "2000-11-11 00:00:00"},
	sio{"0000-00-00 00:00:00", "0000-00-00 00:00:00"},
	sio{"0000-00-00", "0000-00-00 00:00:00"},
	sio{"2000-11-22 11:11:11.000111222", "2000-11-22 11:11:11.000111222"},
}

func TestConvTime(t *testing.T) {
	conv := func(str string) interface{} {
		d, err := ParseTime(str, time.Local)
		if err != nil {
			return err
		}
		return d
	}
	checkRow(t, datetimes, conv)
}

var times = []sio{
	sio{"1:23:45", "1:23:45"},
	sio{"-112:23:45", "-112:23:45"},
	sio{"+112:23:45", "112:23:45"},
	sio{"1:60:00", "invalid MySQL TIME string: 1:60:00"},
	sio{"1:00:60", "invalid MySQL TIME string: 1:00:60"},
	sio{"1:23:45.000111333", "1:23:45.000111333"},
	sio{"-1:23:45.000111333", "-1:23:45.000111333"},
}

func TestConvDuration(t *testing.T) {
	conv := func(str string) interface{} {
		d, err := ParseDuration(str)
		if err != nil {
			return err
		}
		return d

	}
	checkRow(t, times, conv)
}

func TestEscapeString(t *testing.T) {
	txt := " \000 \n \r \\ ' \" \032 "
	exp := ` \0 \n \r \\ \' \" \Z `
	out := escapeString(txt)
	if out != exp {
		t.Fatalf("escapeString: ret='%s' exp='%s'", out, exp)
	}
}

func TestEscapeQuotes(t *testing.T) {
	txt := " '' '' ' ' ' "
	exp := ` '''' '''' '' '' '' `
	out := escapeQuotes(txt)
	if out != exp {
		t.Fatalf("escapeString: ret='%s' exp='%s'", out, exp)
	}
}
