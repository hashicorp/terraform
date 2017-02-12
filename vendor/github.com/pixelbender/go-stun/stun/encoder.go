package stun

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"hash/crc32"
)

// An Encoder writes STUN message to a buffer.
type Encoder struct {
	*Config
	w writer
}

func NewEncoder(config *Config) *Encoder {
	if config == nil {
		config = DefaultConfig
	}
	return &Encoder{Config: config}
}

// Encode writes STUN message to the buffer.
// Generates MESSAGE-INTEGRITY attribute if Key is specified.
// Appends FINGERPRINT attribute if Fingerprint is true.
func (enc *Encoder) Encode(m *Message) ([]byte, error) {
	w := &enc.w
	w.pos = 0
	h := w.Next(20)
	be.PutUint16(h, m.Method)

	if m.Transaction == nil {
		tx := make([]byte, 16)
		be.PutUint32(tx, magicCookie)
		rand.Read(tx[4:])
		m.Transaction = tx
	}
	if enc.Software != "" {
		m.Attributes[AttrSoftware] = enc.Software
	}

	copy(h[4:], m.Transaction)

	for at, v := range m.Attributes {
		b := w.Next(4)
		p := w.pos
		codec := enc.getAttrCodec(at)
		if codec == nil {
			return nil, &errUnknownAttrCodec{at}
		}
		err := codec.Encode(w, v)
		if err != nil {
			return nil, err
		}
		n := w.pos - p
		be.PutUint16(b, at)
		be.PutUint16(b[2:], uint16(n))

		// Padding
		if mod := n & 3; mod != 0 {
			b = w.Next(4 - mod)
			for i := range b {
				b[i] = 0
			}
		}
	}
	if m.Key == nil && enc.GetAuthKey != nil {
		key, err := enc.getAuthKey(m.Attributes)
		if err != nil {
			return nil, err
		}
		m.Key = key
	}
	if m.Key != nil {
		be.PutUint16(h[2:], uint16(w.pos+4))
		p := w.pos
		b := w.Next(24)
		be.PutUint16(b, AttrMessageIntegrity)
		be.PutUint16(b[2:], 20)
		// TODO: sum into byte slice
		copy(b[4:], integrity(w.buf[:p], m.Key))
	}
	if enc.Fingerprint {
		be.PutUint16(h[2:], uint16(w.pos-12))
		p := w.pos
		b := w.Next(8)
		be.PutUint16(b, AttrFingerprint)
		be.PutUint16(b[2:], 4)
		be.PutUint32(b[4:], fingerprint(w.buf[:p]))
	}
	be.PutUint16(h[2:], uint16(w.pos-20))
	return w.buf[:w.pos], nil
}

// checksum calculates FINGERPRINT attribute value for the STUN message bytes.
// See RFC 5389 Section 15.5
func fingerprint(v []byte) uint32 {
	return crc32.ChecksumIEEE(v) ^ 0x5354554e
}

// integrity calculates MESSAGE-INTEGRITY attribute value for the STUN message bytes.
func integrity(v, key []byte) []byte {
	h := hmac.New(sha1.New, key)
	h.Write(v)
	return h.Sum(nil)
}

type Writer interface {
	// Next returns a slice of the next n bytes.
	Next(n int) []byte
}

type writer struct {
	buf []byte
	pos int
}

func (w *writer) Next(n int) (b []byte) {
	p := w.pos + n
	if len(w.buf) < p {
		b := make([]byte, (1+((p-1)>>10))<<10)
		if w.pos > 0 {
			copy(b, w.buf[:w.pos])
		}
		w.buf = b
	}
	b, w.pos = w.buf[w.pos:p], p
	return
}
