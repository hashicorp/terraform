package stun

// MethodBinding is the STUN binding request method.
const MethodBinding = uint16(0x0001)

// Types of a STUN message.
const (
	TypeRequest    = uint16(0x0000)
	TypeIndication = uint16(0x0010)
	TypeResponse   = uint16(0x0100)
	TypeError      = uint16(0x0110)
)

// Message represents a STUN message.
type Message struct {
	Method      uint16
	Transaction []byte
	Attributes  Attributes
	Key         []byte
}

// IsType checks if the STUN message corresponds the specified type.
func (m *Message) IsType(t uint16) bool {
	return (m.Method & 0x110) == t
}

const magicCookie = uint32(0x2112a442)
