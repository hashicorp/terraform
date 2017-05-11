package native

import (
	"github.com/ziutek/mymysql/mysql"
	"time"
)

// Integers

func DecodeU16(buf []byte) uint16 {
	return uint16(buf[1])<<8 | uint16(buf[0])
}
func (pr *pktReader) readU16() uint16 {
	buf := pr.buf[:2]
	pr.readFull(buf)
	return DecodeU16(buf)
}

func DecodeU24(buf []byte) uint32 {
	return (uint32(buf[2])<<8|uint32(buf[1]))<<8 | uint32(buf[0])
}
func (pr *pktReader) readU24() uint32 {
	buf := pr.buf[:3]
	pr.readFull(buf)
	return DecodeU24(buf)
}

func DecodeU32(buf []byte) uint32 {
	return ((uint32(buf[3])<<8|uint32(buf[2]))<<8|
		uint32(buf[1]))<<8 | uint32(buf[0])
}
func (pr *pktReader) readU32() uint32 {
	buf := pr.buf[:4]
	pr.readFull(buf)
	return DecodeU32(buf)
}

func DecodeU64(buf []byte) (rv uint64) {
	for ii, vv := range buf {
		rv |= uint64(vv) << uint(ii*8)
	}
	return
}
func (pr *pktReader) readU64() (rv uint64) {
	buf := pr.buf[:8]
	pr.readFull(buf)
	return DecodeU64(buf)
}

func EncodeU16(buf []byte, val uint16) {
	buf[0] = byte(val)
	buf[1] = byte(val >> 8)
}
func (pw *pktWriter) writeU16(val uint16) {
	buf := pw.buf[:2]
	EncodeU16(buf, val)
	pw.write(buf)
}

func EncodeU24(buf []byte, val uint32) {
	buf[0] = byte(val)
	buf[1] = byte(val >> 8)
	buf[2] = byte(val >> 16)
}
func (pw *pktWriter) writeU24(val uint32) {
	buf := pw.buf[:3]
	EncodeU24(buf, val)
	pw.write(buf)
}

func EncodeU32(buf []byte, val uint32) {
	buf[0] = byte(val)
	buf[1] = byte(val >> 8)
	buf[2] = byte(val >> 16)
	buf[3] = byte(val >> 24)
}
func (pw *pktWriter) writeU32(val uint32) {
	buf := pw.buf[:4]
	EncodeU32(buf, val)
	pw.write(buf)
}

func EncodeU64(buf []byte, val uint64) {
	buf[0] = byte(val)
	buf[1] = byte(val >> 8)
	buf[2] = byte(val >> 16)
	buf[3] = byte(val >> 24)
	buf[4] = byte(val >> 32)
	buf[5] = byte(val >> 40)
	buf[6] = byte(val >> 48)
	buf[7] = byte(val >> 56)
}
func (pw *pktWriter) writeU64(val uint64) {
	buf := pw.buf[:8]
	EncodeU64(buf, val)
	pw.write(buf)
}

// Variable length values

func (pr *pktReader) readNullLCB() (lcb uint64, null bool) {
	bb := pr.readByte()
	switch bb {
	case 251:
		null = true
	case 252:
		lcb = uint64(pr.readU16())
	case 253:
		lcb = uint64(pr.readU24())
	case 254:
		lcb = pr.readU64()
	default:
		lcb = uint64(bb)
	}
	return
}

func (pr *pktReader) readLCB() uint64 {
	lcb, null := pr.readNullLCB()
	if null {
		panic(mysql.ErrUnexpNullLCB)
	}
	return lcb
}

func (pw *pktWriter) writeLCB(val uint64) {
	switch {
	case val <= 250:
		pw.writeByte(byte(val))

	case val <= 0xffff:
		pw.writeByte(252)
		pw.writeU16(uint16(val))

	case val <= 0xffffff:
		pw.writeByte(253)
		pw.writeU24(uint32(val))

	default:
		pw.writeByte(254)
		pw.writeU64(val)
	}
}

