package stun

import (
	"net"
	"reflect"
	"strconv"
)

// Addr represents a transport address attribute.
type Addr struct {
	IP   net.IP
	Port int
}

// String returns the "host:port" form of the transport address.
func (addr *Addr) String() string {
	return net.JoinHostPort(addr.IP.String(), strconv.Itoa(addr.Port))
}

var xorMask = []byte{0x21, 0x12, 0xa4, 0x42}

// AddrCodec is the codec for a transport address attribute.
const AddrCodec = addrCodec(false)

// XorAddrCodec is the codec for a XOR-obfuscated transport address attribute.
const XorAddrCodec = addrCodec(true)

type addrCodec bool

func (c addrCodec) Encode(w Writer, v interface{}) error {
	switch addr := v.(type) {
	case *net.UDPAddr:
		c.writeAddress(w, addr.IP, addr.Port)
	case *net.TCPAddr:
		c.writeAddress(w, addr.IP, addr.Port)
	case *Addr:
		c.writeAddress(w, addr.IP, addr.Port)
	default:
		return &errUnsupportedAttrType{Type: reflect.TypeOf(v)}
	}
	return nil
}

func (c addrCodec) writeAddress(w Writer, ip net.IP, port int) {
	fam, sh := byte(0x01), ip.To4()
	if len(sh) == 0 {
		fam, sh = byte(0x02), ip
	}
	b := w.Next(4 + len(sh))
	b[0] = 0
	b[1] = fam
	if c {
		be.PutUint16(b[2:], uint16(port)^0x2112)
		b = b[4:]
		if enc, ok := w.(*writer); ok {
			for i, it := range sh {
				b[i] = it ^ enc.buf[i+4]
			}
		} else {
			for i, it := range sh {
				b[i] = it ^ xorMask[i%4]
			}
		}
	} else {
		be.PutUint16(b[2:], uint16(port))
		copy(b[4:], sh)
	}
}

func (c addrCodec) Decode(r Reader) (interface{}, error) {
	b, err := r.Next(4)
	if err != nil {
		return nil, err
	}
	n, port := net.IPv4len, int(be.Uint16(b[2:]))
	if b[1] == 0x02 {
		n = net.IPv6len
	}
	if b, err = r.Next(n); err != nil {
		return nil, err
	}
	ip := make([]byte, len(b))
	if c {
		if dec, ok := r.(*reader); ok {
			for i, it := range b {
				ip[i] = it ^ dec.msg[i+4]
			}
		} else {
			for i, it := range b {
				ip[i] = it ^ xorMask[i%4]
			}
		}
		return &Addr{IP: ip, Port: port ^ 0x2112}, nil
	} else {
		copy(ip, b)
	}
	return &Addr{IP: ip, Port: port}, nil
}
