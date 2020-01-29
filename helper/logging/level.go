package logging

import (
	"bytes"
	"io"
	"sync"
)

// LogLevel is a special string, conventionally written all in uppercase, that
// can be used to mark a log line for filtering and to specify filtering
// levels in the LevelFilter type.
type LogLevel string

// LevelFilter is an io.Writer that can be used with a logger that
// will attempt to filter out log messages that aren't at least a certain
// level.
//
// This filtering is HEURISTIC-BASED, and so will not be 100% reliable. The
// assumptions it makes are:
//
//   - Individual log messages are never split across multiple calls to the
//     Write method.
//
//   - Messages that carry levels are marked by a sequence starting with "[",
//     then the level name string, and then "]". Any message without a sequence
//     like this is an un-levelled message, and is not subject to filtering.
//
//   - Each \n-delimited line in a write is a separate log message, unless a
//     line starts with at least one space in which case it is interpreted
//     as a continuation of the previous line.
//
//   - If a log line starts with a non-whitespace character that isn't a digit
//     then it's recognized as a degenerate continuation, because "real" log
//     lines should start with a date/time and thus always have a leading
//     digit. (This also cleans up after some situations where the assumptuion
//     that messages arrive atomically aren't met, which is sadly sometimes
//     true for longer messages that trip over some buffering behavior in
//     panicwrap.)
//
// Because logging is a cross-cutting concern and not fully under the control
// of Terraform itself, there will certainly be cases where the above
// heuristics will fail. For example, it is likely that LevelFilter will
// occasionally misinterpret a continuation line as a new message because the
// code generating it doesn't know about our indentation convention.
//
// Our goal here is just to make a best effort to reduce the log volume,
// accepting that the results will not be 100% correct.
//
// Logging calls within Terraform Core should follow the above conventions so
// that the log output is broadly correct, however.
//
// Once the filter is in use somewhere, it is not safe to modify
// the structure.
type LevelFilter struct {
	// Levels is the list of log levels, in increasing order of
	// severity. Example might be: {"DEBUG", "WARN", "ERROR"}.
	Levels []LogLevel

	// MinLevel is the minimum level allowed through
	MinLevel LogLevel

	// The underlying io.Writer where log messages that pass the filter
	// will be set.
	Writer io.Writer

	badLevels map[LogLevel]struct{}
	show      bool
	once      sync.Once
}

// Check will check a given line if it would be included in the level
// filter.
func (f *LevelFilter) Check(line []byte) bool {
	f.once.Do(f.init)

	// Check for a log level
	var level LogLevel
	x := bytes.IndexByte(line, '[')
	if x >= 0 {
		y := bytes.IndexByte(line[x:], ']')
		if y >= 0 {
			level = LogLevel(line[x+1 : x+y])
		}
	}

	//return level == ""

	_, ok := f.badLevels[level]
	return !ok
}

// Write is a specialized implementation of io.Writer suitable for being
// the output of a logger from the "log" package.
//
// This Writer implementation assumes that it will only recieve byte slices
// containing one or more entire lines of log output, each one terminated by
// a newline. This is compatible with the behavior of the "log" package
// directly, and is also tolerant of intermediaries that might buffer multiple
// separate writes together, as long as no individual log line is ever
// split into multiple slices.
//
// Behavior is undefined if any log line is split across multiple writes or
// written without a trailing '\n' delimiter.
func (f *LevelFilter) Write(p []byte) (n int, err error) {
	for len(p) > 0 {
		// Split at the first \n, inclusive
		idx := bytes.IndexByte(p, '\n')
		if idx == -1 {
			// Invalid, undelimited write. We'll tolerate it assuming that
			// our assumptions are being violated, but the results may be
			// non-ideal.
			idx = len(p) - 1
			break
		}
		var l []byte
		l, p = p[:idx+1], p[idx+1:]
		// Lines starting with characters other than decimal digits (including
		// whitespace) are assumed to be continuations lines. This is an
		// imprecise heuristic, but experimentally it seems to generate
		// "good enough" results from Terraform Core's own logging. Its mileage
		// may vary with output from other systems.
		if l[0] >= '0' && l[0] <= '9' {
			f.show = f.Check(l)
		}
		if f.show {
			_, err = f.Writer.Write(l)
			if err != nil {
				// Technically it's not correct to say we've written the whole
				// buffer, but for our purposes here it's good enough as we're
				// only implementing io.Writer enough to satisfy logging
				// use-cases.
				return len(p), err
			}
		}
	}

	// We always behave as if we wrote the whole of the buffer, even if
	// we actually skipped some lines. We're only implementiong io.Writer
	// enough to satisfy logging use-cases.
	return len(p), nil
}

// SetMinLevel is used to update the minimum log level
func (f *LevelFilter) SetMinLevel(min LogLevel) {
	f.MinLevel = min
	f.init()
}

func (f *LevelFilter) init() {
	badLevels := make(map[LogLevel]struct{})
	for _, level := range f.Levels {
		if level == f.MinLevel {
			break
		}
		badLevels[level] = struct{}{}
	}
	f.badLevels = badLevels
	f.show = true
}