func lenLCB(val uint64) int {
	switch {
	case val <= 250:
		return 1

	case val <= 0xffff:
		return 3

	case val <= 0xffffff:
		return 4
	}
	return 9
}

func (pr *pktReader) readNullBin() (buf []byte, null bool) {
	var l uint64
	l, null = pr.readNullLCB()
	if null {
		return
	}
	buf = make([]byte, l)
	pr.readFull(buf)
	return
}

func (pr *pktReader) readBin() []byte {
	buf, null := pr.readNullBin()
	if null {
		panic(mysql.ErrUnexpNullLCS)
	}
	return buf
}

func (pr *pktReader) skipBin() {
	n, _ := pr.readNullLCB()
	pr.skipN(int(n))
}

func (pw *pktWriter) writeBin(buf []byte) {
	pw.writeLCB(uint64(len(buf)))
	pw.write(buf)
}

func lenBin(buf []byte) int {
	return lenLCB(uint64(len(buf))) + len(buf)
}

func lenStr(str string) int {
	return lenLCB(uint64(len(str))) + len(str)
}

func (pw *pktWriter) writeLC(v interface{}) {
	switch val := v.(type) {
	case []byte:
		pw.writeBin(val)
	case *[]byte:
		pw.writeBin(*val)
	case string:
		pw.writeBin([]byte(val))
	case *string:
		pw.writeBin([]byte(*val))
	default:
		panic("Unknown data type for write as length coded string")
	}
}

func lenLC(v interface{}) int {
	switch val := v.(type) {
	case []byte:
		return lenBin(val)
	case *[]byte:
		return lenBin(*val)
	case string:
		return lenStr(val)
	case *string:
		return lenStr(*val)
	}
	panic("Unknown data type for write as length coded string")
}

func (pr *pktReader) readNTB() (buf []byte) {
	for {
		ch := pr.readByte()
		if ch == 0 {
			break
		}
		buf = append(buf, ch)
	}
	return
}

func (pw *pktWriter) writeNTB(buf []byte) {
	pw.write(buf)
	pw.writeByte(0)
}

func (pw *pktWriter) writeNT(v interface{}) {
	switch val := v.(type) {
	case []byte:
		pw.writeNTB(val)
	case string:
		pw.writeNTB([]byte(val))
	default:
		panic("Unknown type for write as null terminated data")
	}
}

// Date and time

func (pr *pktReader) readDuration() time.Duration {
	dlen := pr.readByte()
	switch dlen {
	case 251:
		// Null
		panic(mysql.ErrUnexpNullTime)
	case 0:
		// 00:00:00
		return 0
	case 5, 8, 12:
		// Properly time length
	default:
		panic(mysql.ErrWrongDateLen)
	}
	buf := pr.buf[:dlen]
	pr.readFull(buf)
	tt := int64(0)
	switch dlen {
	case 12:
		// Nanosecond part
		tt += int64(DecodeU32(buf[8:]))
		fallthrough
	case 8:
		// HH:MM:SS part
		tt += int64(int(buf[5])*3600+int(buf[6])*60+int(buf[7])) * 1e9
		fallthrough
	case 5:
		// Day part
		tt += int64(DecodeU32(buf[1:5])) * (24 * 3600 * 1e9)
	}
	if buf[0] != 0 {
		tt = -tt
	}
	return time.Duration(tt)
}

func EncodeDuration(buf []byte, d time.Duration) int {
	buf[0] = 0
	if d < 0 {
		buf[1] = 1
		d = -d
	}
	if ns := uint32(d % 1e9); ns != 0 {
		EncodeU32(buf[9:13], ns) // nanosecond
		buf[0] += 4
	}
	d /= 1e9
	if hms := int(d % (24 * 3600)); buf[0] != 0 || hms != 0 {
		buf[8] = byte(hms % 60) // second
		hms /= 60
		buf[7] = byte(hms % 60) // minute
		buf[6] = byte(hms / 60) // hour
		buf[0] += 3
	}
	if day := uint32(d / (24 * 3600)); buf[0] != 0 || day != 0 {
		EncodeU32(buf[2:6], day) // day
		buf[0] += 4
	}
	buf[0]++ // For sign byte
	return int(buf[0] + 1)
}

