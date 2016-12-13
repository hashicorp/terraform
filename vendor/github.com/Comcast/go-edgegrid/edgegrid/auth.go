package edgegrid

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/satori/go.uuid"
)

// AuthParams is used to house various request details such that
// the AuthParams can be passed to Auth to sign using the
// Akamai {OPEN} EdgeGrid Authentication scheme.
type AuthParams struct {
	req           *http.Request
	clientToken   string
	accessToken   string
	clientSecret  string
	timestamp     string
	nonce         string
	headersToSign []string
}

// NewAuthParams returns an AuthParams generated from req, accessToken,
// clientToken, and clientSecret.
func NewAuthParams(req *http.Request, accessToken, clientToken, clientSecret string) AuthParams {
	return AuthParams{
		req,
		clientToken,
		accessToken,
		clientSecret,
		time.Now().UTC().Format("20060102T15:04:05+0000"),
		uuid.NewV4().String(),
		[]string{},
	}
}

// Auth takes prm and returns a string that can be
// used as the `Authorization` header in making Akamai API requests.
//
// The string returned by Auth conforms to the
// Akamai {OPEN} EdgeGrid Authentication scheme.
// https://developer.akamai.com/introduction/Client_Auth.html
func Auth(prm AuthParams) string {
	var auth bytes.Buffer
	orderedKeys := []string{"client_token", "access_token", "timestamp", "nonce"}
	timestamp := prm.timestamp

	m := map[string]string{
		orderedKeys[0]: prm.clientToken,
		orderedKeys[1]: prm.accessToken,
		orderedKeys[2]: timestamp,
		orderedKeys[3]: prm.nonce,
	}

	auth.WriteString("EG1-HMAC-SHA256 ")

	for _, each := range orderedKeys {
		auth.WriteString(concat([]string{
			each,
			"=",
			m[each],
			";",
		}))
	}

	auth.WriteString(signRequest(prm.req, timestamp, prm.clientSecret, auth.String(), prm.headersToSign))

	return auth.String()
}

func signRequest(request *http.Request, timestamp, clientSecret, authHeader string, headersToSign []string) string {
	dataToSign := makeDataToSign(request, authHeader, headersToSign)
	signingKey := makeSigningKey(timestamp, clientSecret)

	return concat([]string{
		"signature=",
		base64HmacSha256(dataToSign, signingKey),
	})
}

func base64Sha256(str string) string {
	h := sha256.New()

	h.Write([]byte(str))

	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func base64HmacSha256(message, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))

	h.Write([]byte(message))

	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func makeDataToSign(request *http.Request, authHeader string, headersToSign []string) string {
	var data bytes.Buffer
	values := []string{
		request.Method,
		request.URL.Scheme,
		request.Host,
		urlPathWithQuery(request),
		canonicalizeHeaders(request, headersToSign),
		makeContentHash(request),
		authHeader,
	}

	data.WriteString(strings.Join(values, "\t"))

	return data.String()
}

func canonicalizeHeaders(request *http.Request, headersToSign []string) string {
	var canonicalized bytes.Buffer

	for key, values := range request.Header {
		if stringInSlice(key, headersToSign) {
			canonicalized.WriteString(concat([]string{
				strings.ToLower(key),
				":",
				strings.Join(strings.Fields(values[0]), " "),
				"\t",
			}))
		}
	}

	return canonicalized.String()
}

func makeContentHash(req *http.Request) string {
	if req.Method == "POST" {
		buf, err := ioutil.ReadAll(req.Body)
		rdr := reader{bytes.NewBuffer(buf)}

		if err != nil {
			panic(err)
		}

		req.Body = rdr

		return base64Sha256(string(buf))
	}

	return ""
}

func makeSigningKey(timestamp, clientSecret string) string {
	return base64HmacSha256(timestamp, clientSecret)
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if strings.ToLower(b) == strings.ToLower(a) {
			return true
		}
	}

	return false
}
