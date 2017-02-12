package stun

import "reflect"

// Error codes introduced by the RFC 5389 Section 15.6
const (
	CodeTryAlternate     = 300
	CodeBadRequest       = 400
	CodeUnauthorized     = 401
	CodeUnknownAttribute = 420
	CodeStaleNonce       = 438
	CodeServerError      = 500
)

// Error codes introduced by the RFC 3489 Section 11.2.9 except listed in RFC 5389.
const (
	CodeStaleCredentials      = 430
	CodeIntegrityCheckFailure = 431
	CodeMissingUsername       = 432
	CodeUseTLS                = 433
	CodeGlobalFailure         = 600
)

var errorText = map[int]string{
	CodeTryAlternate:          "Try Alternate",
	CodeBadRequest:            "Bad Request",
	CodeUnauthorized:          "Unauthorized",
	CodeUnknownAttribute:      "Unknown Attribute",
	CodeStaleCredentials:      "Stale Credentials",
	CodeIntegrityCheckFailure: "Integrity Check Failure",
	CodeMissingUsername:       "Missing Username",
	CodeUseTLS:                "Use TLS",
	CodeStaleNonce:            "Stale Nonce",
	CodeServerError:           "Server Error",
	CodeGlobalFailure:         "Global Failure",
}

// ErrorText returns a reason phrase text for the STUN error code. It returns the empty string if the code is unknown.
func ErrorText(code int) string {
	return errorText[code]
}

// Error represents the ERROR-CODE attribute.
type Error struct {
	Code   int
	Reason string
}

// NewError returns Error with code and default reason phrase.
func NewError(code int) *Error {
	return &Error{Code: code, Reason: ErrorText(code)}
}

type errorCodec struct{}

func (c errorCodec) Encode(w Writer, v interface{}) error {
	switch attr := v.(type) {
	case *Error:
		c.writeErrorCode(w, attr.Code, attr.Reason)
	default:
		return &errUnsupportedAttrType{Type: reflect.TypeOf(v)}
	}
	return nil
}

func (c errorCodec) writeErrorCode(w Writer, code int, reason string) {
	b := w.Next(4 + len(reason))
	b[0] = 0
	b[1] = 0
	b[2] = byte(code / 100)
	b[3] = byte(code % 100)
	copy(b[4:], reason)
}

func (c errorCodec) Decode(r Reader) (interface{}, error) {
	b, err := r.Next(4)
	if err != nil {
		return nil, err
	}
	code := int(b[2])*100 + int(b[3])
	b, _ = r.Next(r.Available())
	return &Error{code, string(b)}, nil
}
