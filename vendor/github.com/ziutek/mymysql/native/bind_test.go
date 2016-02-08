package native

import (
	"bufio"
	"bytes"
	"github.com/ziutek/mymysql/mysql"
	"math"
	"reflect"
	"strconv"
	"testing"
	"time"
)

var (
	Bytes  = []byte("Ala ma Kota!")
	String = "ssss" //"A kot ma Alę!"
	blob   = mysql.Blob{1, 2, 3}
	dateT  = time.Date(2010, 12, 30, 17, 21, 01, 0, time.Local)
	tstamp = mysql.Timestamp{dateT.Add(1e9)}
	date   = mysql.Date{Year: 2011, Month: 2, Day: 3}
	tim    = -time.Duration((5*24*3600+4*3600+3*60+2)*1e9 + 1)
	bol    = true

	pBytes  *[]byte
	pString *string
	pBlob   *mysql.Blob
	pDateT  *time.Time
	pTstamp *mysql.Timestamp
	pDate   *mysql.Date
	pTim    *time.Duration
	pBol    *bool

	raw = mysql.Raw{MYSQL_TYPE_INT24, &[]byte{3, 2, 1, 0}}

	Int8   = int8(1)
	Uint8  = uint8(2)
	Int16  = int16(3)
	Uint16 = uint16(4)
	Int32  = int32(5)
	Uint32 = uint32(6)
	Int64  = int64(0x7000100020003001)
	Uint64 = uint64(0xffff0000ffff0000)
	Int    = int(7)
	Uint   = uint(8)

	Float32 = float32(1e10)
	Float64 = 256e256

	pInt8    *int8
	pUint8   *uint8
	pInt16   *int16
	pUint16  *uint16
	pInt32   *int32
	pUint32  *uint32
	pInt64   *int64
	pUint64  *uint64
	pInt     *int
	pUint    *uint
	pFloat32 *float32
	pFloat64 *float64
)

type BindTest struct {
	val    interface{}
	typ    uint16
	length int
}

func intSize() int {
	switch strconv.IntSize {
	case 32:
		return 4
	case 64:
		return 8
	}
	panic("bad int size")
}

func intType() uint16 {
	switch strconv.IntSize {
	case 32:
		return MYSQL_TYPE_LONG
	case 64:
		return MYSQL_TYPE_LONGLONG
	}
	panic("bad int size")

}

