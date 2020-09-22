package cos

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
)

// ErrorResponse 包含 API 返回的错误信息
//
// https://www.qcloud.com/document/product/436/7730
type ErrorResponse struct {
	XMLName   xml.Name       `xml:"Error"`
	Response  *http.Response `xml:"-"`
	Code      string
	Message   string
	Resource  string
	RequestID string `header:"x-cos-request-id,omitempty" url:"-" xml:"-"`
	TraceID   string `xml:"TraceId,omitempty"`
}

// Error returns the error msg
func (r *ErrorResponse) Error() string {
	RequestID := r.RequestID
	if RequestID == "" {
		RequestID = r.Response.Header.Get("X-Cos-Request-Id")
	}
	TraceID := r.TraceID
	if TraceID == "" {
		TraceID = r.Response.Header.Get("X-Cos-Trace-Id")
	}
	return fmt.Sprintf("%v %v: %d %v(Message: %v, RequestId: %v, TraceId: %v)",
		r.Response.Request.Method, r.Response.Request.URL,
		r.Response.StatusCode, r.Code, r.Message, RequestID, TraceID)
}

// 检查 response 是否是出错时的返回的 response
func checkResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}
	errorResponse := &ErrorResponse{Response: r}
	data, err := ioutil.ReadAll(r.Body)
	if err == nil && data != nil {
		xml.Unmarshal(data, errorResponse)
	}
	return errorResponse
}
