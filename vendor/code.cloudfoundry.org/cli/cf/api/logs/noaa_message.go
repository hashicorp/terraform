package logs

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"code.cloudfoundry.org/cli/cf/terminal"
	"github.com/cloudfoundry/sonde-go/events"
)

type noaaLogMessage struct {
	msg *events.LogMessage
}

func NewNoaaLogMessage(m *events.LogMessage) *noaaLogMessage {
	return &noaaLogMessage{
		msg: m,
	}
}

func (m *noaaLogMessage) ToSimpleLog() string {
	msgText := string(m.msg.GetMessage())

	return strings.TrimRight(msgText, "\r\n")
}

func (m *noaaLogMessage) GetSourceName() string {
	return m.msg.GetSourceType()
}

func (m *noaaLogMessage) ToLog(loc *time.Location) string {
	logMsg := m.msg

	sourceName := logMsg.GetSourceType()
	sourceID := logMsg.GetSourceInstance()
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

	if logMsg.GetMessageType() == events.LogMessage_ERR {
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
