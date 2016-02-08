package native

import (
	"errors"
	"github.com/ziutek/mymysql/mysql"
	"log"
	"math"
	"strconv"
)

type Result struct {
	my          *Conn
	status_only bool // true if result doesn't contain result set
	binary      bool // Binary result expected

	field_count int
	fields      []*mysql.Field // Fields table
	fc_map      map[string]int // Maps field name to column number

	message       []byte
	affected_rows uint64

	// Primary key value (useful for AUTO_INCREMENT primary keys)
	insert_id uint64

	// Number of warinigs during command execution
	// You can use the SHOW WARNINGS query for details.
	warning_count int

	// MySQL server status immediately after the query execution
	status mysql.ConnStatus

	// Seted by GetRow if it returns nil row
	eor_returned bool
}

// Returns true if this is status result that includes no result set
func (res *Result) StatusOnly() bool {
	return res.status_only
}

// Returns a table containing descriptions of the columns
func (res *Result) Fields() []*mysql.Field {
	return res.fields
}

// Returns index for given name or -1 if field of that name doesn't exist
func (res *Result) Map(field_name string) int {
	if fi, ok := res.fc_map[field_name]; ok {
		return fi
	}
	return -1
}

func (res *Result) Message() string {
	return string(res.message)
}

func (res *Result) AffectedRows() uint64 {
	return res.affected_rows
}

func (res *Result) InsertId() uint64 {
	return res.insert_id
}

func (res *Result) WarnCount() int {
	return res.warning_count
}

func (res *Result) MakeRow() mysql.Row {
	return make(mysql.Row, res.field_count)
}

func (my *Conn) getResult(res *Result, row mysql.Row) *Result {
loop:
	pr := my.newPktReader() // New reader for next packet
	pkt0 := pr.readByte()

	if pkt0 == 255 {
		// Error packet
		my.getErrorPacket(pr)
	}

	if res == nil {
		switch {
		case pkt0 == 0:
			// OK packet
			return my.getOkPacket(pr)

		case pkt0 > 0 && pkt0 < 251:
			// Result set header packet
			res = my.getResSetHeadPacket(pr)
			// Read next packet
			goto loop
		case pkt0 == 251:
			// Load infile response
			// Handle response
			goto loop
		case pkt0 == 254:
			// EOF packet (without body)
			return nil
		}
	} else {
		switch {
		case pkt0 == 254:
			// EOF packet
			res.warning_count, res.status = my.getEofPacket(pr)
			my.status = res.status
			return res

		case pkt0 > 0 && pkt0 < 251 && res.field_count < len(res.fields):
			// Field packet
			field := my.getFieldPacket(pr)
			res.fields[res.field_count] = field
			res.fc_map[field.Name] = res.field_count
			// Increment field count
			res.field_count++
			// Read next packet
			goto loop

		case pkt0 < 254 && res.field_count == len(res.fields):
			// Row Data Packet
			if len(row) != res.field_count {
				panic(mysql.ErrRowLength)
			}
			if res.binary {
				my.getBinRowPacket(pr, res, row)
			} else {
				my.getTextRowPacket(pr, res, row)
			}
			return nil
		}
	}
	panic(mysql.ErrUnkResultPkt)
}

func (my *Conn) getOkPacket(pr *pktReader) (res *Result) {
	if my.Debug {
		log.Printf("[%2d ->] OK packet:", my.seq-1)
	}
	res = new(Result)
	res.status_only = true
	res.my = my
	// First byte was readed by getResult
	res.affected_rows = pr.readLCB()
	res.insert_id = pr.readLCB()
	res.status = mysql.ConnStatus(pr.readU16())
	my.status = res.status
	res.warning_count = int(pr.readU16())
	res.message = pr.readAll()
	pr.checkEof()

	if my.Debug {
		log.Printf(tab8s+"AffectedRows=%d InsertId=0x%x Status=0x%x "+
			"WarningCount=%d Message=\"%s\"", res.affected_rows, res.insert_id,
			res.status, res.warning_count, res.message,
		)
	}
	return
}

