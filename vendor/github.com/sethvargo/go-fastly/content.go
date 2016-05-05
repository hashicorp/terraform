package fastly

import "net/http"

// EdgeCheck represents an edge check response from the Fastly API.
type EdgeCheck struct {
	Hash         string             `mapstructure:"hash"`
	Server       string             `mapstructure:"server"`
	ResponseTime float64            `mapstructure:"response_time"`
	Request      *EdgeCheckRequest  `mapstructure:"request"`
	Response     *EdgeCheckResponse `mapstructure:"response"`
}

// EdgeCheckRequest is the request part of an EdgeCheck response.
type EdgeCheckRequest struct {
	URL     string       `mapstructure:"url"`
	Method  string       `mapstructure:"method"`
	Headers *http.Header `mapstructure:"headers"`
}

// EdgeCheckResponse is the response part of an EdgeCheck response.
type EdgeCheckResponse struct {
	Status  uint         `mapstructure:"status"`
	Headers *http.Header `mapstructure:"headers"`
}

// EdgeCheckInput is used as input to the EdgeCheck function.
type EdgeCheckInput struct {
	URL string `form:"url,omitempty"`
}

// EdgeCheck queries the edge cache for all of Fastly's servers for the given
// URL.
func (c *Client) EdgeCheck(i *EdgeCheckInput) ([]*EdgeCheck, error) {
	resp, err := c.Get("/content/edge_check", &RequestOptions{
		Params: map[string]string{
			"url": i.URL,
		},
	})
	if err != nil {
		return nil, err
	}

	var e []*EdgeCheck
	if err := decodeJSON(&e, resp.Body); err != nil {
		return nil, err
	}
	return e, nil
}
