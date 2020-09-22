package cos

import (
	"context"
	"encoding/xml"
	"net/http"
)

// BucketPutVersionOptions is the options of PutBucketVersioning
type BucketPutVersionOptions struct {
	XMLName xml.Name `xml:"VersioningConfiguration"`
	Status  string   `xml:"Status"`
}

// BucketGetVersionResult is the result of GetBucketVersioning
type BucketGetVersionResult struct {
	XMLName xml.Name `xml:"VersioningConfiguration"`
	Status  string   `xml:"Status"`
}

// PutVersion https://cloud.tencent.com/document/product/436/19889
// Status has Suspended\Enabled
func (s *BucketService) PutVersioning(ctx context.Context, opt *BucketPutVersionOptions) (*Response, error) {
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?versioning",
		method:  http.MethodPut,
		body:    opt,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return resp, err
}

// GetVersion https://cloud.tencent.com/document/product/436/19888
func (s *BucketService) GetVersioning(ctx context.Context) (*BucketGetVersionResult, *Response, error) {
	var res BucketGetVersionResult
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?versioning",
		method:  http.MethodGet,
		result:  &res,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return &res, resp, err
}
