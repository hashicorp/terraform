package native

import (
	"github.com/ziutek/mymysql/mysql"
	"reflect"
	"time"
)

var (
	timeType      = reflect.TypeOf(time.Time{})
	timestampType = reflect.TypeOf(mysql.Timestamp{})
	dateType      = reflect.TypeOf(mysql.Date{})
	durationType  = reflect.TypeOf(time.Duration(0))
	blobType      = reflect.TypeOf(mysql.Blob{})
	rawType       = reflect.TypeOf(mysql.Raw{})
)

// val should be an addressable value
func bindValue(val reflect.Value) (out paramValue) {
	if !val.IsValid() {
		out.typ = MYSQL_TYPE_NULL
		return
	}
	typ := val.Type()
	if typ.Kind() == reflect.Ptr {
		// We have addressable pointer
		out.addr = val.Addr()
		// Dereference pointer for next operation on its value
		typ = typ.Elem()
		val = val.Elem()
	} else {
		// We have addressable value. Create a pointer to it
		pv := val.Addr()
		// This pointer is unaddressable so copy it and return an address
		out.addr = reflect.New(pv.Type())
		out.addr.Elem().Set(pv)
	}

	// Obtain value type
	switch typ.Kind() {
	case reflect.String:
		out.typ = MYSQL_TYPE_STRING
		out.length = -1
		return

	case reflect.Int:
		out.typ = _INT_TYPE
		out.length = _SIZE_OF_INT
		return

	case reflect.Int8:
		out.typ = MYSQL_TYPE_TINY
		out.length = 1
		return

	case reflect.Int16:
		out.typ = MYSQL_TYPE_SHORT
		out.length = 2
		return

	case reflect.Int32:
		out.typ = MYSQL_TYPE_LONG
		out.length = 4
		return

	case reflect.Int64:
		if typ == durationType {
			out.typ = MYSQL_TYPE_TIME
			out.length = -1
			return
		}
		out.typ = MYSQL_TYPE_LONGLONG
		out.length = 8
		return

	case reflect.Uint:
		out.typ = _INT_TYPE | MYSQL_UNSIGNED_MASK
		out.length = _SIZE_OF_INT
		return

	case reflect.Uint8:
		out.typ = MYSQL_TYPE_TINY | MYSQL_UNSIGNED_MASK
		out.length = 1
		return

	case reflect.Uint16:
		out.typ = MYSQL_TYPE_SHORT | MYSQL_UNSIGNED_MASK
		out.length = 2
		return

	case reflect.Uint32:
		out.typ = MYSQL_TYPE_LONG | MYSQL_UNSIGNED_MASK
		out.length = 4
		return

	case reflect.Uint64:
		out.typ = MYSQL_TYPE_LONGLONG | MYSQL_UNSIGNED_MASK
		out.length = 8
		return

	case reflect.Float32:
		out.typ = MYSQL_TYPE_FLOAT
		out.length = 4
		return

	case reflect.Float64:
		out.typ = MYSQL_TYPE_DOUBLE
		out.length = 8
		return

	case reflect.Slice:
		out.length = -1
		if typ == blobType {
			out.typ = MYSQL_TYPE_BLOB
			return
		}
		if typ.Elem().Kind() == reflect.Uint8 {
			out.typ = MYSQL_TYPE_VAR_STRING
			return
		}

	case reflect.Struct:
		out.length = -1
		if typ == timeType {
			out.typ = MYSQL_TYPE_DATETIME
			return
		}
		if typ == dateType {
			out.typ = MYSQL_TYPE_DATE
			return
		}
		if typ == timestampType {
			out.typ = MYSQL_TYPE_TIMESTAMP
			return
		}
		if typ == rawType {
			out.typ = val.FieldByName("Typ").Interface().(uint16)
			out.addr = val.FieldByName("Val").Addr()
			out.raw = true
			return
		}

	case reflect.Bool:
		out.typ = MYSQL_TYPE_TINY
		// bool implementation isn't documented so we treat it in special way
		out.length = -1
		return
	}
	panic(mysql.ErrBindUnkType)
}
