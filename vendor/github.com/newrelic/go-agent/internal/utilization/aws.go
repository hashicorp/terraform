package utilization

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	maxResponseLengthBytes = 255

	// AWS data gathering requires making three web requests, therefore this
	// timeout is in keeping with the spec's total timeout of 1 second.
	individualConnectionTimeout = 300 * time.Millisecond
)

const (
	awsHost = "169.254.169.254"

	typeEndpointPath = "/2008-02-01/meta-data/instance-type"
	idEndpointPath   = "/2008-02-01/meta-data/instance-id"
	zoneEndpointPath = "/2008-02-01/meta-data/placement/availability-zone"

	typeEndpoint = "http://" + awsHost + typeEndpointPath
	idEndpoint   = "http://" + awsHost + idEndpointPath
	zoneEndpoint = "http://" + awsHost + zoneEndpointPath
)

// awsValidationError represents a response from an AWS endpoint that doesn't
// match the format expectations.
type awsValidationError struct {
	e error
}

func (a awsValidationError) Error() string {
	return a.e.Error()
}

func isAWSValidationError(e error) bool {
	_, is := e.(awsValidationError)
	return is
}

func getAWS() (*vendor, error) {
	return getEndpoints(&http.Client{
		Timeout: individualConnectionTimeout,
	})
}

func getEndpoints(client *http.Client) (*vendor, error) {
	v := &vendor{}
	var err error

	v.ID, err = getAndValidate(client, idEndpoint)
	if err != nil {
		return nil, err
	}
	v.Type, err = getAndValidate(client, typeEndpoint)
	if err != nil {
		return nil, err
	}
	v.Zone, err = getAndValidate(client, zoneEndpoint)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func getAndValidate(client *http.Client, endpoint string) (string, error) {
	response, err := client.Get(endpoint)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", fmt.Errorf("unexpected response code %d", response.StatusCode)
	}

	b := make([]byte, maxResponseLengthBytes+1)
	num, err := response.Body.Read(b)
	if err != nil && err != io.EOF {
		return "", err
	}

	if num > maxResponseLengthBytes {
		return "", awsValidationError{
			fmt.Errorf("maximum length %d exceeded", maxResponseLengthBytes),
		}
	}

	responseText := string(b[:num])

	for _, r := range responseText {
		if !isAcceptableRune(r) {
			return "", awsValidationError{
				fmt.Errorf("invalid character %x", r),
			}
		}
	}

	return responseText, nil
}

// See:
// https://source.datanerd.us/agents/agent-specs/blob/master/Utilization.md#normalizing-aws-data
func isAcceptableRune(r rune) bool {
	switch r {
	case 0xFFFD:
		return false
	case '_', ' ', '/', '.', '-':
		return true
	default:
		return r > 0x7f ||
			('0' <= r && r <= '9') ||
			('a' <= r && r <= 'z') ||
			('A' <= r && r <= 'Z')
	}
}
