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
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// FormatterSymbol is a special symbol used in config files to mark special format aliases.
const (
	FormatterSymbol = '%'
)

const (
	formatterParameterStart = '('
	formatterParameterEnd   = ')'
)

// Time and date formats used for %Date and %Time aliases.
const (
	DateDefaultFormat = "2006-01-02"
	TimeFormat        = "15:04:05"
)

var DefaultMsgFormat = "%Ns [%Level] %Msg%n"

var (
	DefaultFormatter *formatter
	msgonlyformatter *formatter
)

func init() {
	var err error
	if DefaultFormatter, err = NewFormatter(DefaultMsgFormat); err != nil {
		reportInternalError(fmt.Errorf("error during creating DefaultFormatter: %s", err))
	}
	if msgonlyformatter, err = NewFormatter("%Msg"); err != nil {
		reportInternalError(fmt.Errorf("error during creating msgonlyformatter: %s", err))
	}
}

// FormatterFunc represents one formatter object that starts with '%' sign in the 'format' attribute
// of the 'format' config item. These special symbols are replaced with context values or special
// strings when message is written to byte receiver.
//
// Check https://github.com/cihub/seelog/wiki/Formatting for details.
// Full list (with descriptions) of formatters: https://github.com/cihub/seelog/wiki/Format-reference
//
// FormatterFunc takes raw log message, level, log context and returns a string, number (of any type) or any object
// that can be evaluated as string.
type FormatterFunc func(message string, level LogLevel, context LogContextInterface) interface{}

// FormatterFuncCreator is a factory of FormatterFunc objects. It is used to generate parameterized
// formatters (such as %Date or %EscM) and custom user formatters.
type FormatterFuncCreator func(param string) FormatterFunc

var formatterFuncs = map[string]FormatterFunc{
	"Level":     formatterLevel,
	"Lev":       formatterLev,
	"LEVEL":     formatterLEVEL,
	"LEV":       formatterLEV,
	"l":         formatterl,
	"Msg":       formatterMsg,
	"FullPath":  formatterFullPath,
	"File":      formatterFile,
	"RelFile":   formatterRelFile,
	"Func":      FormatterFunction,
	"FuncShort": FormatterFunctionShort,
	"Line":      formatterLine,
	"Time":      formatterTime,
	"UTCTime":   formatterUTCTime,
	"Ns":        formatterNs,
	"UTCNs":     formatterUTCNs,
	"r":         formatterr,
	"n":         formattern,
	"t":         formattert,
}

var formatterFuncsParameterized = map[string]FormatterFuncCreator{
	"Date":    createDateTimeFormatterFunc,
	"UTCDate": createUTCDateTimeFormatterFunc,
	"EscM":    createANSIEscapeFunc,
}

func errorAliasReserved(name string) error {
	return fmt.Errorf("cannot use '%s' as custom formatter name. Name is reserved", name)
}

// RegisterCustomFormatter registers a new custom formatter factory with a given name. If returned error is nil,
// then this name (prepended by '%' symbol) can be used in 'format' attributes in configuration and
// it will be treated like the standard parameterized formatter identifiers.
//
// RegisterCustomFormatter needs to be called before creating a logger for it to take effect. The general recommendation
// is to call it once in 'init' func of your application or any initializer func.
//
// For usage examples, check https://github.com/cihub/seelog/wiki/Custom-formatters.
//
// Name must only consist of letters (unicode.IsLetter).
//
// Name must not be one of the already registered standard formatter names
// (https://github.com/cihub/seelog/wiki/Format-reference) and previously registered
// custom format names. To avoid any potential name conflicts (in future releases), it is recommended
// to start your custom formatter name with a namespace (e.g. 'MyCompanySomething') or a 'Custom' keyword.
func RegisterCustomFormatter(name string, creator FormatterFuncCreator) error {
	if _, ok := formatterFuncs[name]; ok {
		return errorAliasReserved(name)
	}
	if _, ok := formatterFuncsParameterized[name]; ok {
		return errorAliasReserved(name)
	}
	formatterFuncsParameterized[name] = creator
	return nil
}

// formatter is used to write messages in a specific format, inserting such additional data
// as log level, date/time, etc.
type formatter struct {
	fmtStringOriginal string
	fmtString         string
	formatterFuncs    []FormatterFunc
}

// NewFormatter creates a new formatter using a format string
func NewFormatter(formatString string) (*formatter, error) {
	fmtr := new(formatter)
	fmtr.fmtStringOriginal = formatString
	if err := buildFormatterFuncs(fmtr); err != nil {
		return nil, err
	}
	return fmtr, nil
}

