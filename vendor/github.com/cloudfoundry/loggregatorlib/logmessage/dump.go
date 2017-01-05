package logmessage

import (
	"bytes"
	"encoding/binary"

	"github.com/gogo/protobuf/proto"
)

func DumpMessage(msg Message, buffer *bytes.Buffer) {
	binary.Write(buffer, binary.BigEndian, msg.GetRawMessageLength())
	buffer.Write(msg.GetRawMessage())
}

func ParseDumpedLogMessages(b []byte) (messages []*LogMessage, err error) {
	buffer := bytes.NewBuffer(b)
	var length uint32
	for buffer.Len() > 0 {
		lengthBytes := bytes.NewBuffer(buffer.Next(4))
		err = binary.Read(lengthBytes, binary.BigEndian, &length)
		if err != nil {
			return
		}

		msgBytes := buffer.Next(int(length))
		var msg *LogMessage
		msg, err = parseLogMessage(msgBytes)
		if err != nil {
			return
		}
		messages = append(messages, msg)
	}
	return
}

func parseLogMessage(data []byte) (logMessage *LogMessage, err error) {
	logMessage = new(LogMessage)
	err = proto.Unmarshal(data, logMessage)
	return logMessage, err
}
