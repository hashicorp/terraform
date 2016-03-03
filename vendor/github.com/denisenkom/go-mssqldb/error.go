package mssql

import (
	"fmt"
)

type Error struct {
	Number     int32
	State      uint8
	Class      uint8
	Message    string
	ServerName string
	ProcName   string
	LineNo     int32
}

func (e Error) Error() string {
	return "mssql: " + e.Message
}

type StreamError struct {
	Message string
}

func (e StreamError) Error() string {
	return e.Message
}

func streamErrorf(format string, v ...interface{}) StreamError {
	return StreamError{"Invalid TDS stream: " + fmt.Sprintf(format, v...)}
}

func badStreamPanic(err error) {
	panic(err)
}

func badStreamPanicf(format string, v ...interface{}) {
	panic(streamErrorf(format, v...))
}
