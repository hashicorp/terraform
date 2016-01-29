package iso8601

import (
	"testing"
	"time"
)

func Test_parse_duration(t *testing.T) {
	var dur time.Duration
	var err error

	// test with bad format
	_, err = ParseDuration("asdf")
	if err != ErrBadFormat {
		t.Fatalf("Expected an ErrBadFormat")
	}

	// test with month
	_, err = ParseDuration("P1M")
	if err != ErrNoMonth {
		t.Fatalf("Expected an ErrNoMonth")
	}

	// test with good full string
	exp, _ := time.ParseDuration("51h4m5s")
	dur, err = ParseDuration("P2DT3H4M5S")
	if err != nil {
		t.Fatalf("Did not expect err: %v", err)
	}
	if dur.Hours() != exp.Hours() {
		t.Errorf("Expected %v hours, not %v", exp.Hours(), dur.Hours())
	}
	if dur.Minutes() != exp.Minutes() {
		t.Errorf("Expected %v minutes, not %v", exp.Hours(), dur.Minutes())
	}
	if dur.Seconds() != exp.Seconds() {
		t.Errorf("Expected 5 seconds, not %v", exp.Nanoseconds(), dur.Seconds())
	}
	if dur.Nanoseconds() != exp.Nanoseconds() {
		t.Error("Expected %v nanoseconds, not %v", exp.Nanoseconds(), dur.Nanoseconds())
	}

	// test with good week string
	dur, err = ParseDuration("P1W")
	if err != nil {
		t.Fatalf("Did not expect err: %v", err)
	}
	if dur.Hours() != 24*7 {
		t.Errorf("Expected 168 hours, not %d", dur.Hours())
	}
}

func Test_format_duration(t *testing.T) {
	// Test complex duration with hours, minutes, seconds
	d := time.Duration(3701) * time.Second
	s := FormatDuration(d)
	if s != "PT1H1M41S" {
		t.Fatalf("bad ISO 8601 duration string: %s", s)
	}

	// Test only minutes duration
	d = time.Duration(20) * time.Minute
	s = FormatDuration(d)
	if s != "PT20M" {
		t.Fatalf("bad ISO 8601 duration string for 20M: %s", s)
	}

	// Test only seconds
	d = time.Duration(1) * time.Second
	s = FormatDuration(d)
	if s != "PT1S" {
		t.Fatalf("bad ISO 8601 duration string for 1S: %s", s)
	}

	// Test negative duration (unsupported)
	d = time.Duration(-1) * time.Second
	s = FormatDuration(d)
	if s != "PT0S" {
		t.Fatalf("bad ISO 8601 duration string for negative: %s", s)
	}
}
