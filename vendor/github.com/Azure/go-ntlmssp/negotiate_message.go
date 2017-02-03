package ntlmssp

import (
	"bytes"
	"encoding/binary"
)

type negotiateMessageFields struct {
	messageHeader
	NegotiateFlags negotiateFlags
}

//NewNegotiateMessage creates a new NEGOTIATE message with the
//flags that this package supports.
func NewNegotiateMessage() []byte {
	m := negotiateMessageFields{
		messageHeader: newMessageHeader(1),
	}

	m.NegotiateFlags = negotiateFlagNTLMSSPREQUESTTARGET |
		negotiateFlagNTLMSSPNEGOTIATENTLM |
		negotiateFlagNTLMSSPNEGOTIATEALWAYSSIGN |
		negotiateFlagNTLMSSPNEGOTIATEUNICODE

	b := bytes.Buffer{}
	err := binary.Write(&b, binary.LittleEndian, &m)
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}