func buildFormatterFuncs(formatter *formatter) error {
	var (
		fsbuf  = new(bytes.Buffer)
		fsolm1 = len(formatter.fmtStringOriginal) - 1
	)
	for i := 0; i <= fsolm1; i++ {
		if char := formatter.fmtStringOriginal[i]; char != FormatterSymbol {
			fsbuf.WriteByte(char)
			continue
		}
		// Check if the index is at the end of the string.
		if i == fsolm1 {
			return fmt.Errorf("format error: %c cannot be last symbol", FormatterSymbol)
		}
		// Check if the formatter symbol is doubled and skip it as nonmatching.
		if formatter.fmtStringOriginal[i+1] == FormatterSymbol {
			fsbuf.WriteRune(FormatterSymbol)
			i++
			continue
		}
		function, ni, err := formatter.extractFormatterFunc(i + 1)
		if err != nil {
			return err
		}
		// Append formatting string "%v".
		fsbuf.Write([]byte{37, 118})
		i = ni
		formatter.formatterFuncs = append(formatter.formatterFuncs, function)
	}
	formatter.fmtString = fsbuf.String()
	return nil
}

func (formatter *formatter) extractFormatterFunc(index int) (FormatterFunc, int, error) {
	letterSequence := formatter.extractLetterSequence(index)
	if len(letterSequence) == 0 {
		return nil, 0, fmt.Errorf("format error: lack of formatter after %c at %d", FormatterSymbol, index)
	}

	function, formatterLength, ok := formatter.findFormatterFunc(letterSequence)
	if ok {
		return function, index + formatterLength - 1, nil
	}

	function, formatterLength, ok, err := formatter.findFormatterFuncParametrized(letterSequence, index)
	if err != nil {
		return nil, 0, err
	}
	if ok {
		return function, index + formatterLength - 1, nil
	}

	return nil, 0, errors.New("format error: unrecognized formatter at " + strconv.Itoa(index) + ": " + letterSequence)
}

func (formatter *formatter) extractLetterSequence(index int) string {
	letters := ""

	bytesToParse := []byte(formatter.fmtStringOriginal[index:])
	runeCount := utf8.RuneCount(bytesToParse)
	for i := 0; i < runeCount; i++ {
		rune, runeSize := utf8.DecodeRune(bytesToParse)
		bytesToParse = bytesToParse[runeSize:]

		if unicode.IsLetter(rune) {
			letters += string(rune)
		} else {
			break
		}
	}
	return letters
}

func (formatter *formatter) findFormatterFunc(letters string) (FormatterFunc, int, bool) {
	currentVerb := letters
	for i := 0; i < len(letters); i++ {
		function, ok := formatterFuncs[currentVerb]
		if ok {
			return function, len(currentVerb), ok
		}
		currentVerb = currentVerb[:len(currentVerb)-1]
	}

	return nil, 0, false
}

func (formatter *formatter) findFormatterFuncParametrized(letters string, lettersStartIndex int) (FormatterFunc, int, bool, error) {
	currentVerb := letters
	for i := 0; i < len(letters); i++ {
		functionCreator, ok := formatterFuncsParameterized[currentVerb]
		if ok {
			parameter := ""
			parameterLen := 0
			isVerbEqualsLetters := i == 0 // if not, then letter goes after formatter, and formatter is parameterless
			if isVerbEqualsLetters {
				userParameter := ""
				var err error
				userParameter, parameterLen, ok, err = formatter.findparameter(lettersStartIndex + len(currentVerb))
				if ok {
					parameter = userParameter
				} else if err != nil {
					return nil, 0, false, err
				}
			}

			return functionCreator(parameter), len(currentVerb) + parameterLen, true, nil
		}

		currentVerb = currentVerb[:len(currentVerb)-1]
	}

	return nil, 0, false, nil
}

func (formatter *formatter) findparameter(startIndex int) (string, int, bool, error) {
	if len(formatter.fmtStringOriginal) == startIndex || formatter.fmtStringOriginal[startIndex] != formatterParameterStart {
		return "", 0, false, nil
	}

	endIndex := strings.Index(formatter.fmtStringOriginal[startIndex:], string(formatterParameterEnd))
	if endIndex == -1 {
		return "", 0, false, fmt.Errorf("Unmatched parenthesis or invalid parameter at %d: %s",
			startIndex, formatter.fmtStringOriginal[startIndex:])
	}
	endIndex += startIndex

	length := endIndex - startIndex + 1

	return formatter.fmtStringOriginal[startIndex+1 : endIndex], length, true, nil
}

// Format processes a message with special formatters, log level, and context. Returns formatted string
// with all formatter identifiers changed to appropriate values.
func (formatter *formatter) Format(message string, level LogLevel, context LogContextInterface) string {
	if len(formatter.formatterFuncs) == 0 {
		return formatter.fmtString
	}

	params := make([]interface{}, len(formatter.formatterFuncs))
	for i, function := range formatter.formatterFuncs {
		params[i] = function(message, level, context)
	}

	return fmt.Sprintf(formatter.fmtString, params...)
}

