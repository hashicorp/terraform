package cos

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"net/http"
)

// 计算 md5 或 sha1 时的分块大小
const calDigestBlockSize = 1024 * 1024 * 10

func calMD5Digest(msg []byte) []byte {
	// TODO: 分块计算,减少内存消耗
	m := md5.New()
	m.Write(msg)
	return m.Sum(nil)
}

func calSHA1Digest(msg []byte) []byte {
	// TODO: 分块计算,减少内存消耗
	m := sha1.New()
	m.Write(msg)
	return m.Sum(nil)
}

// cloneRequest returns a clone of the provided *http.Request. The clone is a
// shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	return r2
}

// encodeURIComponent like same function in javascript
//
// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/encodeURIComponent
//
// http://www.ecma-international.org/ecma-262/6.0/#sec-uri-syntax-and-semantics
func encodeURIComponent(s string) string {
	var b bytes.Buffer
	written := 0

	for i, n := 0, len(s); i < n; i++ {
		c := s[i]

		switch c {
		case '-', '_', '.', '!', '~', '*', '\'', '(', ')':
			continue
		default:
			// Unreserved according to RFC 3986 sec 2.3
			if 'a' <= c && c <= 'z' {

				continue

			}
			if 'A' <= c && c <= 'Z' {

				continue

			}
			if '0' <= c && c <= '9' {

				continue
			}
		}

		b.WriteString(s[written:i])
		fmt.Fprintf(&b, "%%%02X", c)
		written = i + 1
	}

	if written == 0 {
		return s
	}
	b.WriteString(s[written:])
	return b.String()
}