func (my *Conn) getErrorPacket(pr *pktReader) {
	if my.Debug {
		log.Printf("[%2d ->] Error packet:", my.seq-1)
	}
	var err mysql.Error
	err.Code = pr.readU16()
	if pr.readByte() != '#' {
		panic(mysql.ErrPkt)
	}
	pr.skipN(5)
	err.Msg = pr.readAll()
	pr.checkEof()

	if my.Debug {
		log.Printf(tab8s+"code=0x%x msg=\"%s\"", err.Code, err.Msg)
	}
	panic(&err)
}

func (my *Conn) getEofPacket(pr *pktReader) (warn_count int, status mysql.ConnStatus) {
	if my.Debug {
		if pr.eof() {
			log.Printf("[%2d ->] EOF packet without body", my.seq-1)
		} else {
			log.Printf("[%2d ->] EOF packet:", my.seq-1)
		}
	}
	if pr.eof() {
		return
	}
	warn_count = int(pr.readU16())
	if pr.eof() {
		return
	}
	status = mysql.ConnStatus(pr.readU16())
	pr.checkEof()

	if my.Debug {
		log.Printf(tab8s+"WarningCount=%d Status=0x%x", warn_count, status)
	}
	return
}

func (my *Conn) getResSetHeadPacket(pr *pktReader) (res *Result) {
	if my.Debug {
		log.Printf("[%2d ->] Result set header packet:", my.seq-1)
	}
	pr.unreadByte()

	field_count := int(pr.readLCB())
	pr.checkEof()

	res = &Result{
		my:     my,
		fields: make([]*mysql.Field, field_count),
		fc_map: make(map[string]int),
	}

	if my.Debug {
		log.Printf(tab8s+"FieldCount=%d", field_count)
	}
	return
}

func (my *Conn) getFieldPacket(pr *pktReader) (field *mysql.Field) {
	if my.Debug {
		log.Printf("[%2d ->] Field packet:", my.seq-1)
	}
	pr.unreadByte()

	field = new(mysql.Field)
	if my.fullFieldInfo {
		field.Catalog = string(pr.readBin())
		field.Db = string(pr.readBin())
		field.Table = string(pr.readBin())
		field.OrgTable = string(pr.readBin())
	} else {
		pr.skipBin()
		pr.skipBin()
		pr.skipBin()
		pr.skipBin()
	}
	field.Name = string(pr.readBin())
	if my.fullFieldInfo {
		field.OrgName = string(pr.readBin())
	} else {
		pr.skipBin()
	}
	pr.skipN(1 + 2)
	//field.Charset= pr.readU16()
	field.DispLen = pr.readU32()
	field.Type = pr.readByte()
	field.Flags = pr.readU16()
	field.Scale = pr.readByte()
	pr.skipN(2)
	pr.checkEof()

	if my.Debug {
		log.Printf(tab8s+"Name=\"%s\" Type=0x%x", field.Name, field.Type)
	}
	return
}

func (my *Conn) getTextRowPacket(pr *pktReader, res *Result, row mysql.Row) {
	if my.Debug {
		log.Printf("[%2d ->] Text row data packet", my.seq-1)
	}
	pr.unreadByte()

	for ii := 0; ii < res.field_count; ii++ {
		bin, null := pr.readNullBin()
		if null {
			row[ii] = nil
		} else {
			row[ii] = bin
		}
	}
	pr.checkEof()
}

func (my *Conn) getBinRowPacket(pr *pktReader, res *Result, row mysql.Row) {
	if my.Debug {
		log.Printf("[%2d ->] Binary row data packet", my.seq-1)
	}
	// First byte was readed by getResult

	null_bitmap := make([]byte, (res.field_count+7+2)>>3)
	pr.readFull(null_bitmap)

	for ii, field := range res.fields {
		null_byte := (ii + 2) >> 3
		null_mask := byte(1) << uint(2+ii-(null_byte<<3))
		if null_bitmap[null_byte]&null_mask != 0 {
			// Null field
			row[ii] = nil
			continue
		}
		unsigned := (field.Flags & _FLAG_UNSIGNED) != 0
		if my.narrowTypeSet {
			row[ii] = readValueNarrow(pr, field.Type, unsigned)
		} else {
			row[ii] = readValue(pr, field.Type, unsigned)
		}
	}
}

