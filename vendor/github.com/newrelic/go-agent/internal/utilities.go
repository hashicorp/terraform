package internal

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
)

// JSONString assists in logging JSON:  Based on the formatter used to log
// Context contents, the contents could be marshalled as JSON or just printed
// directly.
type JSONString string

// MarshalJSON returns the JSONString unmodified without any escaping.
func (js JSONString) MarshalJSON() ([]byte, error) {
	if "" == js {
		return []byte("null"), nil
	}
	return []byte(js), nil
}

func removeFirstSegment(name string) string {
	idx := strings.Index(name, "/")
	if -1 == idx {
		return name
	}
	return name[idx+1:]
}

func timeToFloatSeconds(t time.Time) float64 {
	return float64(t.UnixNano()) / float64(1000*1000*1000)
}

func timeToFloatMilliseconds(t time.Time) float64 {
	return float64(t.UnixNano()) / float64(1000*1000)
}

func floatSecondsToDuration(seconds float64) time.Duration {
	nanos := seconds * 1000 * 1000 * 1000
	return time.Duration(nanos) * time.Nanosecond
}

func absTimeDiff(t1, t2 time.Time) time.Duration {
	if t1.After(t2) {
		return t1.Sub(t2)
	}
	return t2.Sub(t1)
}

func compactJSON(js []byte) []byte {
	buf := new(bytes.Buffer)
	if err := json.Compact(buf, js); err != nil {
		return nil
	}
	return buf.Bytes()
}

// CompactJSONString removes the whitespace from a JSON string.
func CompactJSONString(js string) string {
	out := compactJSON([]byte(js))
	return string(out)
}

// StringLengthByteLimit truncates strings using a byte-limit boundary and
// avoids terminating in the middle of a multibyte character.
func StringLengthByteLimit(str string, byteLimit int) string {
	if len(str) <= byteLimit {
		return str
	}

	limitIndex := 0
	for pos := range str {
		if pos > byteLimit {
			break
		}
		limitIndex = pos
	}
	return str[0:limitIndex]
}
