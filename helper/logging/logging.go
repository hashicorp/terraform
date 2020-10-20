package logging

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"syscall"
)

// These are the environmental variables that determine if we log, and if
// we log whether or not the log should go to a file.
const (
	EnvLog              = "TF_LOG"      // Set to True
	EnvLogFile          = "TF_LOG_PATH" // Set to a file
	EnvLogPatternPrefix = "TF_LOG_PATTERN_"
)

// ValidLevels are the log level names that Terraform recognizes.
var ValidLevels = []LogLevel{"TRACE", "DEBUG", "INFO", "WARN", "ERROR"}

// LogOutput determines where we should send logs (if anywhere) and the log level.
func LogOutput() (logOutput io.Writer, err error) {
	logOutput = ioutil.Discard

	logPatterns := WhitelistedLogPatterns()
	logLevel := CurrentLogLevel()

	if logLevel == "" && len(logPatterns) == 0 {
		return
	}

	logOutput = os.Stderr
	if logPath := os.Getenv(EnvLogFile); logPath != "" {
		var err error
		logOutput, err = os.OpenFile(logPath, syscall.O_CREAT|syscall.O_RDWR|syscall.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
	}

	if logLevel == "TRACE" {
		// Just pass through logs directly then, without any level filtering at all.
		return logOutput, nil
	}

	// Otherwise we'll use our level filter, which is a heuristic-based
	// best effort thing that is not totally reliable but helps to reduce
	// the volume of logs in some cases.
	logOutput = &LogFilter{
		Levels:   ValidLevels,
		Patterns: logPatterns,
		MinLevel: LogLevel(logLevel),
		Writer:   logOutput,
	}

	return logOutput, nil
}

// SetOutput checks for a log destination with LogOutput, and calls
// log.SetOutput with the result. If LogOutput returns nil, SetOutput uses
// ioutil.Discard. Any error from LogOutout is fatal.
func SetOutput() {
	out, err := LogOutput()
	if err != nil {
		log.Fatal(err)
	}

	if out == nil {
		out = ioutil.Discard
	}

	log.SetOutput(out)
}

// CurrentLogLevel returns the current log level string based the environment vars
func CurrentLogLevel() string {
	envLevel := os.Getenv(EnvLog)
	if envLevel == "" {
		return ""
	}

	logLevel := "TRACE"
	if isValidLogLevel(envLevel) {
		// allow following for better ux: info, Info or INFO
		logLevel = strings.ToUpper(envLevel)
	} else {
		log.Printf("[WARN] Invalid log level: %q. Defaulting to level: TRACE. Valid levels are: %+v",
			envLevel, ValidLevels)
	}
	if logLevel != "TRACE" {
		log.Printf("[WARN] Log levels other than TRACE are currently unreliable, and are supported only for backward compatibility.\n  Use TF_LOG=TRACE to see Terraform's internal logs.\n  ----")
	}

	return logLevel
}

// WhitelistedLogPatterns returns a list of whitelisted log line patterns.
// Matching lines will be logged regardless of the log level.
func WhitelistedLogPatterns() []*regexp.Regexp {
	patterns := make([]*regexp.Regexp, 0, 0)

	environ := os.Environ()

	for _, e := range environ {
		if !strings.HasPrefix(e, EnvLogPatternPrefix) {
			continue
		}

		splits := strings.SplitN(e, "=", 2)
		envVarName := splits[0]
		envVarValue := splits[1]

		pattern, err := regexp.Compile(envVarValue)

		if err != nil {
			log.Fatalln("Can not compile "+envVarName+":", err)
		}

		patterns = append(patterns, pattern)
	}

	return patterns
}

// IsDebugOrHigher returns whether or not the current log level is debug or trace
func IsDebugOrHigher() bool {
	level := string(CurrentLogLevel())
	return level == "DEBUG" || level == "TRACE"
}

func isValidLogLevel(level string) bool {
	for _, l := range ValidLevels {
		if strings.ToUpper(level) == string(l) {
			return true
		}
	}

	return false
}
