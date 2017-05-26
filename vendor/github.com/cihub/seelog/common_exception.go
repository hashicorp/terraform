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

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Used in rules creation to validate input file and func filters
var (
	fileFormatValidator = regexp.MustCompile(`[a-zA-Z0-9\\/ _\*\.]*`)
	funcFormatValidator = regexp.MustCompile(`[a-zA-Z0-9_\*\.]*`)
)

// LogLevelException represents an exceptional case used when you need some specific files or funcs to
// override general constraints and to use their own.
type LogLevelException struct {
	funcPatternParts []string
	filePatternParts []string

	funcPattern string
	filePattern string

	constraints logLevelConstraints
}

// NewLogLevelException creates a new exception.
func NewLogLevelException(funcPattern string, filePattern string, constraints logLevelConstraints) (*LogLevelException, error) {
	if constraints == nil {
		return nil, errors.New("constraints can not be nil")
	}

	exception := new(LogLevelException)

	err := exception.initFuncPatternParts(funcPattern)
	if err != nil {
		return nil, err
	}
	exception.funcPattern = strings.Join(exception.funcPatternParts, "")

	err = exception.initFilePatternParts(filePattern)
	if err != nil {
		return nil, err
	}
	exception.filePattern = strings.Join(exception.filePatternParts, "")

	exception.constraints = constraints

	return exception, nil
}

// MatchesContext returns true if context matches the patterns of this LogLevelException
func (logLevelEx *LogLevelException) MatchesContext(context LogContextInterface) bool {
	return logLevelEx.match(context.Func(), context.FullPath())
}

// IsAllowed returns true if log level is allowed according to the constraints of this LogLevelException
func (logLevelEx *LogLevelException) IsAllowed(level LogLevel) bool {
	return logLevelEx.constraints.IsAllowed(level)
}

// FuncPattern returns the function pattern of a exception
func (logLevelEx *LogLevelException) FuncPattern() string {
	return logLevelEx.funcPattern
}

// FuncPattern returns the file pattern of a exception
func (logLevelEx *LogLevelException) FilePattern() string {
	return logLevelEx.filePattern
}

// initFuncPatternParts checks whether the func filter has a correct format and splits funcPattern on parts
func (logLevelEx *LogLevelException) initFuncPatternParts(funcPattern string) (err error) {

	if funcFormatValidator.FindString(funcPattern) != funcPattern {
		return errors.New("func path \"" + funcPattern + "\" contains incorrect symbols. Only a-z A-Z 0-9 _ * . allowed)")
	}

	logLevelEx.funcPatternParts = splitPattern(funcPattern)
	return nil
}

// Checks whether the file filter has a correct format and splits file patterns using splitPattern.
func (logLevelEx *LogLevelException) initFilePatternParts(filePattern string) (err error) {

	if fileFormatValidator.FindString(filePattern) != filePattern {
		return errors.New("file path \"" + filePattern + "\" contains incorrect symbols. Only a-z A-Z 0-9 \\ / _ * . allowed)")
	}

	logLevelEx.filePatternParts = splitPattern(filePattern)
	return err
}

func (logLevelEx *LogLevelException) match(funcPath string, filePath string) bool {
	if !stringMatchesPattern(logLevelEx.funcPatternParts, funcPath) {
		return false
	}
	return stringMatchesPattern(logLevelEx.filePatternParts, filePath)
}

func (logLevelEx *LogLevelException) String() string {
	str := fmt.Sprintf("Func: %s File: %s", logLevelEx.funcPattern, logLevelEx.filePattern)

	if logLevelEx.constraints != nil {
		str += fmt.Sprintf("Constr: %s", logLevelEx.constraints)
	} else {
		str += "nil"
	}

	return str
}

// splitPattern splits pattern into strings and asterisks. Example: "ab*cde**f" -> ["ab", "*", "cde", "*", "f"]
func splitPattern(pattern string) []string {
	var patternParts []string
	var lastChar rune
	for _, char := range pattern {
		if char == '*' {
			if lastChar != '*' {
				patternParts = append(patternParts, "*")
			}
		} else {
			if len(patternParts) != 0 && lastChar != '*' {
				patternParts[len(patternParts)-1] += string(char)
			} else {
				patternParts = append(patternParts, string(char))
			}
		}
		lastChar = char
	}

	return patternParts
}

// stringMatchesPattern check whether testString matches pattern with asterisks.
// Standard regexp functionality is not used here because of performance issues.
func stringMatchesPattern(patternparts []string, testString string) bool {
	if len(patternparts) == 0 {
		return len(testString) == 0
	}

	part := patternparts[0]
	if part != "*" {
		index := strings.Index(testString, part)
		if index == 0 {
			return stringMatchesPattern(patternparts[1:], testString[len(part):])
		}
	} else {
		if len(patternparts) == 1 {
			return true
		}

		newTestString := testString
		part = patternparts[1]
		for {
			index := strings.Index(newTestString, part)
			if index == -1 {
				break
			}

			newTestString = newTestString[index+len(part):]
			result := stringMatchesPattern(patternparts[2:], newTestString)
			if result {
				return true
			}
		}
	}
	return false
}
