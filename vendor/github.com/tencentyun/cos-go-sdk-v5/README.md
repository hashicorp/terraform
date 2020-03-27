# cos-go-sdk-v5

腾讯云对象存储服务 COS(Cloud Object Storage) Go SDK（API 版本：V5 版本的 XML API）。

## Install

`go get -u github.com/tencentyun/cos-go-sdk-v5`


## Usage

```go
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
	
	"github.com/tencentyun/cos-go-sdk-v5"
)

func main() {
	//将<bucket>和<region>修改为真实的信息
	//bucket的命名规则为{name}-{appid} ，此处填写的存储桶名称必须为此格式
	u, _ := url.Parse("https://<bucket>.cos.<region>.myqcloud.com")
	b := &cos.BaseURL{BucketURL: u}
	c := cos.NewClient(b, &http.Client{
		//设置超时时间
		Timeout: 100 * time.Second,
		Transport: &cos.AuthorizationTransport{
			//如实填写账号和密钥，也可以设置为环境变量
			SecretID:  os.Getenv("COS_SECRETID"),
			SecretKey: os.Getenv("COS_SECRETKEY"),
		},
	})

	name := "test/hello.txt"
	resp, err := c.Object.Get(context.Background(), name, nil)
	if err != nil {
		panic(err)
	}
	bs, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("%s\n", string(bs))
}
```

所有的 API 在 [example](./example/) 目录下都有对应的使用示例。

Service API:

* [x] Get Service（使用示例：[service/get.go](./example/service/get.go)）

Bucket API:

* [x] Get Bucket（使用示例：[bucket/get.go](./example/bucket/get.go)）
* [x] Get Bucket ACL（使用示例：[bucket/getACL.go](./example/bucket/getACL.go)）
* [x] Get Bucket CORS（使用示例：[bucket/getCORS.go](./example/bucket/getCORS.go)）
* [x] Get Bucket Location（使用示例：[bucket/getLocation.go](./example/bucket/getLocation.go)）
* [x] Get Buket Lifecycle（使用示例：[bucket/getLifecycle.go](./example/bucket/getLifecycle.go)）
* [x] Get Bucket Tagging（使用示例：[bucket/getTagging.go](./example/bucket/getTagging.go)）
* [x] Put Bucket（使用示例：[bucket/put.go](./example/bucket/put.go)）
* [x] Put Bucket ACL（使用示例：[bucket/putACL.go](./example/bucket/putACL.go)）
* [x] Put Bucket CORS（使用示例：[bucket/putCORS.go](./example/bucket/putCORS.go)）
* [x] Put Bucket Lifecycle（使用示例：[bucket/putLifecycle.go](./example/bucket/putLifecycle.go)）
* [x] Put Bucket Tagging（使用示例：[bucket/putTagging.go](./example/bucket/putTagging.go)）
* [x] Delete Bucket（使用示例：[bucket/delete.go](./example/bucket/delete.go)）
* [x] Delete Bucket CORS（使用示例：[bucket/deleteCORS.go](./example/bucket/deleteCORS.go)）
* [x] Delete Bucket Lifecycle（使用示例：[bucket/deleteLifecycle.go](./example/bucket/deleteLifecycle.go)）
* [x] Delete Bucket Tagging（使用示例：[bucket/deleteTagging.go](./example/bucket/deleteTagging.go)）
* [x] Head Bucket（使用示例：[bucket/head.go](./example/bucket/head.go)）
* [x] List Multipart Uploads（使用示例：[bucket/listMultipartUploads.go](./example/bucket/listMultipartUploads.go)）

Object API:

* [x] Get Object（使用示例：[object/get.go](./example/object/get.go)）
* [x] Get Object ACL（使用示例：[object/getACL.go](./example/object/getACL.go)）
* [x] Put Object（使用示例：[object/put.go](./example/object/put.go)）
* [x] Put Object ACL（使用示例：[object/putACL.go](./example/object/putACL.go)）
* [x] Put Object Copy（使用示例：[object/copy.go](./example/object/copy.go)）
* [x] Delete Object（使用示例：[object/delete.go](./example/object/delete.go)）
* [x] Delete Multiple Object（使用示例：[object/deleteMultiple.go](./example/object/deleteMultiple.go)）
* [x] Head Object（使用示例：[object/head.go](./example/object/head.go)）
* [x] Options Object（使用示例：[object/options.go](./example/object/options.go)）
* [x] Initiate Multipart Upload（使用示例：[object/initiateMultipartUpload.go](./example/object/initiateMultipartUpload.go)）
* [x] Upload Part（使用示例：[object/uploadPart.go](./example/object/uploadPart.go)）
* [x] List Parts（使用示例：[object/listParts.go](./example/object/listParts.go)）
* [x] Complete Multipart Upload（使用示例：[object/completeMultipartUpload.go](./example/object/completeMultipartUpload.go)）
* [x] Abort Multipart Upload（使用示例：[object/abortMultipartUpload.go](./example/object/abortMultipartUpload.go)）
* [x] Mutipart Upload（使用示例：[object/MutiUpload.go](./example/object/MutiUpload.go)）
