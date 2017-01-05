// Copyright (c) 2012 - Cloud Instruments Co., Ltd.
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package seelog

// Log level type
type LogLevel uint8

// Log levels
const (
	TraceLvl = iota
	DebugLvl
	InfoLvl
	WarnLvl
	ErrorLvl
	CriticalLvl
	Off
)

// Log level string representations (used in configuration files)
const (
	TraceStr    = "trace"
	DebugStr    = "debug"
	InfoStr     = "info"
	WarnStr     = "warn"
	ErrorStr    = "error"
	CriticalStr = "critical"
	OffStr      = "off"
)

var levelToStringRepresentations = map[LogLevel]string{
	TraceLvl:    TraceStr,
	DebugLvl:    DebugStr,
	InfoLvl:     InfoStr,
	WarnLvl:     WarnStr,
	ErrorLvl:    ErrorStr,
	CriticalLvl: CriticalStr,
	Off:         OffStr,
}

// LogLevelFromString parses a string and returns a corresponding log level, if sucessfull.
func LogLevelFromString(levelStr string) (level LogLevel, found bool) {
	for lvl, lvlStr := range levelToStringRepresentations {
		if lvlStr == levelStr {
			return lvl, true
		}
	}

	return 0, false
}

// LogLevelToString returns seelog string representation for a specified level. Returns "" for invalid log levels.
func (level LogLevel) String() string {
	levelStr, ok := levelToStringRepresentations[level]
	if ok {
		return levelStr
	}

	return ""
}
