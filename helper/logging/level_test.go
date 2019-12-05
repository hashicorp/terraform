package logging

import (
	"bytes"
	"io"
	"log"
	"testing"
)

func TestLevelFilter_impl(t *testing.T) {
	var _ io.Writer = new(LevelFilter)
}

func TestLevelFilter(t *testing.T) {
	buf := new(bytes.Buffer)
	filter := &LevelFilter{
		Levels:   []LogLevel{"DEBUG", "WARN", "ERROR"},
		MinLevel: "WARN",
		Writer:   buf,
	}

	logger := log.New(filter, "", 0)
	logger.Print("[WARN] foo")
	logger.Println("[ERROR] bar\n[DEBUG] buzz")
	logger.Println("[DEBUG] baz\n  continuation\n[WARN] buzz\n  more\n[DEBUG] fizz")

	result := buf.String()
	expected := "[WARN] foo\n[ERROR] bar\n[WARN] buzz\n  more\n"
	if result != expected {
		t.Fatalf("wrong result\ngot:\n%s\nwant:\n%s", result, expected)
	}
}

func TestLevelFilterCheck(t *testing.T) {
	filter := &LevelFilter{
		Levels:   []LogLevel{"DEBUG", "WARN", "ERROR"},
		MinLevel: "WARN",
		Writer:   nil,
	}

	testCases := []struct {
		line  string
		check bool
	}{
		{"[WARN] foo\n", true},
		{"[ERROR] bar\n", true},
		{"[DEBUG] baz\n", false},
		{"[WARN] buzz\n", true},
	}

	for _, testCase := range testCases {
		result := filter.Check([]byte(testCase.line))
		if result != testCase.check {
			t.Errorf("Fail: %s", testCase.line)
		}
	}
}

func TestLevelFilter_SetMinLevel(t *testing.T) {
	filter := &LevelFilter{
		Levels:   []LogLevel{"DEBUG", "WARN", "ERROR"},
		MinLevel: "ERROR",
		Writer:   nil,
	}

	testCases := []struct {
		line        string
		checkBefore bool
		checkAfter  bool
	}{
		{"[WARN] foo\n", false, true},
		{"[ERROR] bar\n", true, true},
		{"[DEBUG] baz\n", false, false},
		{"[WARN] buzz\n", false, true},
	}

	for _, testCase := range testCases {
		result := filter.Check([]byte(testCase.line))
		if result != testCase.checkBefore {
			t.Errorf("Fail: %s", testCase.line)
		}
	}

	// Update the minimum level to WARN
	filter.SetMinLevel("WARN")

	for _, testCase := range testCases {
		result := filter.Check([]byte(testCase.line))
		if result != testCase.checkAfter {
			t.Errorf("Fail: %s", testCase.line)
		}
	}
}
