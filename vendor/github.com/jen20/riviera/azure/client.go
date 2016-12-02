package azure

import (
	"io/ioutil"
	"log"

	"net/http"

	"github.com/hashicorp/go-retryablehttp"
)

type Client struct {
	logger *log.Logger

	subscriptionID string

	tokenRequester *tokenRequester
	httpClient     *retryablehttp.Client
}

func NewClient(creds *AzureResourceManagerCredentials) (*Client, error) {
	defaultLogger := log.New(ioutil.Discard, "", 0)

	httpClient := retryablehttp.NewClient()
	httpClient.Logger = defaultLogger

	tr := newTokenRequester(httpClient, creds.ClientID, creds.ClientSecret, creds.TenantID)

	return &Client{
		subscriptionID: creds.SubscriptionID,
		httpClient:     httpClient,
		tokenRequester: tr,
		logger:         defaultLogger,
	}, nil
}

func (c *Client) SetRequestLoggingHook(hook func(*log.Logger, *http.Request, int)) {
	c.httpClient.RequestLogHook = hook
}

func (c *Client) SetLogger(newLogger *log.Logger) {
	c.logger = newLogger
	c.httpClient.Logger = newLogger
}

func (c *Client) NewRequest() *Request {
	return &Request{
		client: c,
	}
}

func (c *Client) NewRequestForURI(resourceURI string) *Request {
	return &Request{
		URI:    &resourceURI,
		client: c,
	}
}
