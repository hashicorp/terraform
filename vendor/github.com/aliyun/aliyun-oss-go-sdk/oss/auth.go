package oss

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"hash"
	"io"
	"net/http"
	"sort"
	"strings"
)

// headerSorter defines the key-value structure for storing the sorted data in signHeader.
type headerSorter struct {
	Keys []string
	Vals []string
}

// signHeader signs the header and sets it as the authorization header.
func (conn Conn) signHeader(req *http.Request, canonicalizedResource string) {
	// Get the final authorization string
	authorizationStr := "OSS " + conn.config.AccessKeyID + ":" + conn.getSignedStr(req, canonicalizedResource)

	// Give the parameter "Authorization" value
	req.Header.Set(HTTPHeaderAuthorization, authorizationStr)
}

func (conn Conn) getSignedStr(req *http.Request, canonicalizedResource string) string {
	// Find out the "x-oss-"'s address in header of the request
	temp := make(map[string]string)

	for k, v := range req.Header {
		if strings.HasPrefix(strings.ToLower(k), "x-oss-") {
			temp[strings.ToLower(k)] = v[0]
		}
	}
	hs := newHeaderSorter(temp)

	// Sort the temp by the ascending order
	hs.Sort()

	// Get the canonicalizedOSSHeaders
	canonicalizedOSSHeaders := ""
	for i := range hs.Keys {
		canonicalizedOSSHeaders += hs.Keys[i] + ":" + hs.Vals[i] + "\n"
	}

	// Give other parameters values
	// when sign URL, date is expires
	date := req.Header.Get(HTTPHeaderDate)
	contentType := req.Header.Get(HTTPHeaderContentType)
	contentMd5 := req.Header.Get(HTTPHeaderContentMD5)

	signStr := req.Method + "\n" + contentMd5 + "\n" + contentType + "\n" + date + "\n" + canonicalizedOSSHeaders + canonicalizedResource
	h := hmac.New(func() hash.Hash { return sha1.New() }, []byte(conn.config.AccessKeySecret))
	io.WriteString(h, signStr)
	signedStr := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signedStr
}

// newHeaderSorter is an additional function for function SignHeader.
func newHeaderSorter(m map[string]string) *headerSorter {
	hs := &headerSorter{
		Keys: make([]string, 0, len(m)),
		Vals: make([]string, 0, len(m)),
	}

	for k, v := range m {
		hs.Keys = append(hs.Keys, k)
		hs.Vals = append(hs.Vals, v)
	}
	return hs
}

// Sort is an additional function for function SignHeader.
func (hs *headerSorter) Sort() {
	sort.Sort(hs)
}

// Len is an additional function for function SignHeader.
func (hs *headerSorter) Len() int {
	return len(hs.Vals)
}

// Less is an additional function for function SignHeader.
func (hs *headerSorter) Less(i, j int) bool {
	return bytes.Compare([]byte(hs.Keys[i]), []byte(hs.Keys[j])) < 0
}

// Swap is an additional function for function SignHeader.
func (hs *headerSorter) Swap(i, j int) {
	hs.Vals[i], hs.Vals[j] = hs.Vals[j], hs.Vals[i]
	hs.Keys[i], hs.Keys[j] = hs.Keys[j], hs.Keys[i]
}
