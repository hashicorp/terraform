package logmessage

import (
	"errors"
	"time"

	"github.com/cloudfoundry/loggregatorlib/signature"
	"github.com/gogo/protobuf/proto"
)

type Message struct {
	logMessage       *LogMessage
	rawMessage       []byte
	rawMessageLength uint32
}

func NewMessage(logMessage *LogMessage, data []byte) *Message {
	return &Message{logMessage, data, uint32(len(data))}
}

func GenerateMessage(messageType LogMessage_MessageType, messageString, appId, sourceName string) (*Message, error) {
	currentTime := time.Now()
	logMessage := &LogMessage{
		Message:     []byte(messageString),
		AppId:       &appId,
		MessageType: &messageType,
		SourceName:  proto.String(sourceName),
		Timestamp:   proto.Int64(currentTime.UnixNano()),
	}

	lmBytes, err := proto.Marshal(logMessage)
	if err != nil {
		return nil, err
	}
	return NewMessage(logMessage, lmBytes), nil
}

func ParseMessage(data []byte) (*Message, error) {
	logMessage, err := parseLogMessage(data)
	return &Message{logMessage, data, uint32(len(data))}, err
}

func ParseEnvelope(data []byte, secret string) (message *Message, err error) {
	message = &Message{}
	logEnvelope := &LogEnvelope{}

	if err := proto.Unmarshal(data, logEnvelope); err != nil {
		return nil, err
	}
	if !logEnvelope.VerifySignature(secret) {
		return nil, errors.New("Invalid Envelope Signature")
	}

	//we pull out the LogMessage from the LogEnvelope and re-marshal it
	//because the rawMessage should not contain the information in the logEnvelope
	message.rawMessage, err = proto.Marshal(logEnvelope.LogMessage)
	if err != nil {
		return nil, err
	}

	message.logMessage = logEnvelope.LogMessage
	message.rawMessageLength = uint32(len(message.rawMessage))
	return message, nil
}

func (m *Message) GetLogMessage() *LogMessage {
	return m.logMessage
}

func (m *Message) GetRawMessage() []byte {
	return m.rawMessage
}

func (m *Message) GetRawMessageLength() uint32 {
	return m.rawMessageLength
}

func (e *LogEnvelope) VerifySignature(sharedSecret string) bool {
	messageDigest, err := signature.Decrypt(sharedSecret, e.GetSignature())
	if err != nil {
		return false
	}

	expectedDigest := e.logMessageDigest()
	return string(messageDigest) == string(expectedDigest)
}

func (e *LogEnvelope) SignEnvelope(sharedSecret string) error {
	signature, err := signature.Encrypt(sharedSecret, e.logMessageDigest())
	if err == nil {
		e.Signature = signature
	}

	return err
}

func (e *LogEnvelope) logMessageDigest() []byte {
	return signature.DigestBytes(e.LogMessage.GetMessage())
}