func (pw *pktWriter) writeDuration(d time.Duration) {
	buf := pw.buf[:13]
	n := EncodeDuration(buf, d)
	pw.write(buf[:n])
}

func lenDuration(d time.Duration) int {
	if d == 0 {
		return 2
	}
	if d%1e9 != 0 {
		return 13
	}
	d /= 1e9
	if d%(24*3600) != 0 {
		return 9
	}
	return 6
}

func (pr *pktReader) readTime() time.Time {
	dlen := pr.readByte()
	switch dlen {
	case 251:
		// Null
		panic(mysql.ErrUnexpNullDate)
	case 0:
		// return 0000-00-00 converted to time.Time zero
		return time.Time{}
	case 4, 7, 11:
		// Properly datetime length
	default:
		panic(mysql.ErrWrongDateLen)
	}

	buf := pr.buf[:dlen]
	pr.readFull(buf)
	var y, mon, d, h, m, s, u int
	switch dlen {
	case 11:
		// 2006-01-02 15:04:05.001004005
		u = int(DecodeU32(buf[7:]))
		fallthrough
	case 7:
		// 2006-01-02 15:04:05
		h = int(buf[4])
		m = int(buf[5])
		s = int(buf[6])
		fallthrough
	case 4:
		// 2006-01-02
		y = int(DecodeU16(buf[0:2]))
		mon = int(buf[2])
		d = int(buf[3])
	}
	n := u * int(time.Microsecond)
	return time.Date(y, time.Month(mon), d, h, m, s, n, time.Local)
}

func encodeNonzeroTime(buf []byte, y int16, mon, d, h, m, s byte, u uint32) int {
	buf[0] = 0
	switch {
	case u != 0:
		EncodeU32(buf[8:12], u)
		buf[0] += 4
		fallthrough
	case s != 0 || m != 0 || h != 0:
		buf[7] = s
		buf[6] = m
		buf[5] = h
		buf[0] += 3
	}
	buf[4] = d
	buf[3] = mon
	EncodeU16(buf[1:3], uint16(y))
	buf[0] += 4
	return int(buf[0] + 1)
}

func getTimeMicroseconds(t time.Time) int {
	return (t.Nanosecond() + int(time.Microsecond/2)) / int(time.Microsecond)
}

func EncodeTime(buf []byte, t time.Time) int {
	if t.IsZero() {
		// MySQL zero
		buf[0] = 0
		return 1 // MySQL zero
	}
	y, mon, d := t.Date()
	h, m, s := t.Clock()
	u:= getTimeMicroseconds(t)
	return encodeNonzeroTime(
		buf,
		int16(y), byte(mon), byte(d),
		byte(h), byte(m), byte(s), uint32(u),
	)
}

func (pw *pktWriter) writeTime(t time.Time) {
	buf := pw.buf[:12]
	n := EncodeTime(buf, t)
	pw.write(buf[:n])
}

func lenTime(t time.Time) int {
	switch {
	case t.IsZero():
		return 1
	case getTimeMicroseconds(t) != 0:
		return 12
	case t.Second() != 0 || t.Minute() != 0 || t.Hour() != 0:
		return 8
	}
	return 5
}

func (pr *pktReader) readDate() mysql.Date {
	y, m, d := pr.readTime().Date()
	return mysql.Date{int16(y), byte(m), byte(d)}
}

func EncodeDate(buf []byte, d mysql.Date) int {
	if d.IsZero() {
		// MySQL zero
		buf[0] = 0
		return 1
	}
	return encodeNonzeroTime(buf, d.Year, d.Month, d.Day, 0, 0, 0, 0)
}

func (pw *pktWriter) writeDate(d mysql.Date) {
	buf := pw.buf[:5]
	n := EncodeDate(buf, d)
	pw.write(buf[:n])
}

func lenDate(d mysql.Date) int {
	if d.IsZero() {
		return 1
	}
	return 5
}
