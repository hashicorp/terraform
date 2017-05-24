package google

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"sort"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/pathorcontents"
	"github.com/hashicorp/terraform/helper/schema"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

const gcsBaseUrl = "https://storage.googleapis.com"
const googleCredentialsEnvVar = "GOOGLE_APPLICATION_CREDENTIALS"

func dataSourceGoogleSignedUrl() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGoogleSignedUrlRead,

		Schema: map[string]*schema.Schema{
			"bucket": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"content_md5": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"content_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"credentials": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"duration": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "1h",
			},
			"extension_headers": &schema.Schema{
				Type:         schema.TypeMap,
				Optional:     true,
				Elem:         schema.TypeString,
				ValidateFunc: validateExtensionHeaders,
			},
			"http_method": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "GET",
				ValidateFunc: validateHttpMethod,
			},
			"path": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"signed_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func validateExtensionHeaders(v interface{}, k string) (ws []string, errors []error) {
	hdrMap := v.(map[string]interface{})
	for k, _ := range hdrMap {
		if !strings.HasPrefix(strings.ToLower(k), "x-goog-") {
			errors = append(errors, fmt.Errorf(
				"extension_header (%s) not valid, header name must begin with 'x-goog-'", k))
		}
	}
	return
}

func validateHttpMethod(v interface{}, k string) (ws []string, errs []error) {
	value := v.(string)
	value = strings.ToUpper(value)
	if value != "GET" && value != "HEAD" && value != "PUT" && value != "DELETE" {
		errs = append(errs, errors.New("http_method must be one of [GET|HEAD|PUT|DELETE]"))
	}
	return
}

func dataSourceGoogleSignedUrlRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Build UrlData object from data source attributes
	urlData := &UrlData{}

	// HTTP Method
	if method, ok := d.GetOk("http_method"); ok {
		urlData.HttpMethod = method.(string)
	}

	// convert duration to an expiration datetime (unix time in seconds)
	durationString := "1h"
	if v, ok := d.GetOk("duration"); ok {
		durationString = v.(string)
	}
	duration, err := time.ParseDuration(durationString)
	if err != nil {
		return errwrap.Wrapf("could not parse duration: {{err}}", err)
	}
	expires := time.Now().Unix() + int64(duration.Seconds())
	urlData.Expires = int(expires)

	// content_md5 is optional
	if v, ok := d.GetOk("content_md5"); ok {
		urlData.ContentMd5 = v.(string)
	}

	// content_type is optional
	if v, ok := d.GetOk("content_type"); ok {
		urlData.ContentType = v.(string)
	}

	// extension_headers (x-goog-* HTTP headers) are optional
	if v, ok := d.GetOk("extension_headers"); ok {
		hdrMap := v.(map[string]interface{})

		if len(hdrMap) > 0 {
			urlData.HttpHeaders = make(map[string]string, len(hdrMap))
			for k, v := range hdrMap {
				urlData.HttpHeaders[k] = v.(string)
			}
		}
	}

	urlData.Path = fmt.Sprintf("/%s/%s", d.Get("bucket").(string), d.Get("path").(string))

	// Load JWT Config from Google Credentials
	jwtConfig, err := loadJwtConfig(d, config)
	if err != nil {
		return err
	}
	urlData.JwtConfig = jwtConfig

	// Construct URL
	signedUrl, err := urlData.SignedUrl()
	if err != nil {
		return err
	}

	// Success
	d.Set("signed_url", signedUrl)

	encodedSig, err := urlData.EncodedSignature()
	if err != nil {
		return err
	}
	d.SetId(encodedSig)

	return nil
}

// loadJwtConfig looks for credentials json in the following places,
// in order of preference:
//   1. `credentials` attribute of the datasource
//   2. `credentials` attribute in the provider definition.
//   3. A JSON file whose path is specified by the
//      GOOGLE_APPLICATION_CREDENTIALS environment variable.
func loadJwtConfig(d *schema.ResourceData, meta interface{}) (*jwt.Config, error) {
	config := meta.(*Config)

	credentials := ""
	if v, ok := d.GetOk("credentials"); ok {
		log.Println("[DEBUG] using data source credentials to sign URL")
		credentials = v.(string)

	} else if config.Credentials != "" {
		log.Println("[DEBUG] using provider credentials to sign URL")
		credentials = config.Credentials

	} else if filename := os.Getenv(googleCredentialsEnvVar); filename != "" {
		log.Println("[DEBUG] using env GOOGLE_APPLICATION_CREDENTIALS credentials to sign URL")
		credentials = filename

	}

	if strings.TrimSpace(credentials) != "" {
		contents, _, err := pathorcontents.Read(credentials)
		if err != nil {
			return nil, errwrap.Wrapf("Error loading credentials: {{err}}", err)
		}

		cfg, err := google.JWTConfigFromJSON([]byte(contents), "")
		if err != nil {
			return nil, errwrap.Wrapf("Error parsing credentials: {{err}}", err)
		}
		return cfg, nil
	}

	return nil, errors.New("Credentials not found in datasource, provider configuration or GOOGLE_APPLICATION_CREDENTIALS environment variable.")
}