var bindTests = []BindTest{
	BindTest{nil, MYSQL_TYPE_NULL, 0},

	BindTest{Bytes, MYSQL_TYPE_VAR_STRING, -1},
	BindTest{String, MYSQL_TYPE_STRING, -1},
	BindTest{blob, MYSQL_TYPE_BLOB, -1},
	BindTest{dateT, MYSQL_TYPE_DATETIME, -1},
	BindTest{tstamp, MYSQL_TYPE_TIMESTAMP, -1},
	BindTest{date, MYSQL_TYPE_DATE, -1},
	BindTest{tim, MYSQL_TYPE_TIME, -1},
	BindTest{bol, MYSQL_TYPE_TINY, -1},

	BindTest{&Bytes, MYSQL_TYPE_VAR_STRING, -1},
	BindTest{&String, MYSQL_TYPE_STRING, -1},
	BindTest{&blob, MYSQL_TYPE_BLOB, -1},
	BindTest{&dateT, MYSQL_TYPE_DATETIME, -1},
	BindTest{&tstamp, MYSQL_TYPE_TIMESTAMP, -1},
	BindTest{&date, MYSQL_TYPE_DATE, -1},
	BindTest{&tim, MYSQL_TYPE_TIME, -1},

	BindTest{pBytes, MYSQL_TYPE_VAR_STRING, -1},
	BindTest{pString, MYSQL_TYPE_STRING, -1},
	BindTest{pBlob, MYSQL_TYPE_BLOB, -1},
	BindTest{pDateT, MYSQL_TYPE_DATETIME, -1},
	BindTest{pTstamp, MYSQL_TYPE_TIMESTAMP, -1},
	BindTest{pDate, MYSQL_TYPE_DATE, -1},
	BindTest{pTim, MYSQL_TYPE_TIME, -1},
	BindTest{pBol, MYSQL_TYPE_TINY, -1},

	BindTest{raw, MYSQL_TYPE_INT24, -1},

	BindTest{Int8, MYSQL_TYPE_TINY, 1},
	BindTest{Int16, MYSQL_TYPE_SHORT, 2},
	BindTest{Int32, MYSQL_TYPE_LONG, 4},
	BindTest{Int64, MYSQL_TYPE_LONGLONG, 8},
	BindTest{Int, intType(), intSize()},

	BindTest{&Int8, MYSQL_TYPE_TINY, 1},
	BindTest{&Int16, MYSQL_TYPE_SHORT, 2},
	BindTest{&Int32, MYSQL_TYPE_LONG, 4},
	BindTest{&Int64, MYSQL_TYPE_LONGLONG, 8},
	BindTest{&Int, intType(), intSize()},

	BindTest{pInt8, MYSQL_TYPE_TINY, 1},
	BindTest{pInt16, MYSQL_TYPE_SHORT, 2},
	BindTest{pInt32, MYSQL_TYPE_LONG, 4},
	BindTest{pInt64, MYSQL_TYPE_LONGLONG, 8},
	BindTest{pInt, intType(), intSize()},

	BindTest{Uint8, MYSQL_TYPE_TINY | MYSQL_UNSIGNED_MASK, 1},
	BindTest{Uint16, MYSQL_TYPE_SHORT | MYSQL_UNSIGNED_MASK, 2},
	BindTest{Uint32, MYSQL_TYPE_LONG | MYSQL_UNSIGNED_MASK, 4},
	BindTest{Uint64, MYSQL_TYPE_LONGLONG | MYSQL_UNSIGNED_MASK, 8},
	BindTest{Uint, intType() | MYSQL_UNSIGNED_MASK, intSize()},

	BindTest{&Uint8, MYSQL_TYPE_TINY | MYSQL_UNSIGNED_MASK, 1},
	BindTest{&Uint16, MYSQL_TYPE_SHORT | MYSQL_UNSIGNED_MASK, 2},
	BindTest{&Uint32, MYSQL_TYPE_LONG | MYSQL_UNSIGNED_MASK, 4},
	BindTest{&Uint64, MYSQL_TYPE_LONGLONG | MYSQL_UNSIGNED_MASK, 8},
	BindTest{&Uint, intType() | MYSQL_UNSIGNED_MASK, intSize()},

	BindTest{pUint8, MYSQL_TYPE_TINY | MYSQL_UNSIGNED_MASK, 1},
	BindTest{pUint16, MYSQL_TYPE_SHORT | MYSQL_UNSIGNED_MASK, 2},
	BindTest{pUint32, MYSQL_TYPE_LONG | MYSQL_UNSIGNED_MASK, 4},
	BindTest{pUint64, MYSQL_TYPE_LONGLONG | MYSQL_UNSIGNED_MASK, 8},
	BindTest{pUint, intType() | MYSQL_UNSIGNED_MASK, intSize()},

	BindTest{Float32, MYSQL_TYPE_FLOAT, 4},
	BindTest{Float64, MYSQL_TYPE_DOUBLE, 8},

	BindTest{&Float32, MYSQL_TYPE_FLOAT, 4},
	BindTest{&Float64, MYSQL_TYPE_DOUBLE, 8},
}

func makeAddressable(v reflect.Value) reflect.Value {
	if v.IsValid() {
		// Make an addresable value
		av := reflect.New(v.Type()).Elem()
		av.Set(v)
		v = av
	}
	return v
}

func TestBind(t *testing.T) {
	for _, test := range bindTests {
		v := makeAddressable(reflect.ValueOf(test.val))
		val := bindValue(v)
		if val.typ != test.typ || val.length != test.length {
			t.Errorf(
				"Type: %s exp=0x%x res=0x%x Len: exp=%d res=%d",
				reflect.TypeOf(test.val), test.typ, val.typ, test.length,
				val.length,
			)
		}
	}
}

type WriteTest struct {
	val interface{}
	exp []byte
}

var writeTest []WriteTest

func encodeU16(v uint16) []byte {
	buf := make([]byte, 2)
	EncodeU16(buf, v)
	return buf
}

func encodeU24(v uint32) []byte {
	buf := make([]byte, 3)
	EncodeU24(buf, v)
	return buf
}

func encodeU32(v uint32) []byte {
	buf := make([]byte, 4)
	EncodeU32(buf, v)
	return buf
}

func encodeU64(v uint64) []byte {
	buf := make([]byte, 8)
	EncodeU64(buf, v)
	return buf
}

func encodeDuration(d time.Duration) []byte {
	buf := make([]byte, 13)
	n := EncodeDuration(buf, d)
	return buf[:n]
}

func encodeTime(t time.Time) []byte {
	buf := make([]byte, 12)
	n := EncodeTime(buf, t)
	return buf[:n]
}

func encodeDate(d mysql.Date) []byte {
	buf := make([]byte, 5)
	n := EncodeDate(buf, d)
	return buf[:n]
}

func encodeUint(u uint) []byte {
	switch strconv.IntSize {
	case 32:
		return encodeU32(uint32(u))
	case 64:
		return encodeU64(uint64(u))
	}
	panic("bad int size")

}

