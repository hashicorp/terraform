package stun

import (
	"crypto/md5"
	"encoding/hex"
	"net"
	"testing"
)

func TestTCPClientServer(t *testing.T) {
	srv := NewServer(nil)
	l, err := net.Listen("tcp", "")
	if err != nil {
		t.Fatal("listen error", err)
	}
	defer l.Close()
	go srv.Serve(l)

	c, err := net.Dial(l.Addr().Network(), l.Addr().String())
	if err != nil {
		t.Fatal("dial error", err)
	}
	conn := NewClient(c, nil)
	defer conn.Close()

	req := &Message{Method: MethodBinding}
	msg, err := conn.RoundTrip(req)
	if err != nil {
		t.Fatal("exchange error", err)
	}
	if msg == nil || msg.Attributes[AttrXorMappedAddress] == nil {
		t.Fatal("response error")
	}
}

func TestUDPClientServer(t *testing.T) {
	srv := NewServer(nil)
	l, err := net.ListenPacket("udp", "")
	if err != nil {
		t.Fatal("listen error", err)
	}
	defer l.Close()
	go srv.ServePacket(l)

	c, err := net.Dial(l.LocalAddr().Network(), l.LocalAddr().String())
	if err != nil {
		t.Fatal("dial error", err)
	}
	conn := NewClient(c, nil)
	defer conn.Close()

	req := &Message{Method: MethodBinding}
	msg, err := conn.RoundTrip(req)
	if err != nil {
		t.Fatal("exchange error", err)
	}
	if msg == nil {
		t.Fatal("response error")
	}
	if msg == nil || msg.Attributes[AttrXorMappedAddress] == nil {
		t.Fatal("response error")
	}
}

func TestLookupAddr(t *testing.T) {
	srv := NewServer(nil)
	l, err := net.ListenPacket("udp", "")
	if err != nil {
		t.Fatal("listen error", err)
	}
	defer l.Close()
	go srv.ServePacket(l)

	addr, err := Lookup("stun:"+l.LocalAddr().String(), "", "")
	if err != nil {
		t.Fatal("lookup", err)
	}
	if addr == nil {
		t.Fatal("no address")
	}
}

// Test Vectors for STUN. RFC 5769.

func TestVectorsSampleRequest(t *testing.T) {
	b, err := hex.DecodeString("000100582112a442b7e7a701bc34d686fa87dfae802200105354554e207465737420636c69656e74002400046e0001ff80290008932ff9b151263b36000600096576746a3a68367659202020000800149aeaa70cbfd8cb56781ef2b5b2d3f249c1b571a280280004e57a3bcf")
	if err != nil {
		t.Fatal("decode hex", err)
	}
	dec := NewDecoder(nil)
	msg, err := dec.Decode(b, []byte("VOkJxbRl1RmTxUk/WvJxBt"))
	if unk, ok := err.(*ErrUnknownAttrs); ok {
		if len(unk.Attributes) == 1 && unk.Attributes[0] == 0x24 {
			err = nil
		} else {
			t.Fatal("unknown attributes")
		}
	}
	if err != nil {
		t.Fatal("decode error", err)
	}
	if !msg.IsType(TypeRequest) || msg.Method&^0x110 != MethodBinding {
		t.Fatal("message type error", msg.Method)
	}
	if v := msg.Attributes.String(AttrSoftware); v != "STUN test client" {
		t.Fatal("software check", v)
	}
	if v := msg.Attributes.String(AttrUsername); v != "evtj:h6vY" {
		t.Fatal("username check", v)
	}
}

func TestVectorsSampleIPv4Response(t *testing.T) {
	b, err := hex.DecodeString("0101003c2112a442b7e7a701bc34d686fa87dfae8022000b7465737420766563746f7220002000080001a147e112a643000800142b91f599fd9e90c38c7489f92af9ba53f06be7d780280004c07d4c96")
	if err != nil {
		t.Fatal("decode hex", err)
	}
	dec := NewDecoder(nil)
	msg, err := dec.Decode(b, []byte("VOkJxbRl1RmTxUk/WvJxBt"))
	if err != nil {
		t.Fatal("decode error", err)
	}
	if !msg.IsType(TypeResponse) || msg.Method&^0x110 != MethodBinding {
		t.Fatal("message type error", msg.Method)
	}
	if v := msg.Attributes.String(AttrSoftware); v != "test vector" {
		t.Fatal("software check", v)
	}
	addr := msg.Attributes[AttrXorMappedAddress].(*Addr)
	if addr == nil || !addr.IP.Equal(net.ParseIP("192.0.2.1")) || addr.Port != 32853 {
		t.Fatal("address check", addr)
	}
}

func TestVectorsSampleIPv6Response(t *testing.T) {
	b, err := hex.DecodeString("010100482112a442b7e7a701bc34d686fa87dfae8022000b7465737420766563746f7220002000140002a1470113a9faa5d3f179bc25f4b5bed2b9d900080014a382954e4be67bf11784c97c8292c275bfe3ed4180280004c8fb0b4c")
	if err != nil {
		t.Fatal("decode hex", err)
	}
	dec := NewDecoder(nil)
	msg, err := dec.Decode(b, []byte("VOkJxbRl1RmTxUk/WvJxBt"))
	if err != nil {
		t.Fatal("decode error", err)
	}
	if !msg.IsType(TypeResponse) || msg.Method&^0x110 != MethodBinding {
		t.Fatal("message type error", msg.Method)
	}
	if v := msg.Attributes.String(AttrSoftware); v != "test vector" {
		t.Fatal("software check", v)
	}
	addr := msg.Attributes[AttrXorMappedAddress].(*Addr)
	if addr == nil || !addr.IP.Equal(net.ParseIP("2001:db8:1234:5678:11:2233:4455:6677")) || addr.Port != 32853 {
		t.Fatal("address check", addr)
	}
}

func TestVectorsSampleLongTermAuth(t *testing.T) {
	b, err := hex.DecodeString("000100602112a44278ad3433c6ad72c029da412e00060012e3839ee38388e383aae38383e382afe382b900000015001c662f2f3439396b39353464364f4c33346f4c394653547679363473410014000b6578616d706c652e6f72670000080014f67024656dd64a3e02b8e0712e85c9a28ca89666")
	if err != nil {
		t.Fatal("decode hex", err)
	}
	checked := false
	config := &Config{
		GetAuthKey: func(attrs Attributes) ([]byte, error) {
			checked = true
			u, r := attrs.String(AttrUsername), attrs.String(AttrRealm)
			h := md5.New()
			h.Write([]byte(u + ":" + r + ":TheMatrIX"))
			return h.Sum(nil), nil
		},
	}
	dec := NewDecoder(config)
	msg, err := dec.Decode(b, nil)
	if err != nil {
		t.Fatal("decode error", err)
	}
	if !msg.IsType(TypeRequest) || msg.Method&^0x110 != MethodBinding {
		t.Fatal("message type error", msg.Method)
	}
	if !checked {
		t.Fatal("message integrity not checked")
	}
	if v := msg.Attributes.String(AttrNonce); v != "f//499k954d6OL34oL9FSTvy64sA" {
		t.Fatal("nonce check", v)
	}
	if v := msg.Attributes.String(AttrRealm); v != "example.org" {
		t.Fatal("realm check", v)
	}
}
