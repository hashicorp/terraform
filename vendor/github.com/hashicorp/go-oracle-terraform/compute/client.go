package compute

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-oracle-terraform/opc"
)

const CMP_USERNAME = "/Compute-%s/%s"
const CMP_QUALIFIED_NAME = "%s/%s"
const DEFAULT_MAX_RETRIES = 1

// Client represents an authenticated compute client, with compute credentials and an api client.
type Client struct {
	identityDomain *string
	userName       *string
	password       *string
	apiEndpoint    *url.URL
	httpClient     *http.Client
	authCookie     *http.Cookie
	cookieIssued   time.Time
	maxRetries     *int
	logger         opc.Logger
	loglevel       opc.LogLevelType
}

func NewComputeClient(c *opc.Config) (*Client, error) {
	// First create a client
	client := &Client{
		identityDomain: c.IdentityDomain,
		userName:       c.Username,
		password:       c.Password,
		apiEndpoint:    c.APIEndpoint,
		httpClient:     c.HTTPClient,
		maxRetries:     c.MaxRetries,
		loglevel:       c.LogLevel,
	}

	// Setup logger; defaults to stdout
	if c.Logger == nil {
		client.logger = opc.NewDefaultLogger()
	} else {
		client.logger = c.Logger
	}

	// If LogLevel was not set to something different,
	// double check for env var
	if c.LogLevel == 0 {
		client.loglevel = opc.LogLevel()
	}

	if err := client.getAuthenticationCookie(); err != nil {
		return nil, err
	}

	// Default max retries if unset
	if c.MaxRetries == nil {
		client.maxRetries = opc.Int(DEFAULT_MAX_RETRIES)
	}

	// Protect against any nil http client
	if c.HTTPClient == nil {
		return nil, fmt.Errorf("No HTTP client specified in config")
	}

	return client, nil
}

func (c *Client) executeRequest(method, path string, body interface{}) (*http.Response, error) {
	// Parse URL Path
	urlPath, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	// Marshall request body
	var requestBody io.ReadSeeker
	var marshaled []byte
	if body != nil {
		marshaled, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewReader(marshaled)
	}

	// Create request
	req, err := http.NewRequest(method, c.formatURL(urlPath), requestBody)
	if err != nil {
		return nil, err
	}

	debugReqString := fmt.Sprintf("HTTP %s Req (%s)", method, path)
	if body != nil {
		req.Header.Set("Content-Type", "application/oracle-compute-v3+json")
		// Don't leak creds in STDERR
		if path != "/authenticate/" {
			debugReqString = fmt.Sprintf("%s:\n %s", debugReqString, string(marshaled))
		}
	}

	// Log the request before the authentication cookie, so as not to leak credentials
	c.debugLogString(debugReqString)

	// If we have an authentication cookie, let's authenticate, refreshing cookie if need be
	if c.authCookie != nil {
		if time.Since(c.cookieIssued).Minutes() > 25 {
			if err := c.getAuthenticationCookie(); err != nil {
				return nil, err
			}
		}
		req.AddCookie(c.authCookie)
	}

	// Execute request with supplied client
	resp, err := c.retryRequest(req)
	//resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return resp, nil
	}

	oracleErr := &opc.OracleError{
		StatusCode: resp.StatusCode,
	}

	// Even though the returned body will be in json form, it's undocumented what
	// fields are actually returned. Once we get documentation of the actual
	// error fields that are possible to be returned we can have stricter error types.
	if resp.Body != nil {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		oracleErr.Message = buf.String()
	}

	return nil, oracleErr
}

// Allow retrying the request until it either returns no error,
// or we exceed the number of max retries
func (c *Client) retryRequest(req *http.Request) (*http.Response, error) {
	// Double check maxRetries is not nil
	var retries int
	if c.maxRetries == nil {
		retries = DEFAULT_MAX_RETRIES
	} else {
		retries = *c.maxRetries
	}

	var statusCode int
	var errMessage string

	for i := 0; i < retries; i++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			return resp, nil
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		errMessage = buf.String()
		statusCode = resp.StatusCode
		c.debugLogString(fmt.Sprintf("Encountered HTTP (%d) Error: %s", statusCode, errMessage))
		c.debugLogString(fmt.Sprintf("%d/%d retries left", i+1, retries))
	}

	oracleErr := &opc.OracleError{
		StatusCode: statusCode,
		Message:    errMessage,
	}

	// We ran out of retries to make, return the error and response
	return nil, oracleErr
}

func (c *Client) formatURL(path *url.URL) string {
	return c.apiEndpoint.ResolveReference(path).String()
}

func (c *Client) getUserName() string {
	return fmt.Sprintf(CMP_USERNAME, *c.identityDomain, *c.userName)
}

// From compute_client
// GetObjectName returns the fully-qualified name of an OPC object, e.g. /identity-domain/user@email/{name}
func (c *Client) getQualifiedName(name string) string {
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, "/oracle") || strings.HasPrefix(name, "/Compute-") {
		return name
	}
	return fmt.Sprintf(CMP_QUALIFIED_NAME, c.getUserName(), name)
}

func (c *Client) getObjectPath(root, name string) string {
	return fmt.Sprintf("%s%s", root, c.getQualifiedName(name))
}

// GetUnqualifiedName returns the unqualified name of an OPC object, e.g. the {name} part of /identity-domain/user@email/{name}
func (c *Client) getUnqualifiedName(name string) string {
	if name == "" {
		return name
	}
	if strings.HasPrefix(name, "/oracle") {
		return name
	}
	if !strings.Contains(name, "/") {
		return name
	}

	nameParts := strings.Split(name, "/")
	return strings.Join(nameParts[3:], "/")
}

func (c *Client) unqualify(names ...*string) {
	for _, name := range names {
		*name = c.getUnqualifiedName(*name)
	}
}

func (c *Client) unqualifyUrl(url *string) {
	var validID = regexp.MustCompile(`(\/(Compute[^\/\s]+))(\/[^\/\s]+)(\/[^\/\s]+)`)
	name := validID.FindString(*url)
	*url = c.getUnqualifiedName(name)
}

func (c *Client) getQualifiedList(list []string) []string {
	for i, name := range list {
		list[i] = c.getQualifiedName(name)
	}
	return list
}

func (c *Client) getUnqualifiedList(list []string) []string {
	for i, name := range list {
		list[i] = c.getUnqualifiedName(name)
	}
	return list
}

func (c *Client) getQualifiedListName(name string) string {
	nameParts := strings.Split(name, ":")
	listType := nameParts[0]
	listName := nameParts[1]
	return fmt.Sprintf("%s:%s", listType, c.getQualifiedName(listName))
}

func (c *Client) unqualifyListName(qualifiedName string) string {
	nameParts := strings.Split(qualifiedName, ":")
	listType := nameParts[0]
	listName := nameParts[1]
	return fmt.Sprintf("%s:%s", listType, c.getUnqualifiedName(listName))
}

// Retry function
func (c *Client) waitFor(description string, timeoutSeconds int, test func() (bool, error)) error {
	tick := time.Tick(1 * time.Second)

	for i := 0; i < timeoutSeconds; i++ {
		select {
		case <-tick:
			completed, err := test()
			c.debugLogString(fmt.Sprintf("Waiting for %s (%d/%ds)", description, i, timeoutSeconds))
			if err != nil || completed {
				return err
			}
		}
	}
	return fmt.Errorf("Timeout waiting for %s", description)
}

// Used to determine if the checked resource was found or not.
func WasNotFoundError(e error) bool {
	err, ok := e.(*opc.OracleError)
	if ok {
		return err.StatusCode == 404
	}
	return false
}
