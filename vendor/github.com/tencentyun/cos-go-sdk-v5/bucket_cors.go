package cos

import (
	"context"
	"encoding/xml"
	"net/http"
)

// BucketCORSRule is the rule of BucketCORS
type BucketCORSRule struct {
	ID             string   `xml:"ID,omitempty"`
	AllowedMethods []string `xml:"AllowedMethod"`
	AllowedOrigins []string `xml:"AllowedOrigin"`
	AllowedHeaders []string `xml:"AllowedHeader,omitempty"`
	MaxAgeSeconds  int      `xml:"MaxAgeSeconds,omitempty"`
	ExposeHeaders  []string `xml:"ExposeHeader,omitempty"`
}

// BucketGetCORSResult is the result of GetBucketCORS
type BucketGetCORSResult struct {
	XMLName xml.Name         `xml:"CORSConfiguration"`
	Rules   []BucketCORSRule `xml:"CORSRule,omitempty"`
}

// GetCORS 实现 Bucket 跨域访问配置读取。
//
// https://www.qcloud.com/document/product/436/8274
func (s *BucketService) GetCORS(ctx context.Context) (*BucketGetCORSResult, *Response, error) {
	var res BucketGetCORSResult
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?cors",
		method:  http.MethodGet,
		result:  &res,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return &res, resp, err
}

// BucketPutCORSOptions is the option of PutBucketCORS 
type BucketPutCORSOptions struct {
	XMLName xml.Name         `xml:"CORSConfiguration"`
	Rules   []BucketCORSRule `xml:"CORSRule,omitempty"`
}

// PutCORS 实现 Bucket 跨域访问设置，您可以通过传入XML格式的配置文件实现配置，文件大小限制为64 KB。
//
// https://www.qcloud.com/document/product/436/8279
func (s *BucketService) PutCORS(ctx context.Context, opt *BucketPutCORSOptions) (*Response, error) {
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?cors",
		method:  http.MethodPut,
		body:    opt,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return resp, err
}

// DeleteCORS 实现 Bucket 跨域访问配置删除。
//
// https://www.qcloud.com/document/product/436/8283
func (s *BucketService) DeleteCORS(ctx context.Context) (*Response, error) {
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?cors",
		method:  http.MethodDelete,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return resp, err
}
