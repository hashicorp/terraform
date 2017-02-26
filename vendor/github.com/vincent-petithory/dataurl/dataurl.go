package dataurl

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

const (
	// EncodingBase64 is base64 encoding for the data url
	EncodingBase64 = "base64"
	// EncodingASCII is ascii encoding for the data url
	EncodingASCII = "ascii"
)

func defaultMediaType() MediaType {
	return MediaType{
		"text",
		"plain",
		map[string]string{"charset": "US-ASCII"},
	}
}

// MediaType is the combination of a media type, a media subtype
// and optional parameters.
type MediaType struct {
	Type    string
	Subtype string
	Params  map[string]string
}

// ContentType returns the content type of the dataurl's data, in the form type/subtype.
func (mt *MediaType) ContentType() string {
	return fmt.Sprintf("%s/%s", mt.Type, mt.Subtype)
}

// String implements the Stringer interface.
//
// Params values are escaped with the Escape function, rather than in a quoted string.
func (mt *MediaType) String() string {
	var buf bytes.Buffer
	for k, v := range mt.Params {
		fmt.Fprintf(&buf, ";%s=%s", k, EscapeString(v))
	}
	return mt.ContentType() + (&buf).String()
}

// DataURL is the combination of a MediaType describing the type of its Data.
type DataURL struct {
	MediaType
	Encoding string
	Data     []byte
}

// New returns a new DataURL initialized with data and
// a MediaType parsed from mediatype and paramPairs.
// mediatype must be of the form "type/subtype" or it will panic.
// paramPairs must have an even number of elements or it will panic.
// For more complex DataURL, initialize a DataURL struct.
// The DataURL is initialized with base64 encoding.
func New(data []byte, mediatype string, paramPairs ...string) *DataURL {
	parts := strings.Split(mediatype, "/")
	if len(parts) != 2 {
		panic("dataurl: invalid mediatype")
	}

	nParams := len(paramPairs)
	if nParams%2 != 0 {
		panic("dataurl: requires an even number of param pairs")
	}
	params := make(map[string]string)
	for i := 0; i < nParams; i += 2 {
		params[paramPairs[i]] = paramPairs[i+1]
	}

	mt := MediaType{
		parts[0],
		parts[1],
		params,
	}
	return &DataURL{
		MediaType: mt,
		Encoding:  EncodingBase64,
		Data:      data,
	}
}

// String implements the Stringer interface.
//
// Note: it doesn't guarantee the returned string is equal to
// the initial source string that was used to create this DataURL.
// The reasons for that are:
//  * Insertion of default values for MediaType that were maybe not in the initial string,
//  * Various ways to encode the MediaType parameters (quoted string or url encoded string, the latter is used),
func (du *DataURL) String() string {
	var buf bytes.Buffer
	du.WriteTo(&buf)
	return (&buf).String()
}

// WriteTo implements the WriterTo interface.
// See the note about String().
func (du *DataURL) WriteTo(w io.Writer) (n int64, err error) {
	var ni int
	ni, _ = fmt.Fprint(w, "data:")
	n += int64(ni)

	ni, _ = fmt.Fprint(w, du.MediaType.String())
	n += int64(ni)

	if du.Encoding == EncodingBase64 {
		ni, _ = fmt.Fprint(w, ";base64")
		n += int64(ni)
	}

	ni, _ = fmt.Fprint(w, ",")
	n += int64(ni)

	if du.Encoding == EncodingBase64 {
		encoder := base64.NewEncoder(base64.StdEncoding, w)
		ni, err = encoder.Write(du.Data)
		if err != nil {
			return
		}
		encoder.Close()
	} else if du.Encoding == EncodingASCII {
		ni, _ = fmt.Fprint(w, Escape(du.Data))
		n += int64(ni)
	} else {
		err = fmt.Errorf("dataurl: invalid encoding %s", du.Encoding)
		return
	}

	return
}

// UnmarshalText decodes a Data URL string and sets it to *du
func (du *DataURL) UnmarshalText(text []byte) error {
	decoded, err := DecodeString(string(text))
	if err != nil {
		return err
	}
	*du = *decoded
	return nil
}

// MarshalText writes du as a Data URL
func (du *DataURL) MarshalText() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if _, err := du.WriteTo(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type encodedDataReader func(string) ([]byte, error)

var asciiDataReader encodedDataReader = func(s string) ([]byte, error) {
	us, err := Unescape(s)
	if err != nil {
		return nil, err
	}
	return []byte(us), nil
}

var base64DataReader encodedDataReader = func(s string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return []byte(data), nil
}

type parser struct {
	du                  *DataURL
	l                   *lexer
	currentAttr         string
	unquoteParamVal     bool
	encodedDataReaderFn encodedDataReader
}

func (p *parser) parse() error {
	for item := range p.l.items {
		switch item.t {
		case itemError:
			return errors.New(item.String())
		case itemMediaType:
			p.du.MediaType.Type = item.val
			// Should we clear the default
			// "charset" parameter at this point?
			delete(p.du.MediaType.Params, "charset")
		case itemMediaSubType:
			p.du.MediaType.Subtype = item.val
		case itemParamAttr:
			p.currentAttr = item.val
		case itemLeftStringQuote:
			p.unquoteParamVal = true
		case itemParamVal:
			val := item.val
			if p.unquoteParamVal {
				p.unquoteParamVal = false
				us, err := strconv.Unquote("\"" + val + "\"")
				if err != nil {
					return err
				}
				val = us
			} else {
				us, err := UnescapeToString(val)
				if err != nil {
					return err
				}
				val = us
			}
			p.du.MediaType.Params[p.currentAttr] = val
		case itemBase64Enc:
			p.du.Encoding = EncodingBase64
			p.encodedDataReaderFn = base64DataReader
		case itemDataComma:
			if p.encodedDataReaderFn == nil {
				p.encodedDataReaderFn = asciiDataReader
			}
		case itemData:
			reader, err := p.encodedDataReaderFn(item.val)
			if err != nil {
				return err
			}
			p.du.Data = reader
		case itemEOF:
			if p.du.Data == nil {
				p.du.Data = []byte("")
			}
			return nil
		}
	}
	panic("EOF not found")
}

// DecodeString decodes a Data URL scheme string.
func DecodeString(s string) (*DataURL, error) {
	du := &DataURL{
		MediaType: defaultMediaType(),
		Encoding:  EncodingASCII,
	}

	parser := &parser{
		du: du,
		l:  lex(s),
	}
	if err := parser.parse(); err != nil {
		return nil, err
	}
	return du, nil
}

// Decode decodes a Data URL scheme from a io.Reader.
func Decode(r io.Reader) (*DataURL, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return DecodeString(string(data))
}

// EncodeBytes encodes the data bytes into a Data URL string, using base 64 encoding.
//
// The media type of data is detected using http.DetectContentType.
func EncodeBytes(data []byte) string {
	mt := http.DetectContentType(data)
	// http.DetectContentType may add spurious spaces between ; and a parameter.
	// The canonical way is to not have them.
	cleanedMt := strings.Replace(mt, "; ", ";", -1)

	return New(data, cleanedMt).String()
}
