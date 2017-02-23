package xmlrpc

import (
	"regexp"
)

var (
	faultRx = regexp.MustCompile(`<fault>(\s|\S)+</fault>`)
)

type failedResponse struct {
	Code  string `xmlrpc:"faultCode"`
	Error string `xmlrpc:"faultString"`
	HttpStatusCode int
}

func (r *failedResponse) err() error {
	return &XmlRpcError{
		Code: r.Code,
		Err:  r.Error,
		HttpStatusCode: r.HttpStatusCode,
	}
}

type Response struct {
	data []byte
	httpStatusCode int
}

func NewResponse(data []byte, httpStatusCode int) *Response {
	return &Response{
		data: data,
		httpStatusCode: httpStatusCode,
	}
}

func (r *Response) Failed() bool {
	return faultRx.Match(r.data)
}

func (r *Response) Err() error {
	failedResp := new(failedResponse)
	if err := unmarshal(r.data, failedResp); err != nil {
		return err
	}
	failedResp.HttpStatusCode = r.httpStatusCode

	return failedResp.err()
}

func (r *Response) Unmarshal(v interface{}) error {
	if err := unmarshal(r.data, v); err != nil {
		return err
	}

	return nil
}
