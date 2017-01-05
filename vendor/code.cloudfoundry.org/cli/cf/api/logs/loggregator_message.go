package logs

import (
	"strings"
	"time"
	"unicode/utf8"

	"fmt"

	"code.cloudfoundry.org/cli/cf/terminal"
	"github.com/cloudfoundry/loggregatorlib/logmessage"
)

type loggregatorLogMessage struct {
	msg *logmessage.LogMessage
}

func NewLoggregatorLogMessage(m *logmessage.LogMessage) *loggregatorLogMessage {
	return &loggregatorLogMessage{
		msg: m,
	}
}

func (m *loggregatorLogMessage) GetSourceName() string {
	return m.msg.GetSourceName()
}

func (m *loggregatorLogMessage) ToLog(loc *time.Location) string {
	logMsg := m.msg

	sourceName := logMsg.GetSourceName()
	sourceID := logMsg.GetSourceId()
	t := time.Unix(0, logMsg.GetTimestamp())
	timeFormat := "2006-01-02T15:04:05.00-0700"
	timeString := t.In(loc).Format(timeFormat)

	var logHeader string

	if sourceID == "" {
		logHeader = fmt.Sprintf("%s [%s]", timeString, sourceName)
	} else {
		logHeader = fmt.Sprintf("%s [%s/%s]", timeString, sourceName, sourceID)
	}

	coloredLogHeader := terminal.LogSysHeaderColor(logHeader)

	// Calculate padding
	longestHeader := fmt.Sprintf("%s  [HEALTH/10] ", timeFormat)
	expectedHeaderLength := utf8.RuneCountInString(longestHeader)
	headerPadding := strings.Repeat(" ", max(0, expectedHeaderLength-utf8.RuneCountInString(logHeader)))

	logHeader = logHeader + headerPadding
	coloredLogHeader = coloredLogHeader + headerPadding

	msgText := string(logMsg.GetMessage())
	msgText = strings.TrimRight(msgText, "\r\n")

	msgLines := strings.Split(msgText, "\n")
	contentPadding := strings.Repeat(" ", utf8.RuneCountInString(logHeader))
	coloringFunc := terminal.LogStdoutColor
	logType := "OUT"

	if logMsg.GetMessageType() == logmessage.LogMessage_ERR {
		coloringFunc = terminal.LogStderrColor
		logType = "ERR"
	}

	logContent := fmt.Sprintf("%s %s", logType, msgLines[0])
	for _, msgLine := range msgLines[1:] {
		logContent = fmt.Sprintf("%s\n%s%s", logContent, contentPadding, msgLine)
	}

	logContent = coloringFunc(logContent)

	return fmt.Sprintf("%s%s", coloredLogHeader, logContent)
}

func (m *loggregatorLogMessage) ToSimpleLog() string {
	msgText := string(m.msg.GetMessage())

	return strings.TrimRight(msgText, "\r\n")
}
