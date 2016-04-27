package cloudflare

import "encoding/json"

// Response - Cloudflare API Response.
type Response struct {
	Result     json.RawMessage `json:"result"`
	ResultInfo *ResultInfo     `json:"result_info"`

	Errors  []*ResponseError `json:"errors"`
	Success bool             `json:"success"`
}

// ResultInfo - Cloudflare API Response Result Info.
type ResultInfo struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
	Count      int `json:"count,omitempty"`
	TotalCount int `json:"total_count,omitempty"`
}

// ResponseError - Cloudflare API Response error.
type ResponseError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// Err - Gets response error if any.
func (response *Response) Err() error {
	if len(response.Errors) > 0 {
		return response.Errors[0]
	}
	return nil
}

// Error - Returns response error message.
func (err *ResponseError) Error() string {
	return err.Message
}
