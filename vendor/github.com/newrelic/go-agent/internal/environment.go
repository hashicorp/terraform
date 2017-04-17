package internal

import (
	"encoding/json"
	"reflect"
	"runtime"
)

// Environment describes the application's environment.
type Environment struct {
	Compiler string `env:"runtime.Compiler"`
	GOARCH   string `env:"runtime.GOARCH"`
	GOOS     string `env:"runtime.GOOS"`
	Version  string `env:"runtime.Version"`
	NumCPU   int    `env:"runtime.NumCPU"`
}

var (
	// SampleEnvironment is useful for testing.
	SampleEnvironment = Environment{
		Compiler: "comp",
		GOARCH:   "arch",
		GOOS:     "goos",
		Version:  "vers",
		NumCPU:   8,
	}
)

// NewEnvironment returns a new Environment.
func NewEnvironment() Environment {
	return Environment{
		Compiler: runtime.Compiler,
		GOARCH:   runtime.GOARCH,
		GOOS:     runtime.GOOS,
		Version:  runtime.Version(),
		NumCPU:   runtime.NumCPU(),
	}
}

// MarshalJSON prepares Environment JSON in the format expected by the collector
// during the connect command.
func (e Environment) MarshalJSON() ([]byte, error) {
	var arr [][]interface{}

	val := reflect.ValueOf(e)
	numFields := val.NumField()

	arr = make([][]interface{}, numFields)

	for i := 0; i < numFields; i++ {
		v := val.Field(i)
		t := val.Type().Field(i).Tag.Get("env")

		arr[i] = []interface{}{
			t,
			v.Interface(),
		}
	}

	return json.Marshal(arr)
}