func init() {
	b := make([]byte, 64*1024)
	for ii := range b {
		b[ii] = byte(ii)
	}
	blob = mysql.Blob(b)

	writeTest = []WriteTest{
		WriteTest{Bytes, append([]byte{byte(len(Bytes))}, Bytes...)},
		WriteTest{String, append([]byte{byte(len(String))}, []byte(String)...)},
		WriteTest{pBytes, nil},
		WriteTest{pString, nil},
		WriteTest{
			blob,
			append(
				append([]byte{253}, byte(len(blob)), byte(len(blob)>>8), byte(len(blob)>>16)),
				[]byte(blob)...),
		},
		WriteTest{
			dateT,
			[]byte{
				7, byte(dateT.Year()), byte(dateT.Year() >> 8),
				byte(dateT.Month()),
				byte(dateT.Day()), byte(dateT.Hour()), byte(dateT.Minute()),
				byte(dateT.Second()),
			},
		},
		WriteTest{
			&dateT,
			[]byte{
				7, byte(dateT.Year()), byte(dateT.Year() >> 8),
				byte(dateT.Month()),
				byte(dateT.Day()), byte(dateT.Hour()), byte(dateT.Minute()),
				byte(dateT.Second()),
			},
		},
		WriteTest{
			date,
			[]byte{
				4, byte(date.Year), byte(date.Year >> 8), byte(date.Month),
				byte(date.Day),
			},
		},
		WriteTest{
			&date,
			[]byte{
				4, byte(date.Year), byte(date.Year >> 8), byte(date.Month),
				byte(date.Day),
			},
		},
		WriteTest{
			tim,
			[]byte{12, 1, 5, 0, 0, 0, 4, 3, 2, 1, 0, 0, 0},
		},
		WriteTest{
			&tim,
			[]byte{12, 1, 5, 0, 0, 0, 4, 3, 2, 1, 0, 0, 0},
		},
		WriteTest{bol, []byte{1}},
		WriteTest{&bol, []byte{1}},
		WriteTest{pBol, nil},

		WriteTest{dateT, encodeTime(dateT)},
		WriteTest{&dateT, encodeTime(dateT)},
		WriteTest{pDateT, nil},

		WriteTest{tstamp, encodeTime(tstamp.Time)},
		WriteTest{&tstamp, encodeTime(tstamp.Time)},
		WriteTest{pTstamp, nil},

		WriteTest{date, encodeDate(date)},
		WriteTest{&date, encodeDate(date)},
		WriteTest{pDate, nil},

		WriteTest{tim, encodeDuration(tim)},
		WriteTest{&tim, encodeDuration(tim)},
		WriteTest{pTim, nil},

		WriteTest{Int, encodeUint(uint(Int))},
		WriteTest{Int16, encodeU16(uint16(Int16))},
		WriteTest{Int32, encodeU32(uint32(Int32))},
		WriteTest{Int64, encodeU64(uint64(Int64))},

		WriteTest{Uint, encodeUint(Uint)},
		WriteTest{Uint16, encodeU16(Uint16)},
		WriteTest{Uint32, encodeU32(Uint32)},
		WriteTest{Uint64, encodeU64(Uint64)},

		WriteTest{&Int, encodeUint(uint(Int))},
		WriteTest{&Int16, encodeU16(uint16(Int16))},
		WriteTest{&Int32, encodeU32(uint32(Int32))},
		WriteTest{&Int64, encodeU64(uint64(Int64))},

		WriteTest{&Uint, encodeUint(Uint)},
		WriteTest{&Uint16, encodeU16(Uint16)},
		WriteTest{&Uint32, encodeU32(Uint32)},
		WriteTest{&Uint64, encodeU64(Uint64)},

		WriteTest{pInt, nil},
		WriteTest{pInt16, nil},
		WriteTest{pInt32, nil},
		WriteTest{pInt64, nil},

		WriteTest{Float32, encodeU32(math.Float32bits(Float32))},
		WriteTest{Float64, encodeU64(math.Float64bits(Float64))},

		WriteTest{&Float32, encodeU32(math.Float32bits(Float32))},
		WriteTest{&Float64, encodeU64(math.Float64bits(Float64))},

		WriteTest{pFloat32, nil},
		WriteTest{pFloat64, nil},
	}
}

func TestWrite(t *testing.T) {
	buf := new(bytes.Buffer)
	for _, test := range writeTest {
		buf.Reset()
		var seq byte
		pw := &pktWriter{
			wr:       bufio.NewWriter(buf),
			seq:      &seq,
			to_write: len(test.exp),
		}
		v := makeAddressable(reflect.ValueOf(test.val))
		val := bindValue(v)
		pw.writeValue(&val)
		if !reflect.Indirect(v).IsValid() && len(buf.Bytes()) == 0 {
			// writeValue writes nothing for nil
			continue
		}
		if len(buf.Bytes()) != len(test.exp)+4 || !bytes.Equal(buf.Bytes()[4:], test.exp) || val.Len() != len(test.exp) {
			t.Fatalf("%s - exp_len=%d res_len=%d exp: %v res: %v",
				reflect.TypeOf(test.val), len(test.exp), val.Len(),
				test.exp, buf.Bytes(),
			)
		}
	}
}
