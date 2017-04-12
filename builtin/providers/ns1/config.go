package ns1

import (
	"crypto/tls"
	"errors"
	"log"
	"net/http"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
)

type Config struct {
	Key       string
	Endpoint  string
	IgnoreSSL bool
}

// Client() returns a new NS1 client.
func (c *Config) Client() (*ns1.Client, error) {
	httpClient := &http.Client{}
	decos := []func(*ns1.Client){}

	if c.Key == "" {
		return nil, errors.New(`No valid credential sources found for NS1 Provider.
  Please see https://terraform.io/docs/providers/ns1/index.html for more information on
  providing credentials for the NS1 Provider`)
	}

	decos = append(decos, ns1.SetAPIKey(c.Key))
	if c.Endpoint != "" {
		decos = append(decos, ns1.SetEndpoint(c.Endpoint))
	}
	if c.IgnoreSSL {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient.Transport = tr
	}

	client := ns1.NewClient(httpClient, decos...)
	client.RateLimitStrategySleep()

	log.Printf("[INFO] NS1 Client configured for Endpoint: %s", client.Endpoint.String())

	return client, nil
}
