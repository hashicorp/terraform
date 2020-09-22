package tablestore

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"hash"
	"sort"
	"strings"
)

const (
	xOtsDate                = "x-ots-date"
	xOtsApiversion          = "x-ots-apiversion"
	xOtsAccesskeyid         = "x-ots-accesskeyid"
	xOtsContentmd5          = "x-ots-contentmd5"
	xOtsHeaderStsToken      = "x-ots-ststoken"
	xOtsSignature           = "x-ots-signature"
	xOtsRequestCompressType = "x-ots-request-compress-type"
	xOtsRequestCompressSize = "x-ots-request-compress-size"
	xOtsResponseCompressTye = "x-ots-response-compress-type"
)

type otsHeader struct {
	name  string
	value string
	must  bool
}

type otsHeaders struct {
	headers  []*otsHeader
	hmacSha1 hash.Hash
}

func createOtsHeaders(accessKey string) *otsHeaders {
	h := new(otsHeaders)

	h.headers = []*otsHeader{
		&otsHeader{name: xOtsDate, must: true},
		&otsHeader{name: xOtsApiversion, must: true},
		&otsHeader{name: xOtsAccesskeyid, must: true},
		&otsHeader{name: xOtsContentmd5, must: true},
		&otsHeader{name: xOtsInstanceName, must: true},
		&otsHeader{name: xOtsSignature, must: true},
		&otsHeader{name: xOtsRequestCompressSize, must: false},
		&otsHeader{name: xOtsResponseCompressTye, must: false},
		&otsHeader{name: xOtsRequestCompressType, must: false},
		&otsHeader{name: xOtsHeaderStsToken, must: false},
	}

	sort.Sort(h)

	h.hmacSha1 = hmac.New(sha1.New, []byte(accessKey))
	return h
}

func (h *otsHeaders) Len() int {
	return len(h.headers)
}

func (h *otsHeaders) Swap(i, j int) {
	h.headers[i], h.headers[j] = h.headers[j], h.headers[i]
}

func (h *otsHeaders) Less(i, j int) bool {
	if h.headers[i].name == xOtsSignature {
		return false
	}

	if h.headers[j].name == xOtsSignature {
		return true
	}

	return h.headers[i].name < h.headers[j].name
}

func (h *otsHeaders) search(name string) *otsHeader {
	index := sort.Search(len(h.headers)-1, func(i int) bool {
		return h.headers[i].name >= name
	})

	if index >= len(h.headers) {
		return nil
	}

	return h.headers[index]
}

func (h *otsHeaders) set(name, value string) {
	header := h.search(name)
	if header == nil {
		return
	}

	header.value = value
}

func (h *otsHeaders) signature(uri, method, accessKey string) (string, error) {
	for _, header := range h.headers[:len(h.headers)-1] {
		if header.must && header.value == "" {
			return "", errMissMustHeader(header.name)
		}
	}

	// StringToSign = CanonicalURI + '\n' + HTTPRequestMethod + '\n' + CanonicalQueryString + '\n' + CanonicalHeaders + '\n'
	// TODO CanonicalQueryString 为空
	stringToSign := uri + "\n" + method + "\n" + "\n"

	// 最后一个header 为 xOtsSignature
	for _, header := range h.headers[:len(h.headers)-1] {
		if header.value != "" {
			stringToSign = stringToSign + header.name + ":" + strings.TrimSpace(header.value) + "\n"
		}
	}

	h.hmacSha1.Reset()
	h.hmacSha1.Write([]byte(stringToSign))

	// fmt.Println("stringToSign:" + stringToSign)
	sign := base64.StdEncoding.EncodeToString(h.hmacSha1.Sum(nil))
	h.set(xOtsSignature, sign)
	// fmt.Println("sign:" + sign)
	return sign, nil
}
