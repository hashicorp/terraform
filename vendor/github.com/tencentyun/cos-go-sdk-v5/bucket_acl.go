package cos

import (
	"context"
	"net/http"
)

// BucketGetACLResult is same to the ACLXml
type BucketGetACLResult ACLXml

// GetACL 使用API读取Bucket的ACL表，只有所有者有权操作。
//
// https://www.qcloud.com/document/product/436/7733
func (s *BucketService) GetACL(ctx context.Context) (*BucketGetACLResult, *Response, error) {
	var res BucketGetACLResult
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?acl",
		method:  http.MethodGet,
		result:  &res,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return &res, resp, err
}

// BucketPutACLOptions is the option of PutBucketACL
type BucketPutACLOptions struct {
	Header *ACLHeaderOptions `url:"-" xml:"-"`
	Body   *ACLXml           `url:"-" header:"-"`
}

// PutACL 使用API写入Bucket的ACL表，您可以通过Header："x-cos-acl","x-cos-grant-read",
// "x-cos-grant-write","x-cos-grant-full-control"传入ACL信息，也可以通过body以XML格式传入ACL信息，
//
// 但是只能选择Header和Body其中一种，否则返回冲突。
//
// Put Bucket ACL是一个覆盖操作，传入新的ACL将覆盖原有ACL。只有所有者有权操作。
//
//   "x-cos-acl"：枚举值为public-read，private；public-read意味这个Bucket有公有读私有写的权限，
//   private意味这个Bucket有私有读写的权限。
//
//   "x-cos-grant-read"：意味被赋予权限的用户拥有该Bucket的读权限
//   "x-cos-grant-write"：意味被赋予权限的用户拥有该Bucket的写权限
//   "x-cos-grant-full-control"：意味被赋予权限的用户拥有该Bucket的读写权限
//
// https://www.qcloud.com/document/product/436/7737
func (s *BucketService) PutACL(ctx context.Context, opt *BucketPutACLOptions) (*Response, error) {
	header := opt.Header
	body := opt.Body
	if body != nil {
		header = nil
	}
	sendOpt := sendOptions{
		baseURL:   s.client.BaseURL.BucketURL,
		uri:       "/?acl",
		method:    http.MethodPut,
		body:      body,
		optHeader: header,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return resp, err
}