// parsePrivateKey converts the binary contents of a private key file
// to an *rsa.PrivateKey. It detects whether the private key is in a
// PEM container or not. If so, it extracts the the private key
// from PEM container before conversion. It only supports PEM
// containers with no passphrase.
// copied from golang.org/x/oauth2/internal
func parsePrivateKey(key []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(key)
	if block != nil {
		key = block.Bytes
	}
	parsedKey, err := x509.ParsePKCS8PrivateKey(key)
	if err != nil {
		parsedKey, err = x509.ParsePKCS1PrivateKey(key)
		if err != nil {
			return nil, errwrap.Wrapf("private key should be a PEM or plain PKSC1 or PKCS8; parse error: {{err}}", err)
		}
	}
	parsed, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is invalid")
	}
	return parsed, nil
}

// UrlData stores the values required to create a Signed Url
type UrlData struct {
	JwtConfig   *jwt.Config
	ContentMd5  string
	ContentType string
	HttpMethod  string
	Expires     int
	HttpHeaders map[string]string
	Path        string
}

// SigningString creates a string representation of the UrlData in a form ready for signing:
// see https://cloud.google.com/storage/docs/access-control/create-signed-urls-program
// Example output:
// -------------------
// GET
//
//
// 1388534400
// bucket/objectname
// -------------------
func (u *UrlData) SigningString() []byte {
	var buf bytes.Buffer

	// HTTP Verb
	buf.WriteString(u.HttpMethod)
	buf.WriteString("\n")

	// Content MD5 (optional, always add new line)
	buf.WriteString(u.ContentMd5)
	buf.WriteString("\n")

	// Content Type (optional, always add new line)
	buf.WriteString(u.ContentType)
	buf.WriteString("\n")

	// Expiration
	buf.WriteString(strconv.Itoa(u.Expires))
	buf.WriteString("\n")

	// Extra HTTP headers (optional)
	// Must be sorted in lexigraphical order
	var keys []string
	for k := range u.HttpHeaders {
		keys = append(keys, strings.ToLower(k))
	}
	sort.Strings(keys)
	// Write sorted headers to signing string buffer
	for _, k := range keys {
		buf.WriteString(fmt.Sprintf("%s:%s\n", k, u.HttpHeaders[k]))
	}

	// Storate Object path (includes bucketname)
	buf.WriteString(u.Path)

	return buf.Bytes()
}

func (u *UrlData) Signature() ([]byte, error) {
	// Sign url data
	signature, err := SignString(u.SigningString(), u.JwtConfig)
	if err != nil {
		return nil, err

	}

	return signature, nil
}

// EncodedSignature returns the Signature() after base64 encoding and url escaping
func (u *UrlData) EncodedSignature() (string, error) {
	signature, err := u.Signature()
	if err != nil {
		return "", err
	}

	// base64 encode signature
	encoded := base64.StdEncoding.EncodeToString(signature)
	// encoded signature may include /, = characters that need escaping
	encoded = url.QueryEscape(encoded)

	return encoded, nil
}

// SignedUrl constructs the final signed URL a client can use to retrieve storage object
func (u *UrlData) SignedUrl() (string, error) {

	encodedSig, err := u.EncodedSignature()
	if err != nil {
		return "", err
	}

	// build url
	// https://cloud.google.com/storage/docs/access-control/create-signed-urls-program
	var urlBuffer bytes.Buffer
	urlBuffer.WriteString(gcsBaseUrl)
	urlBuffer.WriteString(u.Path)
	urlBuffer.WriteString("?GoogleAccessId=")
	urlBuffer.WriteString(u.JwtConfig.Email)
	urlBuffer.WriteString("&Expires=")
	urlBuffer.WriteString(strconv.Itoa(u.Expires))
	urlBuffer.WriteString("&Signature=")
	urlBuffer.WriteString(encodedSig)

	return urlBuffer.String(), nil
}

// SignString calculates the SHA256 signature of the input string
func SignString(toSign []byte, cfg *jwt.Config) ([]byte, error) {
	// Parse private key
	pk, err := parsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, errwrap.Wrapf("failed to sign string, could not parse key: {{err}}", err)
	}

	// Hash string
	hasher := sha256.New()
	hasher.Write(toSign)

	// Sign string
	signed, err := rsa.SignPKCS1v15(rand.Reader, pk, crypto.SHA256, hasher.Sum(nil))
	if err != nil {
		return nil, errwrap.Wrapf("failed to sign string, an error occurred: {{err}}", err)
	}

	return signed, nil
}
