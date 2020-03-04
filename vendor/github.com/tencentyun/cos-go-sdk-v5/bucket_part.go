package cos

import (
	"context"
	"encoding/xml"
	"net/http"
)

// ListMultipartUploadsResult is the result of ListMultipartUploads
type ListMultipartUploadsResult struct {
	XMLName            xml.Name `xml:"ListMultipartUploadsResult"`
	Bucket             string   `xml:"Bucket"`
	EncodingType       string   `xml:"Encoding-Type"`
	KeyMarker          string
	UploadIDMarker     string `xml:"UploadIdMarker"`
	NextKeyMarker      string
	NextUploadIDMarker string `xml:"NextUploadIdMarker"`
	MaxUploads         int
	IsTruncated        bool
	Uploads            []struct {
		Key          string
		UploadID     string `xml:"UploadId"`
		StorageClass string
		Initiator    *Initiator
		Owner        *Owner
		Initiated    string
	} `xml:"Upload,omitempty"`
	Prefix         string
	Delimiter      string   `xml:"delimiter,omitempty"`
	CommonPrefixes []string `xml:"CommonPrefixs>Prefix,omitempty"`
}

// ListMultipartUploadsOptions is the option of ListMultipartUploads
type ListMultipartUploadsOptions struct {
	Delimiter      string `url:"delimiter,omitempty"`
	EncodingType   string `url:"encoding-type,omitempty"`
	Prefix         string `url:"prefix,omitempty"`
	MaxUploads     int    `url:"max-uploads,omitempty"`
	KeyMarker      string `url:"key-marker,omitempty"`
	UploadIDMarker string `url:"upload-id-marker,omitempty"`
}

// ListMultipartUploads 用来查询正在进行中的分块上传。单次最多列出1000个正在进行中的分块上传。
//
// https://www.qcloud.com/document/product/436/7736
func (s *BucketService) ListMultipartUploads(ctx context.Context, opt *ListMultipartUploadsOptions) (*ListMultipartUploadsResult, *Response, error) {
	var res ListMultipartUploadsResult
	sendOpt := sendOptions{
		baseURL:  s.client.BaseURL.BucketURL,
		uri:      "/?uploads",
		method:   http.MethodGet,
		result:   &res,
		optQuery: opt,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return &res, resp, err
}
