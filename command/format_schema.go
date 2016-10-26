package command

import (
	"bytes"
	"fmt"
	"strings"

	"encoding/json"
	"encoding/xml"
	"github.com/mitchellh/colorstring"
)

// FormatSchemaOpts are the options for formatting a schema.
type FormatSchemaOpts struct {
	// Name of schema to format. This is required.
	Name string

	// Schema is the schema to format. This is required.
	Schema *interface{}

	// Output format. Either 'json' or 'xml'. This is required.
	Format string

	// Is output should be indented. This is optional.
	Indent bool

	// This is optional.
	Colorize bool

	// Colorizer is the colorizer. This is required only if Colorize == true.
	Colorizer *colorstring.Colorize
}

// FormatState takes a state and returns a string
func FormatSchema(opts *FormatSchemaOpts) string {
	if opts.Colorize && opts.Colorizer == nil {
		panic("Colorizer not given")
	}

	s := opts.Schema

	var buf bytes.Buffer
	//	buf.WriteString("[reset]")

	var ser []byte
	var err error
	format := strings.ToLower(opts.Format)
	switch format {
	case "json":
		if opts.Indent {
			ser, err = json.MarshalIndent(s, "", "  ")
		} else {
			ser, err = json.Marshal(s)
		}
	case "xml":
		// FIXME: Use custom serializer for map[string]interface{} (SchemaInfo)
		if opts.Indent {
			ser, err = xml.MarshalIndent(s, "", "  ")
		} else {
			ser, err = xml.Marshal(s)
		}
	case "plain":
		ser, err = marshalPlain(s, opts.Indent)
	default:
		panic(fmt.Sprintf("Unsupported format %s", format))
	}

	if err != nil {
		return fmt.Sprintf("Cannot serialize schema for '%s': %s\n", opts.Name, err)
	}
	buf.Write(ser)

	trimmed := strings.TrimSpace(buf.String())
	if opts.Colorize {
		return opts.Colorizer.Color(trimmed)
	}
	return trimmed
}

func marshalPlain(in *interface{}, indent bool) ([]byte, error) {
	var b bytes.Buffer
	// TODO: Implement
	//	enc := NewEncoder(&b)
	//	enc.Indent(prefix, indent)
	//	if err := enc.Encode(v); err != nil {
	//		return nil, err
	//	}
	return b.Bytes(), nil
}
