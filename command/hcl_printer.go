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
	if err != nil {
		return nil, err
	}

	// now strip that first assignment off
	eq := regexp.MustCompile(`=\s+`).FindIndex(hcl)

	// strip of an extra \n if it's there
	end := len(hcl)
	if hcl[end-1] == '\n' {
		end -= 1
	}

	return hcl[eq[1]:end], nil
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

		e.WriteString(fmt.Sprintf("%q = ", k))
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

func (e *encodeState) encodeInt(i interface{}) error {
	_, err := fmt.Fprintf(e, "%d", i)
	return err
}

func (e *encodeState) encodeFloat(f interface{}) error {
	_, err := fmt.Fprintf(e, "%g", f)
	return err
}

func (e *encodeState) encodeString(s string) error {
	e.Write(quoteHCLString(s))
	return nil
}

// Quote an HCL string, which may contain interpolations.
// Since the string was already parsed from HCL, we have to assume the
// required characters are sanely escaped. All we need to do is escape double
// quotes in the string, unless they are in an interpolation block.
func quoteHCLString(s string) []byte {
	out := make([]byte, 0, len(s))
	out = append(out, '"')

	// our parse states
	var (
		outer  = 1 // the starting state for the string
		dollar = 2 // look for '{' in the next character
		interp = 3 // inside an interpolation block
		escape = 4 // take the next character and pop back to prev state
	)

	// we could have nested interpolations
	state := stack{}
	state.push(outer)

	for i := 0; i < len(s); i++ {
		switch state.peek() {
		case outer:
			switch s[i] {
			case '"':
				out = append(out, '\\')
			case '$':
				state.push(dollar)
			case '\\':
				state.push(escape)
			}
		case dollar:
			state.pop()
			switch s[i] {
			case '{':
				state.push(interp)
			case '\\':
				state.push(escape)
			}
		case interp:
			switch s[i] {
			case '}':
				state.pop()
			}
		case escape:
			state.pop()
		}

		out = append(out, s[i])
	}

	out = append(out, '"')

	return out
}

type stack []int

func (s *stack) push(i int) {
	*s = append(*s, i)
}

func (s *stack) pop() int {
	last := len(*s) - 1
	i := (*s)[last]
	*s = (*s)[:last]
	return i
}

func (s *stack) peek() int {
	return (*s)[len(*s)-1]
}
