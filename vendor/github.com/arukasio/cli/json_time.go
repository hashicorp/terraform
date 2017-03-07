package arukas

import (
	"fmt"
	"time"
)

// JSONTime is time.Time that serializes as unix timestamp (in microseconds).
type JSONTime time.Time

// UnmarshalJSON sets *t to a copy of data.
func (t *JSONTime) UnmarshalJSON(data []byte) (err error) {
	parsed, err := time.Parse(`"`+time.RFC3339Nano+`"`, string(data))
	if err != nil {
		return err
	}
	*t = JSONTime(parsed)
	return
}

// MarshalJSON returns t as the JSON encoding of t.
func (t JSONTime) MarshalJSON() ([]byte, error) {
	stamp := fmt.Sprintf("\"%s\"", time.Time(t).Format(time.RFC3339Nano))
	return []byte(stamp), nil
}

// String return t as the string of t.
func (t JSONTime) String() string {
	return time.Time(t).Format(time.RFC3339Nano)
}

// Time return t as the time of t.
func (t JSONTime) Time() time.Time {
	return time.Time(t)
}
