package internal

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	xRequestStart = "X-Request-Start"
	xQueueStart   = "X-Queue-Start"
)

var (
	earliestAcceptableSeconds = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	latestAcceptableSeconds   = time.Date(2050, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
)

func checkQueueTimeSeconds(secondsFloat float64) time.Time {
	seconds := int64(secondsFloat)
	nanos := int64((secondsFloat - float64(seconds)) * (1000.0 * 1000.0 * 1000.0))
	if seconds > earliestAcceptableSeconds && seconds < latestAcceptableSeconds {
		return time.Unix(seconds, nanos)
	}
	return time.Time{}
}

func parseQueueTime(s string) time.Time {
	f, err := strconv.ParseFloat(s, 64)
	if nil != err {
		return time.Time{}
	}
	if f <= 0 {
		return time.Time{}
	}

	// try microseconds
	if t := checkQueueTimeSeconds(f / (1000.0 * 1000.0)); !t.IsZero() {
		return t
	}
	// try milliseconds
	if t := checkQueueTimeSeconds(f / (1000.0)); !t.IsZero() {
		return t
	}
	// try seconds
	if t := checkQueueTimeSeconds(f); !t.IsZero() {
		return t
	}
	return time.Time{}
}

// QueueDuration TODO
func QueueDuration(hdr http.Header, txnStart time.Time) time.Duration {
	s := hdr.Get(xQueueStart)
	if "" == s {
		s = hdr.Get(xRequestStart)
	}
	if "" == s {
		return 0
	}

	s = strings.TrimPrefix(s, "t=")
	qt := parseQueueTime(s)
	if qt.IsZero() {
		return 0
	}
	if qt.After(txnStart) {
		return 0
	}
	return txnStart.Sub(qt)
}
