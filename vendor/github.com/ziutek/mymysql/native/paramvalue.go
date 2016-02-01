package native

import (
	"github.com/ziutek/mymysql/mysql"
	"math"
	"reflect"
	"time"
)

type paramValue struct {
	typ    uint16
	addr   reflect.Value
	raw    bool
	length int // >=0 - length of value, <0 - unknown length
}

func (val *paramValue) Len() int {
	if !val.addr.IsValid() {
		// Invalid Value was binded
		return 0
	}
	// val.addr always points to the pointer - lets dereference it
	v := val.addr.Elem()
	if v.IsNil() {
		// Binded Ptr Value is nil
		return 0
	}
	v = v.Elem()

	if val.length >= 0 {
		return val.length
	}

	switch val.typ {
	case MYSQL_TYPE_STRING:
		return lenStr(v.String())

	case MYSQL_TYPE_DATE:
		return lenDate(v.Interface().(mysql.Date))

	case MYSQL_TYPE_TIMESTAMP:
		return lenTime(v.Interface().(mysql.Timestamp).Time)
	case MYSQL_TYPE_DATETIME:
		return lenTime(v.Interface().(time.Time))

	case MYSQL_TYPE_TIME:
		return lenDuration(v.Interface().(time.Duration))

	case MYSQL_TYPE_TINY: // val.length < 0 so this is bool
		return 1
	}
	// MYSQL_TYPE_VAR_STRING, MYSQL_TYPE_BLOB and type of Raw value
	return lenBin(v.Bytes())
}

func (pw *pktWriter) writeValue(val *paramValue) {
	if !val.addr.IsValid() {
		// Invalid Value was binded
		return
	}
	// val.addr always points to the pointer - lets dereference it
	v := val.addr.Elem()
	if v.IsNil() {
		// Binded Ptr Value is nil
		return
	}
	v = v.Elem()

	if val.raw || val.typ == MYSQL_TYPE_VAR_STRING ||
		val.typ == MYSQL_TYPE_BLOB {
		pw.writeBin(v.Bytes())
		return
	}
	// We don't need unsigned bit to check type
	unsign := (val.typ & MYSQL_UNSIGNED_MASK) != 0
	switch val.typ & ^MYSQL_UNSIGNED_MASK {
	case MYSQL_TYPE_NULL:
		// Don't write null values

	case MYSQL_TYPE_STRING:
		pw.writeBin([]byte(v.String()))

	case MYSQL_TYPE_LONG:
		i := v.Interface()
		if unsign {
			l, ok := i.(uint32)
			if !ok {
				l = uint32(i.(uint))
			}
			pw.writeU32(l)
		} else {
			l, ok := i.(int32)
			if !ok {
				l = int32(i.(int))
			}
			pw.writeU32(uint32(l))
		}

	case MYSQL_TYPE_FLOAT:
		pw.writeU32(math.Float32bits(v.Interface().(float32)))

	case MYSQL_TYPE_SHORT:
		if unsign {
			pw.writeU16(v.Interface().(uint16))
		} else {
			pw.writeU16(uint16(v.Interface().(int16)))

		}

	case MYSQL_TYPE_TINY:
		if val.length == -1 {
			// Translate bool value to MySQL tiny
			if v.Bool() {
				pw.writeByte(1)
			} else {
				pw.writeByte(0)
			}
		} else {
			if unsign {
				pw.writeByte(v.Interface().(uint8))
			} else {
				pw.writeByte(uint8(v.Interface().(int8)))
			}
		}

	case MYSQL_TYPE_LONGLONG:
		i := v.Interface()
		if unsign {
			l, ok := i.(uint64)
			if !ok {
				l = uint64(i.(uint))
			}
			pw.writeU64(l)
		} else {
			l, ok := i.(int64)
			if !ok {
				l = int64(i.(int))
			}
			pw.writeU64(uint64(l))
		}

	case MYSQL_TYPE_DOUBLE:
		pw.writeU64(math.Float64bits(v.Interface().(float64)))

	case MYSQL_TYPE_DATE:
		pw.writeDate(v.Interface().(mysql.Date))

	case MYSQL_TYPE_TIMESTAMP:
		pw.writeTime(v.Interface().(mysql.Timestamp).Time)

	case MYSQL_TYPE_DATETIME:
		pw.writeTime(v.Interface().(time.Time))

	case MYSQL_TYPE_TIME:
		pw.writeDuration(v.Interface().(time.Duration))

	default:
		panic(mysql.ErrBindUnkType)
	}
	return
}
