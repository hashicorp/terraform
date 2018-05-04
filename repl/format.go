package repl

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// FormatResult formats the given result value for human-readable output.
//
// The value must currently be a string, list, map, and any nested values
// with those same types.
func FormatResult(value interface{}) (string, error) {
	return formatResult(value, false)
}

func formatResult(value interface{}, nested bool) (string, error) {
	if value == nil {
		return "null", nil
	}
	switch output := value.(type) {
	case string:
		if nested {
			return fmt.Sprintf("%q", output), nil
		}
		return output, nil
	case int:
		return strconv.Itoa(output), nil
	case float64:
		return fmt.Sprintf("%g", output), nil
	case bool:
		switch {
		case output == true:
			return "true", nil
		default:
			return "false", nil
		}
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

	for _, v := range value {
		raw, err := formatResult(v, true)
		if err != nil {
			return "", err
		}

		outputBuf.WriteString(indent(raw))
		outputBuf.WriteString(",\n")
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
		rawK, err := formatResult(k, true)
		if err != nil {
			return "", err
		}
		rawV, err := formatResult(v, true)
		if err != nil {
			return "", err
		}

		outputBuf.WriteString(indent(fmt.Sprintf("%s = %s", rawK, rawV)))
		outputBuf.WriteString("\n")
	}

	outputBuf.WriteString("}")
	return outputBuf.String(), nil
}

func indent(value string) string {
	var outputBuf bytes.Buffer
	s := bufio.NewScanner(strings.NewReader(value))
	newline := false
	for s.Scan() {
		if newline {
			outputBuf.WriteByte('\n')
		}
		outputBuf.WriteString("  " + s.Text())
		newline = true
	}

	return outputBuf.String()
}
