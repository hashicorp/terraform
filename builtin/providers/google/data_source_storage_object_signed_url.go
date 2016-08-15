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
	"fmt"
	"github.com/hashicorp/terraform/helper/pathorcontents"
	"github.com/hashicorp/terraform/helper/schema"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
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
			//TODO: implement support
			//"content_type": &schema.Schema{
			//	Type:     schema.TypeString,
			//	Optional: true,
			//	Default:  "",
			//},
			"credentials": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"duration": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "1h",
			},
			"http_method": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "GET",
			},
			//TODO: implement support
			//"http_headers": &schema.Schema{
			//	Type:     schema.TypeList,
			//	Optional: true,
			//},
			//TODO: implement support
			//"md5_digest": &schema.Schema{
			//	Type:     schema.TypeString,
			//	Optional: true,
			//	Default:  "",
			//},
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

func dataSourceGoogleSignedUrlRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Build UrlData object from data source attributes
	urlData := &UrlData{}

	// HTTP Method
	if method, ok := d.GetOk("http_method"); ok && len(method.(string)) >= 3 {
		urlData.HttpMethod = method.(string)
	} else {
		return fmt.Errorf("not a valid http method")
	}

	// convert duration to an expiration datetime (unix time in seconds)
	durationString := "1h"
	if v, ok := d.GetOk("duration"); ok {
		durationString = v.(string)
	}
	duration, err := time.ParseDuration(durationString)
	if err != nil {
		return fmt.Errorf("could not parse duration")
	}
	expires := time.Now().Unix() + int64(duration.Seconds())
	urlData.Expires = int(expires)

	// object path
	path := []string{
		"",
		d.Get("bucket").(string),
		d.Get("path").(string),
	}
	objectPath := strings.Join(path, "/")
	urlData.Path = objectPath

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Load JWT Config from Google Credentials
	jwtConfig, err := loadJwtConfig(d, config)
	if err != nil {
		return err
	}
	urlData.JwtConfig = jwtConfig

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Sign url object data
	signature, err := SignString(urlData.CreateSigningString(), jwtConfig)
	if err != nil {
		return fmt.Errorf("could not sign data: %v", err)
	}
	urlData.Signature = signature

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Construct URL
	finalUrl := urlData.BuildUrl()
	d.SetId(urlData.EncodedSignature())
	d.Set("signed_url", finalUrl)

	return nil
}

// This looks for credentials json in the following places,
// in order of preference:
//
//   1. Credentials provided in data source `credentials` attribute.
//   2. Credentials provided in the provider definition.
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
			return nil, fmt.Errorf("Error loading credentials: %s", err)
		}

		cfg, err := google.JWTConfigFromJSON([]byte(contents), "")
		if err != nil {
			return nil, fmt.Errorf("Error parsing credentials: \n %s \n Error: %s", contents, err)
		}
		return cfg, nil
	}

	return nil, fmt.Errorf("Credentials not found in datasource, provider configuration or GOOGLE_APPLICATION_CREDENTIALS environment variable.")
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
			return nil, fmt.Errorf("private key should be a PEM or plain PKSC1 or PKCS8; parse error: %v", err)
		}
	}
	parsed, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is invalid")
	}
	return parsed, nil
}

type UrlData struct {
	JwtConfig  *jwt.Config
	HttpMethod string
	Expires    int
	Path       string
	Signature  []byte
}

// Creates a string in the form ready for signing:
// https://cloud.google.com/storage/docs/access-control/create-signed-urls-program
// Example output:
// -------------------
// GET
//
//
// 1388534400
// bucket/objectname
// -------------------
func (u *UrlData) CreateSigningString() []byte {
	var buf bytes.Buffer

	// HTTP VERB
	buf.WriteString(u.HttpMethod)
	buf.WriteString("\n")

	// MD5 digest (optional)
	// TODO
	buf.WriteString("\n")

	// request content-type (optional)
	// TODO
	buf.WriteString("\n")

	// signed url expiration
	buf.WriteString(strconv.Itoa(u.Expires))
	buf.WriteString("\n")

	// additional request headers (optional)
	// TODO

	// object path
	buf.WriteString(u.Path)

	return buf.Bytes()
}

func (u *UrlData) EncodedSignature() string {
	// base64 encode signature
	encoded := base64.StdEncoding.EncodeToString(u.Signature)
	// encoded signature may include /, = characters that need escaping
	encoded = url.QueryEscape(encoded)

	return encoded
}

// Builds the final signed URL a client can use to retrieve storage object
func (u *UrlData) BuildUrl() string {

	// set url
	// https://cloud.google.com/storage/docs/access-control/create-signed-urls-program
	var urlBuffer bytes.Buffer
	urlBuffer.WriteString(gcsBaseUrl)
	urlBuffer.WriteString(u.Path)
	urlBuffer.WriteString("?GoogleAccessId=")
	urlBuffer.WriteString(u.JwtConfig.Email)
	urlBuffer.WriteString("&Expires=")
	urlBuffer.WriteString(strconv.Itoa(u.Expires))
	urlBuffer.WriteString("&Signature=")
	urlBuffer.WriteString(u.EncodedSignature())

	return urlBuffer.String()
}

func SignString(toSign []byte, cfg *jwt.Config) ([]byte, error) {
	// Parse private key
	pk, err := parsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("could not parse key: %v\nKey:%s", err, string(cfg.PrivateKey))
	}

	// Hash string
	hasher := sha256.New()
	hasher.Write(toSign)

	// Sign string
	signed, err := rsa.SignPKCS1v15(rand.Reader, pk, crypto.SHA256, hasher.Sum(nil))
	if err != nil {
		return nil, fmt.Errorf("error signing string: %s\n", err)
	}

	return signed, nil
}
