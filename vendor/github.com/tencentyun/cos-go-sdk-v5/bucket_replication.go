package cos

import (
	"context"
	"encoding/xml"
	"net/http"
)

// ReplicationDestination is the sub struct of BucketReplicationRule
type ReplicationDestination struct {
	Bucket       string `xml:"Bucket"`
	StorageClass string `xml:"StorageClass,omitempty"`
}

// BucketReplicationRule is the main param of replication
type BucketReplicationRule struct {
	ID          string                  `xml:"ID,omitempty"`
	Status      string                  `xml:"Status"`
	Prefix      string                  `xml:"Prefix"`
	Destination *ReplicationDestination `xml:"Destination"`
}

// PutBucketReplicationOptions is the options of PutBucketReplication
type PutBucketReplicationOptions struct {
	XMLName xml.Name                `xml:"ReplicationConfiguration"`
	Role    string                  `xml:"Role"`
	Rule    []BucketReplicationRule `xml:"Rule"`
}

//  GetBucketReplicationResult is the result of GetBucketReplication
type GetBucketReplicationResult struct {
	XMLName xml.Name                `xml:"ReplicationConfiguration"`
	Role    string                  `xml:"Role"`
	Rule    []BucketReplicationRule `xml:"Rule"`
}

// PutBucketReplication https://cloud.tencent.com/document/product/436/19223
func (s *BucketService) PutBucketReplication(ctx context.Context, opt *PutBucketReplicationOptions) (*Response, error) {
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?replication",
		method:  http.MethodPut,
		body:    opt,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return resp, err

}

// GetBucketReplication https://cloud.tencent.com/document/product/436/19222
func (s *BucketService) GetBucketReplication(ctx context.Context) (*GetBucketReplicationResult, *Response, error) {
	var res GetBucketReplicationResult
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?replication",
		method:  http.MethodGet,
		result:  &res,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return &res, resp, err

}

// DeleteBucketReplication https://cloud.tencent.com/document/product/436/19221
func (s *BucketService) DeleteBucketReplication(ctx context.Context) (*Response, error) {
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?replication",
		method:  http.MethodDelete,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return resp, err
}
