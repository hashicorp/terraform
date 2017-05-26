package alicloud

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func (client *AliyunClient) QueryOssBucketById(id string) (info *oss.BucketInfo, err error) {

	bucket, err := client.ossconn.GetBucketInfo(id)
	if err != nil {
		return nil, err
	}

	return &bucket.BucketInfo, nil
}
