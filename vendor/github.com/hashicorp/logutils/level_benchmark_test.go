package logutils

import (
	"io/ioutil"
	"testing"
)

var messages [][]byte

func init() {
	messages = [][]byte{
		[]byte("[TRACE] foo"),
		[]byte("[DEBUG] foo"),
		[]byte("[INFO] foo"),
		[]byte("[WARN] foo"),
		[]byte("[ERROR] foo"),
	}
}

func BenchmarkDiscard(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ioutil.Discard.Write(messages[i%len(messages)])
	}
}

func BenchmarkLevelFilter(b *testing.B) {
	filter := &LevelFilter{
		Levels:   []LogLevel{"TRACE", "DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: "WARN",
		Writer:   ioutil.Discard,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Write(messages[i%len(messages)])
	}
}
