package cos

import (
	"context"
	"encoding/xml"
	"net/http"
)

// BucketLifecycleFilter is the param of BucketLifecycleRule
type BucketLifecycleFilter struct {
	Prefix string `xml:"Prefix,omitempty"`
}

// BucketLifecycleExpiration is the param of BucketLifecycleRule
type BucketLifecycleExpiration struct {
	Date string `xml:"Date,omitempty"`
	Days int    `xml:"Days,omitempty"`
}

// BucketLifecycleTransition is the param of BucketLifecycleRule
type BucketLifecycleTransition struct {
	Date         string `xml:"Date,omitempty"`
	Days         int    `xml:"Days,omitempty"`
	StorageClass string
}

// BucketLifecycleAbortIncompleteMultipartUpload is the param of BucketLifecycleRule
type BucketLifecycleAbortIncompleteMultipartUpload struct {
	DaysAfterInitiation string `xml:"DaysAfterInititation,omitempty"`
}

// BucketLifecycleRule is the rule of BucketLifecycle
type BucketLifecycleRule struct {
	ID                             string `xml:"ID,omitempty"`
	Status                         string
	Filter                         *BucketLifecycleFilter                         `xml:"Filter,omitempty"`
	Transition                     *BucketLifecycleTransition                     `xml:"Transition,omitempty"`
	Expiration                     *BucketLifecycleExpiration                     `xml:"Expiration,omitempty"`
	AbortIncompleteMultipartUpload *BucketLifecycleAbortIncompleteMultipartUpload `xml:"AbortIncompleteMultipartUpload,omitempty"`
}

// BucketGetLifecycleResult is the result of BucketGetLifecycle
type BucketGetLifecycleResult struct {
	XMLName xml.Name              `xml:"LifecycleConfiguration"`
	Rules   []BucketLifecycleRule `xml:"Rule,omitempty"`
}

// GetLifecycle 请求实现读取生命周期管理的配置。当配置不存在时，返回404 Not Found。
// https://www.qcloud.com/document/product/436/8278
func (s *BucketService) GetLifecycle(ctx context.Context) (*BucketGetLifecycleResult, *Response, error) {
	var res BucketGetLifecycleResult
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?lifecycle",
		method:  http.MethodGet,
		result:  &res,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return &res, resp, err
}

// BucketPutLifecycleOptions is the option of PutBucketLifecycle
type BucketPutLifecycleOptions struct {
	XMLName xml.Name              `xml:"LifecycleConfiguration"`
	Rules   []BucketLifecycleRule `xml:"Rule,omitempty"`
}

// PutLifecycle 请求实现设置生命周期管理的功能。您可以通过该请求实现数据的生命周期管理配置和定期删除。
// 此请求为覆盖操作，上传新的配置文件将覆盖之前的配置文件。生命周期管理对文件和文件夹同时生效。
// https://www.qcloud.com/document/product/436/8280
func (s *BucketService) PutLifecycle(ctx context.Context, opt *BucketPutLifecycleOptions) (*Response, error) {
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?lifecycle",
		method:  http.MethodPut,
		body:    opt,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return resp, err
}

// DeleteLifecycle 请求实现删除生命周期管理。
// https://www.qcloud.com/document/product/436/8284
func (s *BucketService) DeleteLifecycle(ctx context.Context) (*Response, error) {
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?lifecycle",
		method:  http.MethodDelete,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return resp, err
}
