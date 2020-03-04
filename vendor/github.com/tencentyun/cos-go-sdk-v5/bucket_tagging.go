package cos

import (
	"context"
	"encoding/xml"
	"net/http"
)

// BucketTaggingTag is the tag of BucketTagging
type BucketTaggingTag struct {
	Key   string
	Value string
}

// BucketGetTaggingResult is the result of BucketGetTagging
type BucketGetTaggingResult struct {
	XMLName xml.Name           `xml:"Tagging"`
	TagSet  []BucketTaggingTag `xml:"TagSet>Tag,omitempty"`
}

// GetTagging 接口实现获取指定Bucket的标签。
//
// https://www.qcloud.com/document/product/436/8277
func (s *BucketService) GetTagging(ctx context.Context) (*BucketGetTaggingResult, *Response, error) {
	var res BucketGetTaggingResult
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?tagging",
		method:  http.MethodGet,
		result:  &res,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return &res, resp, err
}

// BucketPutTaggingOptions is the option of BucketPutTagging
type BucketPutTaggingOptions struct {
	XMLName xml.Name           `xml:"Tagging"`
	TagSet  []BucketTaggingTag `xml:"TagSet>Tag,omitempty"`
}

// PutTagging 接口实现给用指定Bucket打标签。用来组织和管理相关Bucket。
//
// 当该请求设置相同Key名称，不同Value时，会返回400。请求成功，则返回204。
//
// https://www.qcloud.com/document/product/436/8281
func (s *BucketService) PutTagging(ctx context.Context, opt *BucketPutTaggingOptions) (*Response, error) {
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?tagging",
		method:  http.MethodPut,
		body:    opt,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return resp, err
}

// DeleteTagging 接口实现删除指定Bucket的标签。
//
// https://www.qcloud.com/document/product/436/8286
func (s *BucketService) DeleteTagging(ctx context.Context) (*Response, error) {
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?tagging",
		method:  http.MethodDelete,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return resp, err
}
