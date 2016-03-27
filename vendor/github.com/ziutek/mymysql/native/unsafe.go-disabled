package native

import (
	"github.com/ziutek/mymysql/mysql"
	"time"
	"unsafe"
)

type paramValue struct {
	typ    uint16
	addr   unsafe.Pointer
	raw    bool
	length int // >=0 - length of value, <0 - unknown length
}

func (pv *paramValue) SetAddr(addr uintptr) {
	pv.addr = unsafe.Pointer(addr)
}

func (val *paramValue) Len() int {
	if val.addr == nil {
		// Invalid Value was binded
		return 0
	}
	// val.addr always points to the pointer - lets dereference it
	ptr := *(*unsafe.Pointer)(val.addr)
	if ptr == nil {
		// Binded Ptr Value is nil
		return 0
	}

	if val.length >= 0 {
		return val.length
	}

	switch val.typ {
	case MYSQL_TYPE_STRING:
		return lenStr(*(*string)(ptr))

	case MYSQL_TYPE_DATE:
		return lenDate(*(*mysql.Date)(ptr))

	case MYSQL_TYPE_TIMESTAMP, MYSQL_TYPE_DATETIME:
		return lenTime(*(*time.Time)(ptr))

	case MYSQL_TYPE_TIME:
		return lenDuration(*(*time.Duration)(ptr))

	case MYSQL_TYPE_TINY: // val.length < 0 so this is bool
		return 1
	}
	// MYSQL_TYPE_VAR_STRING, MYSQL_TYPE_BLOB and type of Raw value
	return lenBin(*(*[]byte)(ptr))
}

func (pw *pktWriter) writeValue(val *paramValue) {
	if val.addr == nil {
		// Invalid Value was binded
		return
	}
	// val.addr always points to the pointer - lets dereference it
	ptr := *(*unsafe.Pointer)(val.addr)
	if ptr == nil {
		// Binded Ptr Value is nil
		return
	}

	if val.raw || val.typ == MYSQL_TYPE_VAR_STRING ||
		val.typ == MYSQL_TYPE_BLOB {
		pw.writeBin(*(*[]byte)(ptr))
		return
	}
	// We don't need unsigned bit to check type
	switch val.typ & ^MYSQL_UNSIGNED_MASK {
	case MYSQL_TYPE_NULL:
		// Don't write null values

	case MYSQL_TYPE_STRING:
		s := *(*string)(ptr)
		pw.writeBin([]byte(s))

	case MYSQL_TYPE_LONG, MYSQL_TYPE_FLOAT:
		pw.writeU32(*(*uint32)(ptr))

	case MYSQL_TYPE_SHORT:
		pw.writeU16(*(*uint16)(ptr))

	case MYSQL_TYPE_TINY:
		if val.length == -1 {
			// Translate bool value to MySQL tiny
			if *(*bool)(ptr) {
				pw.writeByte(1)
			} else {
				pw.writeByte(0)
			}
		} else {
			pw.writeByte(*(*byte)(ptr))
		}

	case MYSQL_TYPE_LONGLONG, MYSQL_TYPE_DOUBLE:
		pw.writeU64(*(*uint64)(ptr))

	case MYSQL_TYPE_DATE:
		pw.writeDate(*(*mysql.Date)(ptr))

	case MYSQL_TYPE_TIMESTAMP, MYSQL_TYPE_DATETIME:
		pw.writeTime(*(*time.Time)(ptr))

	case MYSQL_TYPE_TIME:
		pw.writeDuration(*(*time.Duration)(ptr))

	default:
		panic(mysql.ErrBindUnkType)
	}
	return
}
