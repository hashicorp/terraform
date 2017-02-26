package dataurl

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// Escape implements URL escaping, as defined in RFC 2397 (http://tools.ietf.org/html/rfc2397).
// It differs a bit from net/url's QueryEscape and QueryUnescape, e.g how spaces are treated (+ instead of %20):
//
// Only ASCII chars are allowed. Reserved chars are escaped to their %xx form.
// Unreserved chars are [a-z], [A-Z], [0-9], and -_.!~*\().
func Escape(data []byte) string {
	var buf = new(bytes.Buffer)
	for _, b := range data {
		switch {
		case isUnreserved(b):
			buf.WriteByte(b)
		default:
			fmt.Fprintf(buf, "%%%02X", b)
		}
	}
	return buf.String()
}

// EscapeString is like Escape, but taking
// a string as argument.
func EscapeString(s string) string {
	return Escape([]byte(s))
}

// isUnreserved return true
// if the byte c is an unreserved char,
// as defined in RFC 2396.
func isUnreserved(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' ||
		c == '_' ||
		c == '.' ||
		c == '!' ||
		c == '~' ||
		c == '*' ||
		c == '\'' ||
		c == '(' ||
		c == ')'
}

func isHex(c byte) bool {
	switch {
	case c >= 'a' && c <= 'f':
		return true
	case c >= 'A' && c <= 'F':
		return true
	case c >= '0' && c <= '9':
		return true
	}
	return false
}

// borrowed from net/url/url.go
func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// Unescape unescapes a character sequence
// escaped with Escape(String?).
func Unescape(s string) ([]byte, error) {
	var buf = new(bytes.Buffer)
	reader := strings.NewReader(s)

	for {
		r, size, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if size > 1 {
			return nil, fmt.Errorf("rfc2396: non-ASCII char detected")
		}

		switch r {
		case '%':
			eb1, err := reader.ReadByte()
			if err == io.EOF {
				return nil, fmt.Errorf("rfc2396: unexpected end of unescape sequence")
			}
			if err != nil {
				return nil, err
			}
			if !isHex(eb1) {
				return nil, fmt.Errorf("rfc2396: invalid char 0x%x in unescape sequence", r)
			}
			eb0, err := reader.ReadByte()
			if err == io.EOF {
				return nil, fmt.Errorf("rfc2396: unexpected end of unescape sequence")
			}
			if err != nil {
				return nil, err
			}
			if !isHex(eb0) {
				return nil, fmt.Errorf("rfc2396: invalid char 0x%x in unescape sequence", r)
			}
			buf.WriteByte(unhex(eb0) + unhex(eb1)*16)
		default:
			buf.WriteByte(byte(r))
		}
	}
	return buf.Bytes(), nil
}

// UnescapeToString is like Unescape, but returning
// a string.
func UnescapeToString(s string) (string, error) {
	b, err := Unescape(s)
	return string(b), err
}
