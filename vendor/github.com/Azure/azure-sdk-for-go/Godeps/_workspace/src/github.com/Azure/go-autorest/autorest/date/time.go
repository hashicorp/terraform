package date

import (
	"time"
)

// Time defines a type similar to time.Time but assumes a layout of RFC3339 date-time (i.e.,
// 2006-01-02T15:04:05Z).
type Time struct {
	time.Time
}

// ParseTime creates a new Time from the passed string.
func ParseTime(date string) (d Time, err error) {
	d = Time{}
	d.Time, err = time.Parse(time.RFC3339, date)
	return d, err
}

// MarshalBinary preserves the Time as a byte array conforming to RFC3339 date-time (i.e.,
// 2006-01-02T15:04:05Z).
func (d Time) MarshalBinary() ([]byte, error) {
	return d.Time.MarshalText()
}

// UnmarshalBinary reconstitutes a Time saved as a byte array conforming to RFC3339 date-time
// (i.e., 2006-01-02T15:04:05Z).
func (d *Time) UnmarshalBinary(data []byte) error {
	return d.Time.UnmarshalText(data)
}

// MarshalJSON preserves the Time as a JSON string conforming to RFC3339 date-time (i.e.,
// 2006-01-02T15:04:05Z).
func (d Time) MarshalJSON() (json []byte, err error) {
	return d.Time.MarshalJSON()
}

// UnmarshalJSON reconstitutes the Time from a JSON string conforming to RFC3339 date-time
// (i.e., 2006-01-02T15:04:05Z).
func (d *Time) UnmarshalJSON(data []byte) (err error) {
	return d.Time.UnmarshalJSON(data)
}

// MarshalText preserves the Time as a byte array conforming to RFC3339 date-time (i.e.,
// 2006-01-02T15:04:05Z).
func (d Time) MarshalText() (text []byte, err error) {
	return d.Time.MarshalText()
}

// UnmarshalText reconstitutes a Time saved as a byte array conforming to RFC3339 date-time
// (i.e., 2006-01-02T15:04:05Z).
func (d *Time) UnmarshalText(data []byte) (err error) {
	return d.Time.UnmarshalText(data)
}

// String returns the Time formatted as an RFC3339 date-time string (i.e.,
// 2006-01-02T15:04:05Z).
func (d Time) String() string {
	// Note: time.Time.String does not return an RFC3339 compliant string, time.Time.MarshalText does.
	b, err := d.Time.MarshalText()
	if err != nil {
		return ""
	}
	return string(b)
}

// ToTime returns a Time as a time.Time
func (d Time) ToTime() time.Time {
	return d.Time
}
