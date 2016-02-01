package autorest

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
)

// EncodedAs is a series of constants specifying various data encodings
type EncodedAs string

const (
	// EncodedAsJSON states that data is encoded as JSON
	EncodedAsJSON EncodedAs = "JSON"

	// EncodedAsXML states that data is encoded as Xml
	EncodedAsXML EncodedAs = "XML"
)

// Decoder defines the decoding method json.Decoder and xml.Decoder share
type Decoder interface {
	Decode(v interface{}) error
}

// NewDecoder creates a new decoder appropriate to the passed encoding.
// encodedAs specifies the type of encoding and r supplies the io.Reader containing the
// encoded data.
func NewDecoder(encodedAs EncodedAs, r io.Reader) Decoder {
	if encodedAs == EncodedAsJSON {
		return json.NewDecoder(r)
	} else if encodedAs == EncodedAsXML {
		return xml.NewDecoder(r)
	}
	return nil
}

// CopyAndDecode decodes the data from the passed io.Reader while making a copy. Having a copy
// is especially useful if there is a chance the data will fail to decode.
// encodedAs specifies the expected encoding, r provides the io.Reader to the data, and v
// is the decoding destination.
func CopyAndDecode(encodedAs EncodedAs, r io.Reader, v interface{}) (bytes.Buffer, error) {
	b := bytes.Buffer{}
	return b, NewDecoder(encodedAs, io.TeeReader(r, &b)).Decode(v)
}

func containsInt(ints []int, n int) bool {
	for _, i := range ints {
		if i == n {
			return true
		}
	}
	return false
}

func escapeValueStrings(m map[string]string) map[string]string {
	for key, value := range m {
		m[key] = url.QueryEscape(value)
	}
	return m
}

func ensureValueStrings(mapOfInterface map[string]interface{}) map[string]string {
	mapOfStrings := make(map[string]string)
	for key, value := range mapOfInterface {
		mapOfStrings[key] = ensureValueString(value)
	}
	return mapOfStrings
}

func ensureValueString(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