func (formatter *formatter) String() string {
	return formatter.fmtStringOriginal
}

//=====================================================

const (
	wrongLogLevel   = "WRONG_LOGLEVEL"
	wrongEscapeCode = "WRONG_ESCAPE"
)

var levelToString = map[LogLevel]string{
	TraceLvl:    "Trace",
	DebugLvl:    "Debug",
	InfoLvl:     "Info",
	WarnLvl:     "Warn",
	ErrorLvl:    "Error",
	CriticalLvl: "Critical",
	Off:         "Off",
}

var levelToShortString = map[LogLevel]string{
	TraceLvl:    "Trc",
	DebugLvl:    "Dbg",
	InfoLvl:     "Inf",
	WarnLvl:     "Wrn",
	ErrorLvl:    "Err",
	CriticalLvl: "Crt",
	Off:         "Off",
}

var levelToShortestString = map[LogLevel]string{
	TraceLvl:    "t",
	DebugLvl:    "d",
	InfoLvl:     "i",
	WarnLvl:     "w",
	ErrorLvl:    "e",
	CriticalLvl: "c",
	Off:         "o",
}

func formatterLevel(message string, level LogLevel, context LogContextInterface) interface{} {
	levelStr, ok := levelToString[level]
	if !ok {
		return wrongLogLevel
	}
	return levelStr
}

func formatterLev(message string, level LogLevel, context LogContextInterface) interface{} {
	levelStr, ok := levelToShortString[level]
	if !ok {
		return wrongLogLevel
	}
	return levelStr
}

func formatterLEVEL(message string, level LogLevel, context LogContextInterface) interface{} {
	return strings.ToTitle(formatterLevel(message, level, context).(string))
}

func formatterLEV(message string, level LogLevel, context LogContextInterface) interface{} {
	return strings.ToTitle(formatterLev(message, level, context).(string))
}

func formatterl(message string, level LogLevel, context LogContextInterface) interface{} {
	levelStr, ok := levelToShortestString[level]
	if !ok {
		return wrongLogLevel
	}
	return levelStr
}

func formatterMsg(message string, level LogLevel, context LogContextInterface) interface{} {
	return message
}

func formatterFullPath(message string, level LogLevel, context LogContextInterface) interface{} {
	return context.FullPath()
}

func formatterFile(message string, level LogLevel, context LogContextInterface) interface{} {
	return context.FileName()
}

func formatterRelFile(message string, level LogLevel, context LogContextInterface) interface{} {
	return context.ShortPath()
}

func FormatterFunction(message string, level LogLevel, context LogContextInterface) interface{} {
	return context.Func()
}

func FormatterFunctionShort(message string, level LogLevel, context LogContextInterface) interface{} {
	f := context.Func()
	spl := strings.Split(f, ".")
	return spl[len(spl)-1]
}

func formatterLine(message string, level LogLevel, context LogContextInterface) interface{} {
	return context.Line()
}

func formatterTime(message string, level LogLevel, context LogContextInterface) interface{} {
	return context.CallTime().Format(TimeFormat)
}

func formatterUTCTime(message string, level LogLevel, context LogContextInterface) interface{} {
	return context.CallTime().UTC().Format(TimeFormat)
}

func formatterNs(message string, level LogLevel, context LogContextInterface) interface{} {
	return context.CallTime().UnixNano()
}

func formatterUTCNs(message string, level LogLevel, context LogContextInterface) interface{} {
	return context.CallTime().UTC().UnixNano()
}

func formatterr(message string, level LogLevel, context LogContextInterface) interface{} {
	return "\r"
}

func formattern(message string, level LogLevel, context LogContextInterface) interface{} {
	return "\n"
}

func formattert(message string, level LogLevel, context LogContextInterface) interface{} {
	return "\t"
}

func createDateTimeFormatterFunc(dateTimeFormat string) FormatterFunc {
	format := dateTimeFormat
	if format == "" {
		format = DateDefaultFormat
	}
	return func(message string, level LogLevel, context LogContextInterface) interface{} {
		return context.CallTime().Format(format)
	}
}

func createUTCDateTimeFormatterFunc(dateTimeFormat string) FormatterFunc {
	format := dateTimeFormat
	if format == "" {
		format = DateDefaultFormat
	}
	return func(message string, level LogLevel, context LogContextInterface) interface{} {
		return context.CallTime().UTC().Format(format)
	}
}

func createANSIEscapeFunc(escapeCodeString string) FormatterFunc {
	return func(message string, level LogLevel, context LogContextInterface) interface{} {
		if len(escapeCodeString) == 0 {
			return wrongEscapeCode
		}

		return fmt.Sprintf("%c[%sm", 0x1B, escapeCodeString)
	}
}
