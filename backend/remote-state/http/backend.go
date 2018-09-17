package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
)

// New Backend
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"address": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "(Required) The address of the REST endpoint.",
			},

			"update_method": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "POST",
				Description: "(Optional) HTTP method to use when updating state. Defaults to POST.",
			},

			"lock_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "(Optional) The address of the lock REST endpoint. Defaults to address.",
			},

			"lock_method": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "LOCK",
				Description: "(Optional) The HTTP method to use when locking. Defaults to LOCK.",
			},

			"unlock_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "(Optional) The address of the unlock REST endpoint. Defaults to address.",
			},

			"unlock_method": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "UNLOCK",
				Description: "(Optional) The HTTP method to use when unlocking. Defaults to UNLOCK.",
			},

			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "(Optional) The username for HTTP basic authentication.",
			},

			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "(Optional) The password for HTTP basic authentication.",
			},

			"skip_cert_verification": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "(Optional) Whether to skip TLS verification. Defaults to false.",
			},

			"local_ca_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "CA to be used for TLS",
			},

			"mutual_tls_authentication": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Use mutual tls authentication. local_ca_file, local_cert_file, local_key_file needs to be set. Defaults to false.",
			},

			"local_cert_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Cert file to be used for mutual tls authentication.",
			},

			"local_key_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Key file to be used for mutual tls authentication.",
			},
		},
	}

	result := &Backend{Backend: s}
	result.Backend.ConfigureFunc = result.configure
	return result
}

// Backend Structure
type Backend struct {
	*schema.Backend
	data   *schema.ResourceData
	client *http.Client

	// The fields below are set from configure
	address         string
	updateMethod    string
	lockAddress     string
	lockMethod      string
	unlockMethod    string
	unlockAddress   string
	skipTLS         bool
	localCertCAFile string
	localCertFile   string
	localKeyFile    string
	localCAFile     string
	mutualTLS       bool
	username        string
	password        string
}

func (b *Backend) configure(ctx context.Context) error {
	if b.client != nil {
		return nil
	}
	// Grab the resource data
	data := schema.FromContextBackendConfig(ctx)

	// Store the URL address
	addressURL := data.Get("address").(string)
	// Validate URL address
	validationErr := assertValidURL(addressURL)
	if validationErr != nil {
		return validationErr
	}
	b.address = addressURL

	b.updateMethod = data.Get("update_method").(string)

	if v, ok := data.GetOk("lock_address"); ok {
		lockAddress := v.(string)
		// Validate lockAddress
		validationErr := assertValidURL(lockAddress)
		if validationErr != nil {
			return validationErr
		}
		b.lockAddress = lockAddress
	} else {
		// If lockAddress is null, use the http rest api address
		b.lockAddress = b.address
	}

	b.lockMethod = data.Get("lock_method").(string)

	if v, ok := data.GetOk("unlock_address"); ok {
		unlockAddress := v.(string)
		// Validate unlockAddress
		validationErr := assertValidURL(unlockAddress)
		if validationErr != nil {
			return validationErr
		}
		b.unlockAddress = unlockAddress
	} else {
		// If unlockAddress is null, use the http rest api address
		b.unlockAddress = b.address
	}

	b.unlockMethod = data.Get("unlock_method").(string)

	if v, ok := data.GetOk("username"); ok {
		b.username = v.(string)
	}

	if v, ok := data.GetOk("password"); ok {
		b.password = v.(string)
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	// skip_cert_verification client
	if b.skipTLS = data.Get("skip_cert_verification").(bool); b.skipTLS {
		// If skip_cert_verification = true, the address must be of type HTTPS
		if !isHTTPS(addressURL) {
			return fmt.Errorf("Address must be of type HTTPS if skip_cert_verification = true")
		}
		log.Printf("[DEBUG] Using https client with skipping cert verification")
		// add the option to ignores TLS verification to our client
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	// CA provided client and mutual tls authentication client
	if v, ok := data.GetOk("local_ca_file"); ok {
		// If local_cert_ca_file exists, the address must be of type HTTPS
		if !isHTTPS(addressURL) {
			return fmt.Errorf("Address must be of type HTTPS if local_ca_file is set")
		}
		b.localCertCAFile = v.(string)

		// Get the SystemCertPool, continue with an empty pool on error
		rootCAs, _ := x509.SystemCertPool()
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}
		// Read in the ca cert file
		cert, err := ioutil.ReadFile(b.localCertCAFile)
		if err != nil {
			return fmt.Errorf("Failed to read %s into memory: %s", b.localCertCAFile, err)
		}
		// Append our cert to the system pool
		if ok := rootCAs.AppendCertsFromPEM(cert); !ok {
			return fmt.Errorf("No certs could be appended: %s", cert)
		}
		// add RootCas to our client
		log.Printf("[DEBUG] Using https client with provided CA")
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: rootCAs,
			},
		}
		if b.mutualTLS = data.Get("mutual_tls_authentication").(bool); b.mutualTLS {
			// If mutual_tls_authentication = true, the local_key_file needs to be set.
			if v, ok := data.GetOk("local_key_file"); ok {
				b.localKeyFile = v.(string)
			} else {
				return fmt.Errorf("mutual_tls_authentication is true and local_key_file is not set %s", b.localKeyFile)
			}
			// If mutual_tls_authentication = true, the local_cert_file needs to be set.
			if v, ok := data.GetOk("local_cert_file"); ok {
				b.localCertFile = v.(string)
			} else {
				return fmt.Errorf("mutual_tls_authentication is true and local_cert_file is not set %s", b.localCertFile)
			}
			// load client cert
			certs, err := tls.LoadX509KeyPair(b.localCertFile, b.localKeyFile)
			if err != nil {
				return fmt.Errorf("Can not load pem files: %s and : %s. Error: %s", b.localCertFile, b.localKeyFile, err)
			}
			log.Printf("[DEBUG] Using https client with provided CA and mutual tls authentication")
			// add the certificate for our mutual tls authentication
			client.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates: []tls.Certificate{certs},
					RootCAs:      rootCAs,
				},
			}
		}
	}
	b.client = client
	return nil
}

//assertValidURL Parse URI and validate
func assertValidURL(addr string) error {
	addre, err := url.ParseRequestURI(addr)
	if err != nil {
		return fmt.Errorf("failed to parse address URL: %s", err)
	}
	if addre.Scheme != "http" && addre.Scheme != "https" {
		return fmt.Errorf("address must be of type HTTP or HTTPS")
	}
	return nil
}

//isHTTPS checks if address is of type https
func isHTTPS(addr string) bool {
	addre, _ := url.ParseRequestURI(addr)
	if addre.Scheme == "https" {
		return true
	}
	return false
}
