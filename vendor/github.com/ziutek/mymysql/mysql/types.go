package mysql

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// For MySQL DATE type
type Date struct {
	Year       int16
	Month, Day byte
}

func (dd Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", dd.Year, dd.Month, dd.Day)
}

// True if date is 0000-00-00
func (dd Date) IsZero() bool {
	return dd.Day == 0 && dd.Month == 0 && dd.Year == 0
}

// Converts Date to time.Time using loc location.
// Converts MySQL zero to time.Time zero.
func (dd Date) Time(loc *time.Location) (t time.Time) {
	if !dd.IsZero() {
		t = time.Date(
			int(dd.Year), time.Month(dd.Month), int(dd.Day),
			0, 0, 0, 0,
			loc,
		)
	}
	return
}

// Converts Date to time.Time using Local location.
// Converts MySQL zero to time.Time zero.
func (dd Date) Localtime() time.Time {
	return dd.Time(time.Local)
}

// Convert string date in format YYYY-MM-DD to Date.
// Leading and trailing spaces are ignored.
func ParseDate(str string) (dd Date, err error) {
	str = strings.TrimSpace(str)
	if str == "0000-00-00" {
		return
	}
	var (
		y, m, d int
	)
	if len(str) != 10 || str[4] != '-' || str[7] != '-' {
		goto invalid
	}
	if y, err = strconv.Atoi(str[0:4]); err != nil {
		return
	}
	if m, err = strconv.Atoi(str[5:7]); err != nil {
		return
	}
	if m < 0 || m > 12 { // MySQL permits month == 0
		goto invalid
	}
	if d, err = strconv.Atoi(str[8:10]); err != nil {
		return
	}
	if d < 0 { // MySQL permits day == 0
		goto invalid
	}
	switch m {
	case 1, 3, 5, 7, 8, 10, 12:
		if d > 31 {
			goto invalid
		}
	case 4, 6, 9, 11:
		if d > 30 {
			goto invalid
		}
	case 2:
		if d > 29 {
			goto invalid
		}
	}
	dd.Year = int16(y)
	dd.Month = byte(m)
	dd.Day = byte(d)
	return

invalid:
	err = errors.New("Invalid MySQL DATE string: " + str)
	return
}

// Sandard MySQL datetime format
const TimeFormat = "2006-01-02 15:04:05.000000000"

// Returns t as string in MySQL format Converts time.Time zero to MySQL zero.
func TimeString(t time.Time) string {
	if t.IsZero() {
		return "0000-00-00 00:00:00"
	}
	if t.Nanosecond() == 0 {
		return t.Format(TimeFormat[:19])
	}
	return t.Format(TimeFormat)
}

// Parses string datetime in TimeFormat using loc location.
// Converts MySQL zero to time.Time zero.
func ParseTime(str string, loc *time.Location) (t time.Time, err error) {
	str = strings.TrimSpace(str)
	format := TimeFormat[:19]
	switch len(str) {
	case 10:
		if str == "0000-00-00" {
			return
		}
		format = format[:10]
	case 19:
		if str == "0000-00-00 00:00:00" {
			return
		}
	}
	// Don't expect 0000-00-00 00:00:00.0+
	t, err = time.ParseInLocation(format, str, loc)
	return
}

// Convert time.Duration to string representation of mysql.TIME
func DurationString(d time.Duration) string {
	sign := 1
	if d < 0 {
		sign = -1
		d = -d
	}
	ns := int(d % 1e9)
	d /= 1e9
	sec := int(d % 60)
	d /= 60
	min := int(d % 60)
	hour := int(d/60) * sign
	if ns == 0 {
		return fmt.Sprintf("%d:%02d:%02d", hour, min, sec)
	}
	return fmt.Sprintf("%d:%02d:%02d.%09d", hour, min, sec, ns)
}

// Parse duration from MySQL string format [+-]H+:MM:SS[.UUUUUUUUU].
// Leading and trailing spaces are ignored. If format is invalid returns nil.
func ParseDuration(str string) (dur time.Duration, err error) {
	str = strings.TrimSpace(str)
	orig := str
	// Check sign
	sign := int64(1)
	switch str[0] {
	case '-':
		sign = -1
		fallthrough
	case '+':
		str = str[1:]
	}
	var i, d int64
	// Find houre
	if nn := strings.IndexRune(str, ':'); nn != -1 {
		if i, err = strconv.ParseInt(str[0:nn], 10, 64); err != nil {
			return
		}
		d = i * 3600
		str = str[nn+1:]
	} else {
		goto invalid
	}
	if len(str) != 5 && len(str) != 15 || str[2] != ':' {
		goto invalid
	}
	if i, err = strconv.ParseInt(str[0:2], 10, 64); err != nil {
		return
	}
	if i < 0 || i > 59 {
		goto invalid
	}
	d += i * 60
	if i, err = strconv.ParseInt(str[3:5], 10, 64); err != nil {
		return
	}
	if i < 0 || i > 59 {
		goto invalid
	}
	d += i
	d *= 1e9
	if len(str) == 15 {
		if str[5] != '.' {
			goto invalid
		}
		if i, err = strconv.ParseInt(str[6:15], 10, 64); err != nil {
			return
		}
		d += i
	}
	dur = time.Duration(d * sign)
	return

invalid:
	err = errors.New("invalid MySQL TIME string: " + orig)
	return

}

type Blob []byte

type Raw struct {
	Typ uint16
	Val *[]byte
}

type Timestamp struct {
	time.Time
}

func (t Timestamp) String() string {
	return TimeString(t.Time)
}