func readValue(pr *pktReader, typ byte, unsigned bool) interface{} {
	switch typ {
	case MYSQL_TYPE_STRING, MYSQL_TYPE_VAR_STRING, MYSQL_TYPE_VARCHAR,
		MYSQL_TYPE_BIT, MYSQL_TYPE_BLOB, MYSQL_TYPE_TINY_BLOB,
		MYSQL_TYPE_MEDIUM_BLOB, MYSQL_TYPE_LONG_BLOB, MYSQL_TYPE_SET,
		MYSQL_TYPE_ENUM, MYSQL_TYPE_GEOMETRY:
		return pr.readBin()
	case MYSQL_TYPE_TINY:
		if unsigned {
			return pr.readByte()
		} else {
			return int8(pr.readByte())
		}
	case MYSQL_TYPE_SHORT, MYSQL_TYPE_YEAR:
		if unsigned {
			return pr.readU16()
		} else {
			return int16(pr.readU16())
		}
	case MYSQL_TYPE_LONG, MYSQL_TYPE_INT24:
		if unsigned {
			return pr.readU32()
		} else {
			return int32(pr.readU32())
		}
	case MYSQL_TYPE_LONGLONG:
		if unsigned {
			return pr.readU64()
		} else {
			return int64(pr.readU64())
		}
	case MYSQL_TYPE_FLOAT:
		return math.Float32frombits(pr.readU32())
	case MYSQL_TYPE_DOUBLE:
		return math.Float64frombits(pr.readU64())
	case MYSQL_TYPE_DECIMAL, MYSQL_TYPE_NEWDECIMAL:
		dec := string(pr.readBin())
		r, err := strconv.ParseFloat(dec, 64)
		if err != nil {
			panic(errors.New("MySQL server returned wrong decimal value: " + dec))
		}
		return r
	case MYSQL_TYPE_DATE, MYSQL_TYPE_NEWDATE:
		return pr.readDate()
	case MYSQL_TYPE_DATETIME, MYSQL_TYPE_TIMESTAMP:
		return pr.readTime()
	case MYSQL_TYPE_TIME:
		return pr.readDuration()
	}
	panic(mysql.ErrUnkMySQLType)
}

func readValueNarrow(pr *pktReader, typ byte, unsigned bool) interface{} {
	switch typ {
	case MYSQL_TYPE_STRING, MYSQL_TYPE_VAR_STRING, MYSQL_TYPE_VARCHAR,
		MYSQL_TYPE_BIT, MYSQL_TYPE_BLOB, MYSQL_TYPE_TINY_BLOB,
		MYSQL_TYPE_MEDIUM_BLOB, MYSQL_TYPE_LONG_BLOB, MYSQL_TYPE_SET,
		MYSQL_TYPE_ENUM, MYSQL_TYPE_GEOMETRY:
		return pr.readBin()
	case MYSQL_TYPE_TINY:
		if unsigned {
			return int64(pr.readByte())
		}
		return int64(int8(pr.readByte()))
	case MYSQL_TYPE_SHORT, MYSQL_TYPE_YEAR:
		if unsigned {
			return int64(pr.readU16())
		}
		return int64(int16(pr.readU16()))
	case MYSQL_TYPE_LONG, MYSQL_TYPE_INT24:
		if unsigned {
			return int64(pr.readU32())
		}
		return int64(int32(pr.readU32()))
	case MYSQL_TYPE_LONGLONG:
		v := pr.readU64()
		if unsigned && v > math.MaxInt64 {
			panic(errors.New("Value to large for int64 type"))
		}
		return int64(v)
	case MYSQL_TYPE_FLOAT:
		return float64(math.Float32frombits(pr.readU32()))
	case MYSQL_TYPE_DOUBLE:
		return math.Float64frombits(pr.readU64())
	case MYSQL_TYPE_DECIMAL, MYSQL_TYPE_NEWDECIMAL:
		dec := string(pr.readBin())
		r, err := strconv.ParseFloat(dec, 64)
		if err != nil {
			panic("MySQL server returned wrong decimal value: " + dec)
		}
		return r
	case MYSQL_TYPE_DATETIME, MYSQL_TYPE_TIMESTAMP, MYSQL_TYPE_DATE, MYSQL_TYPE_NEWDATE:
		return pr.readTime()
	case MYSQL_TYPE_TIME:
		return int64(pr.readDuration())
	}
	panic(mysql.ErrUnkMySQLType)
}
