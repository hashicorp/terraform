package newrelic

import (
	"crypto/tls"
	"crypto/x509"
	"log"

	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/helper/pathorcontents"
	newrelic "github.com/paultyng/go-newrelic/api"
)

// Config contains New Relic provider settings
type Config struct {
	APIKey             string
	APIURL             string
	CACertFile         string
	InsecureSkipVerify bool
}

// Client returns a new client for accessing New Relic
func (c *Config) Client() (*newrelic.Client, error) {
	tlsCfg := &tls.Config{}
	if c.CACertFile != "" {
		caCert, _, err := pathorcontents.Read(c.CACertFile)
		if err != nil {
			log.Printf("Error reading CA Cert: %s", err)
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(caCert))
		tlsCfg.RootCAs = caCertPool
	}

	if c.InsecureSkipVerify {
		tlsCfg.InsecureSkipVerify = true
	}

	nrConfig := newrelic.Config{
		APIKey:    c.APIKey,
		BaseURL:   c.APIURL,
		Debug:     logging.IsDebugOrHigher(),
		TLSConfig: tlsCfg,
	}

	client := newrelic.New(nrConfig)

	log.Printf("[INFO] New Relic client configured")

	return &client, nil
}
