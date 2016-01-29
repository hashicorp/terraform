package native

import (
	"github.com/ziutek/mymysql/mysql"
	"log"
)

type Stmt struct {
	my *Conn

	id  uint32
	sql string // For reprepare during reconnect

	params []paramValue // Parameters binding
	rebind bool
	binded bool

	fields []*mysql.Field

	field_count   int
	param_count   int
	warning_count int
	status        mysql.ConnStatus

	null_bitmap []byte
}

func (stmt *Stmt) Fields() []*mysql.Field {
	return stmt.fields
}

func (stmt *Stmt) NumParam() int {
	return stmt.param_count
}

func (stmt *Stmt) WarnCount() int {
	return stmt.warning_count
}

func (stmt *Stmt) sendCmdExec() {
	// Calculate packet length and NULL bitmap
	pkt_len := 1 + 4 + 1 + 4 + 1 + len(stmt.null_bitmap)
	for ii := range stmt.null_bitmap {
		stmt.null_bitmap[ii] = 0
	}
	for ii, param := range stmt.params {
		par_len := param.Len()
		pkt_len += par_len
		if par_len == 0 {
			null_byte := ii >> 3
			null_mask := byte(1) << uint(ii-(null_byte<<3))
			stmt.null_bitmap[null_byte] |= null_mask
		}
	}
	if stmt.rebind {
		pkt_len += stmt.param_count * 2
	}
	// Reset sequence number
	stmt.my.seq = 0
	// Packet sending
	pw := stmt.my.newPktWriter(pkt_len)
	pw.writeByte(_COM_STMT_EXECUTE)
	pw.writeU32(stmt.id)
	pw.writeByte(0) // flags = CURSOR_TYPE_NO_CURSOR
	pw.writeU32(1)  // iteration_count
	pw.write(stmt.null_bitmap)
	if stmt.rebind {
		pw.writeByte(1)
		// Types
		for _, param := range stmt.params {
			pw.writeU16(param.typ)
		}
	} else {
		pw.writeByte(0)
	}
	// Values
	for i := range stmt.params {
		pw.writeValue(&stmt.params[i])
	}

	if stmt.my.Debug {
		log.Printf("[%2d <-] Exec command packet: len=%d, null_bitmap=%v, rebind=%t",
			stmt.my.seq-1, pkt_len, stmt.null_bitmap, stmt.rebind)
	}

	// Mark that we sended information about binded types
	stmt.rebind = false
}

func (my *Conn) getPrepareResult(stmt *Stmt) interface{} {
loop:
	pr := my.newPktReader() // New reader for next packet
	pkt0 := pr.readByte()

	//log.Println("pkt0:", pkt0, "stmt:", stmt)

	if pkt0 == 255 {
		// Error packet
		my.getErrorPacket(pr)
	}

	if stmt == nil {
		if pkt0 == 0 {
			// OK packet
			return my.getPrepareOkPacket(pr)
		}
	} else {
		unreaded_params := (stmt.param_count < len(stmt.params))
		switch {
		case pkt0 == 254:
			// EOF packet
			stmt.warning_count, stmt.status = my.getEofPacket(pr)
			stmt.my.status = stmt.status
			return stmt

		case pkt0 > 0 && pkt0 < 251 && (stmt.field_count < len(stmt.fields) ||
			unreaded_params):
			// Field packet
			if unreaded_params {
				// Read and ignore parameter field. Sentence from MySQL source:
				/* skip parameters data: we don't support it yet */
				pr.skipAll()
				// Increment param_count count
				stmt.param_count++
			} else {
				field := my.getFieldPacket(pr)
				stmt.fields[stmt.field_count] = field
				// Increment field count
				stmt.field_count++
			}
			// Read next packet
			goto loop
		}
	}
	panic(mysql.ErrUnkResultPkt)
}

func (my *Conn) getPrepareOkPacket(pr *pktReader) (stmt *Stmt) {
	if my.Debug {
		log.Printf("[%2d ->] Perpared OK packet:", my.seq-1)
	}

	stmt = new(Stmt)
	stmt.my = my
	// First byte was readed by getPrepRes
	stmt.id = pr.readU32()
	stmt.fields = make([]*mysql.Field, int(pr.readU16())) // FieldCount
	pl := int(pr.readU16())                               // ParamCount
	if pl > 0 {
		stmt.params = make([]paramValue, pl)
		stmt.null_bitmap = make([]byte, (pl+7)>>3)
	}
	pr.skipN(1)
	stmt.warning_count = int(pr.readU16())
	pr.checkEof()

	if my.Debug {
		log.Printf(tab8s+"ID=0x%x ParamCount=%d FieldsCount=%d WarnCount=%d",
			stmt.id, len(stmt.params), len(stmt.fields), stmt.warning_count,
		)
	}
	return
}
