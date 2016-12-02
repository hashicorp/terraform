package repl

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// FormatResult formats the given result value for human-readable output.
//
// The value must currently be a string, list, map, and any nested values
// with those same types.
func FormatResult(value interface{}) (string, error) {
	return formatResult(value)
}

func formatResult(value interface{}) (string, error) {
	switch output := value.(type) {
	case string:
		return output, nil
	case []interface{}:
		return formatListResult(output)
	case map[string]interface{}:
		return formatMapResult(output)
	default:
		return "", fmt.Errorf("unknown value type: %T", value)
	}
}

func formatListResult(value []interface{}) (string, error) {
	var outputBuf bytes.Buffer
	outputBuf.WriteString("[")
	if len(value) > 0 {
		outputBuf.WriteString("\n")
	}

	lastIdx := len(value) - 1
	for i, v := range value {
		raw, err := formatResult(v)
		if err != nil {
			return "", err
		}

		outputBuf.WriteString(indent(raw))
		if lastIdx != i {
			outputBuf.WriteString(",")
		}
		outputBuf.WriteString("\n")
	}

	outputBuf.WriteString("]")
	return outputBuf.String(), nil
}

func formatMapResult(value map[string]interface{}) (string, error) {
	ks := make([]string, 0, len(value))
	for k, _ := range value {
		ks = append(ks, k)
	}
	sort.Strings(ks)

	var outputBuf bytes.Buffer
	outputBuf.WriteString("{")
	if len(value) > 0 {
		outputBuf.WriteString("\n")
	}

	for _, k := range ks {
		v := value[k]
		raw, err := formatResult(v)
		if err != nil {
			return "", err
		}

		outputBuf.WriteString(indent(fmt.Sprintf("%s = %v\n", k, raw)))
	}

	outputBuf.WriteString("}")
	return outputBuf.String(), nil
}

func indent(value string) string {
	var outputBuf bytes.Buffer
	s := bufio.NewScanner(strings.NewReader(value))
	for s.Scan() {
		outputBuf.WriteString("  " + s.Text())
	}

	return outputBuf.String()
}
