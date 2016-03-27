package native

import (
	"bufio"
	"github.com/ziutek/mymysql/mysql"
	"io"
	"io/ioutil"
)

type pktReader struct {
	rd     *bufio.Reader
	seq    *byte
	remain int
	last   bool
	buf    [12]byte
	ibuf   [3]byte
}

func (my *Conn) newPktReader() *pktReader {
	return &pktReader{rd: my.rd, seq: &my.seq}
}

func (pr *pktReader) readHeader() {
	// Read next packet header
	buf := pr.ibuf[:]
	for {
		n, err := pr.rd.Read(buf)
		if err != nil {
			panic(err)
		}
		buf = buf[n:]
		if len(buf) == 0 {
			break
		}
	}
	pr.remain = int(DecodeU24(pr.ibuf[:]))
	seq, err := pr.rd.ReadByte()
	if err != nil {
		panic(err)
	}
	// Chceck sequence number
	if *pr.seq != seq {
		panic(mysql.ErrSeq)
	}
	*pr.seq++
	// Last packet?
	pr.last = (pr.remain != 0xffffff)
}

func (pr *pktReader) readFull(buf []byte) {
	for len(buf) > 0 {
		if pr.remain == 0 {
			if pr.last {
				// No more packets
				panic(io.EOF)
			}
			pr.readHeader()
		}
		n := len(buf)
		if n > pr.remain {
			n = pr.remain
		}
		n, err := pr.rd.Read(buf[:n])
		pr.remain -= n
		if err != nil {
			panic(err)
		}
		buf = buf[n:]
	}
	return
}

func (pr *pktReader) readByte() byte {
	if pr.remain == 0 {
		if pr.last {
			// No more packets
			panic(io.EOF)
		}
		pr.readHeader()
	}
	b, err := pr.rd.ReadByte()
	if err != nil {
		panic(err)
	}
	pr.remain--
	return b
}

func (pr *pktReader) readAll() (buf []byte) {
	m := 0
	for {
		if pr.remain == 0 {
			if pr.last {
				break
			}
			pr.readHeader()
		}
		new_buf := make([]byte, m+pr.remain)
		copy(new_buf, buf)
		buf = new_buf
		n, err := pr.rd.Read(buf[m:])
		pr.remain -= n
		m += n
		if err != nil {
			panic(err)
		}
	}
	return
}

func (pr *pktReader) skipAll() {
	for {
		if pr.remain == 0 {
			if pr.last {
				break
			}
			pr.readHeader()
		}
		n, err := io.CopyN(ioutil.Discard, pr.rd, int64(pr.remain))
		pr.remain -= int(n)
		if err != nil {
			panic(err)
		}
	}
	return
}

func (pr *pktReader) skipN(n int) {
	for n > 0 {
		if pr.remain == 0 {
			if pr.last {
				panic(io.EOF)
			}
			pr.readHeader()
		}
		m := int64(n)
		if n > pr.remain {
			m = int64(pr.remain)
		}
		m, err := io.CopyN(ioutil.Discard, pr.rd, m)
		pr.remain -= int(m)
		n -= int(m)
		if err != nil {
			panic(err)
		}
	}
	return
}

func (pr *pktReader) unreadByte() {
	if err := pr.rd.UnreadByte(); err != nil {
		panic(err)
	}
	pr.remain++
}

func (pr *pktReader) eof() bool {
	return pr.remain == 0 && pr.last
}

func (pr *pktReader) checkEof() {
	if !pr.eof() {
		panic(mysql.ErrPktLong)
	}
}

type pktWriter struct {
	wr       *bufio.Writer
	seq      *byte
	remain   int
	to_write int
	last     bool
	buf      [23]byte
	ibuf     [3]byte
}

func (my *Conn) newPktWriter(to_write int) *pktWriter {
	return &pktWriter{wr: my.wr, seq: &my.seq, to_write: to_write}
}

func (pw *pktWriter) writeHeader(l int) {
	buf := pw.ibuf[:]
	EncodeU24(buf, uint32(l))
	if _, err := pw.wr.Write(buf); err != nil {
		panic(err)
	}
	if err := pw.wr.WriteByte(*pw.seq); err != nil {
		panic(err)
	}
	// Update sequence number
	*pw.seq++
}

func (pw *pktWriter) write(buf []byte) {
	if len(buf) == 0 {
		return
	}
	var nn int
	for len(buf) != 0 {
		if pw.remain == 0 {
			if pw.to_write == 0 {
				panic("too many data for write as packet")
			}
			if pw.to_write >= 0xffffff {
				pw.remain = 0xffffff
			} else {
				pw.remain = pw.to_write
				pw.last = true
			}
			pw.to_write -= pw.remain
			pw.writeHeader(pw.remain)
		}
		nn = len(buf)
		if nn > pw.remain {
			nn = pw.remain
		}
		var err error
		nn, err = pw.wr.Write(buf[0:nn])
		pw.remain -= nn
		if err != nil {
			panic(err)
		}
		buf = buf[nn:]
	}
	if pw.remain+pw.to_write == 0 {
		if !pw.last {
			// Write  header for empty packet
			pw.writeHeader(0)
		}
		// Flush bufio buffers
		if err := pw.wr.Flush(); err != nil {
			panic(err)
		}
	}
	return
}

func (pw *pktWriter) writeByte(b byte) {
	pw.buf[0] = b
	pw.write(pw.buf[:1])
}

// n should be <= 23
func (pw *pktWriter) writeZeros(n int) {
	buf := pw.buf[:n]
	for i := range buf {
		buf[i] = 0
	}
	pw.write(buf)
}
