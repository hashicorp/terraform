package command

// Marshal an object as an hcl value.
import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/hashicorp/hcl/hcl/printer"
)

// This will only work operate on []interface{}, map[string]interface{}, and
// primitive types.
func encodeHCL(i interface{}) ([]byte, error) {

	state := &encodeState{}
	err := state.encode(i)
	if err != nil {
		return nil, err
	}

	hcl := state.Bytes()
	if len(hcl) == 0 {
		return hcl, nil
	}

	// the HCL parser requires an assignment. Strip it off again later
	fakeAssignment := append([]byte("X = "), hcl...)

	// use the real hcl parser to verify our output, and format it canonically
	hcl, err = printer.Format(fakeAssignment)

	// now strip that first assignment off
	eq := regexp.MustCompile(`=\s+`).FindIndex(hcl)
	return hcl[eq[1]:], err
}

type encodeState struct {
	bytes.Buffer
}

func (e *encodeState) encode(i interface{}) error {
	switch v := i.(type) {
	case []interface{}:
		return e.encodeList(v)

	case map[string]interface{}:
		return e.encodeMap(v)

	case int, int8, int32, int64, uint8, uint32, uint64:
		return e.encodeInt(i)

	case float32, float64:
		return e.encodeFloat(i)

	case string:
		return e.encodeString(v)

	case nil:
		return nil

	default:
		return fmt.Errorf("invalid type %T", i)
	}

}

func (e *encodeState) encodeList(l []interface{}) error {
	e.WriteString("[")
	for i, v := range l {
		err := e.encode(v)
		if err != nil {
			return err
		}
		if i < len(l)-1 {
			e.WriteString(", ")
		}
	}
	e.WriteString("]")
	return nil
}

func (e *encodeState) encodeMap(m map[string]interface{}) error {
	e.WriteString("{\n")
	for i, k := range sortedKeys(m) {
		v := m[k]

		e.WriteString(k + " = ")
		err := e.encode(v)
		if err != nil {
			return err
		}
		if i < len(m)-1 {
			e.WriteString("\n")
		}
	}
	e.WriteString("}")
	return nil
}

func (e *encodeState) encodeString(s string) error {
	_, err := fmt.Fprintf(e, "%q", s)
	return err
}

func (e *encodeState) encodeInt(i interface{}) error {
	_, err := fmt.Fprintf(e, "%d", i)
	return err
}

func (e *encodeState) encodeFloat(f interface{}) error {
	_, err := fmt.Fprintf(e, "%f", f)
	return err
}
