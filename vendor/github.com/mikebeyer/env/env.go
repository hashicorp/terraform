// Package env provides convenience wrapper around getting environment variables.
package env

import (
	"fmt"
	"os"
	"strconv"
)

// String gets string variable from the environment and
// returns it if it exists, otherwise it returns the default.
func String(key string, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

// MustString exits if an environment variable is not present.
func MustString(key string) string {
	val := os.Getenv(key)
	if val == "" {
		fmt.Printf("%s must be provided.", key)
		os.Exit(1)
	}
	return val
}

// Int gets int variable from the environment and
// returns it if it exists, otherwise it returns the default.
func Int(key string, def int) int {
	val := os.Getenv(key)
	if val == "" {
		return def
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return i
}

// Bool gets boolean variable from the environment and
// returns it if it exists, otherwise it returns the default.
func Bool(key string, def bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return def
	}

	b, err := strconv.ParseBool(val)
	if err != nil {
		return def
	}
	return b
}

// Float gets float variable with a provided bit type from the environment and
// returns it if it exists, otherwise it returns the default.
func Float(key string, def float64, bit int) float64 {
	val := os.Getenv(key)
	if val == "" {
		return def
	}

	f, err := strconv.ParseFloat(val, bit)
	if err != nil {
		return def
	}
	return f
}
