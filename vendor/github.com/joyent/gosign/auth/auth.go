//
// gosign - Go HTTP signing library for the Joyent Public Cloud and Joyent Manta
//
//
// Copyright (c) 2013 Joyent Inc.
//
// Written by Daniele Stroppa <daniele.stroppa@joyent.com>
//

package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	// Authorization Headers
	SdcSignature   = "Signature keyId=\"/%s/keys/%s\",algorithm=\"%s\" %s"
	MantaSignature = "Signature keyId=\"/%s/keys/%s\",algorithm=\"%s\",signature=\"%s\""
)

type Endpoint struct {
	URL string
}

type Auth struct {
	User      string
	PrivateKey PrivateKey
	Algorithm string
}

type Credentials struct {
	UserAuthentication *Auth
	SdcKeyId           string
	SdcEndpoint        Endpoint
	MantaKeyId         string
	MantaEndpoint      Endpoint
}

type PrivateKey struct {
	key *rsa.PrivateKey
}

// NewAuth creates a new Auth.
func NewAuth(user, privateKey, algorithm string) (*Auth, error) {
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return nil, fmt.Errorf("invalid private key data: %s", privateKey)
	}
	rsakey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("An error occurred while parsing the key: %s", err)
	}
	return &Auth{user, PrivateKey{rsakey}, algorithm}, nil
}

// The CreateAuthorizationHeader returns the Authorization header for the give request.
func CreateAuthorizationHeader(headers http.Header, credentials *Credentials, isMantaRequest bool) (string, error) {
	if isMantaRequest {
		signature, err := GetSignature(credentials.UserAuthentication, "date: "+headers.Get("Date"))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(MantaSignature, credentials.UserAuthentication.User, credentials.MantaKeyId,
			credentials.UserAuthentication.Algorithm, signature), nil
	}
	signature, err := GetSignature(credentials.UserAuthentication, headers.Get("Date"))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(SdcSignature, credentials.UserAuthentication.User, credentials.SdcKeyId,
		credentials.UserAuthentication.Algorithm, signature), nil
}

// The GetSignature method signs the specified key according to http://apidocs.joyent.com/cloudapi/#issuing-requests
// and http://apidocs.joyent.com/manta/api.html#authentication.
func GetSignature(auth *Auth, signing string) (string, error) {
	hashFunc := getHashFunction(auth.Algorithm)
	hash := hashFunc.New()
	hash.Write([]byte(signing))

	digest := hash.Sum(nil)

	signed, err := rsa.SignPKCS1v15(rand.Reader, auth.PrivateKey.key, hashFunc, digest)
	if err != nil {
		return "", fmt.Errorf("An error occurred while signing the key: %s", err)
	}

	return base64.StdEncoding.EncodeToString(signed), nil
}

// Helper method to get the Hash function based on the algorithm
func getHashFunction(algorithm string) (hashFunc crypto.Hash) {
	switch strings.ToLower(algorithm) {
	case "rsa-sha1":
		hashFunc = crypto.SHA1
	case "rsa-sha224", "rsa-sha256":
		hashFunc = crypto.SHA256
	case "rsa-sha384", "rsa-sha512":
		hashFunc = crypto.SHA512
	default:
		hashFunc = crypto.SHA256
	}
	return
}

func (cred *Credentials) Region() string {
	parsedUrl, err := url.Parse(cred.SdcEndpoint.URL)
	if err != nil {
		// Bogus URL - no region.
		return ""
	}
	if strings.HasPrefix(parsedUrl.Host, "localhost") || strings.HasPrefix(parsedUrl.Host, "127.0.0.1") {
		return "some-region"
	}

	host := parsedUrl.Host
	firstDotIdx := strings.Index(host, ".")
	if firstDotIdx >= 0 {
		return host[:firstDotIdx]
	}
	return host
}
