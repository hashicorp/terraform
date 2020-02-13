package cos

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"time"
)

// ObjectService 相关 API
type ObjectService service

// ObjectGetOptions is the option of GetObject
type ObjectGetOptions struct {
	ResponseContentType        string `url:"response-content-type,omitempty" header:"-"`
	ResponseContentLanguage    string `url:"response-content-language,omitempty" header:"-"`
	ResponseExpires            string `url:"response-expires,omitempty" header:"-"`
	ResponseCacheControl       string `url:"response-cache-control,omitempty" header:"-"`
	ResponseContentDisposition string `url:"response-content-disposition,omitempty" header:"-"`
	ResponseContentEncoding    string `url:"response-content-encoding,omitempty" header:"-"`
	Range                      string `url:"-" header:"Range,omitempty"`
	IfModifiedSince            string `url:"-" header:"If-Modified-Since,omitempty"`
}

// presignedURLTestingOptions is the opt of presigned url
type presignedURLTestingOptions struct {
	authTime *AuthTime
}

// Get Object 请求可以将一个文件（Object）下载至本地。
// 该操作需要对目标 Object 具有读权限或目标 Object 对所有人都开放了读权限（公有读）。
//
// https://www.qcloud.com/document/product/436/7753
func (s *ObjectService) Get(ctx context.Context, name string, opt *ObjectGetOptions, id ...string) (*Response, error) {
	var u string
	if len(id) == 1 {
		u = fmt.Sprintf("/%s?versionId=%s", encodeURIComponent(name), id[0])
	} else if len(id) == 0 {
		u = "/" + encodeURIComponent(name)
	} else {
		return nil, errors.New("wrong params")
	}

	sendOpt := sendOptions{
		baseURL:          s.client.BaseURL.BucketURL,
		uri:              u,
		method:           http.MethodGet,
		optQuery:         opt,
		optHeader:        opt,
		disableCloseBody: true,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return resp, err
}

// GetToFile download the object to local file
func (s *ObjectService) GetToFile(ctx context.Context, name, localpath string, opt *ObjectGetOptions, id ...string) (*Response, error) {
	resp, err := s.Get(ctx, name, opt, id...)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	// If file exist, overwrite it
	fd, err := os.OpenFile(localpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	if err != nil {
		return resp, err
	}

	_, err = io.Copy(fd, resp.Body)
	fd.Close()
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// GetPresignedURL get the object presigned to down or upload file by url
func (s *ObjectService) GetPresignedURL(ctx context.Context, httpMethod, name, ak, sk string, expired time.Duration, opt interface{}) (*url.URL, error) {
	sendOpt := sendOptions{
		baseURL:   s.client.BaseURL.BucketURL,
		uri:       "/" + encodeURIComponent(name),
		method:    httpMethod,
		optQuery:  opt,
		optHeader: opt,
	}
	req, err := s.client.newRequest(ctx, sendOpt.baseURL, sendOpt.uri, sendOpt.method, sendOpt.body, sendOpt.optQuery, sendOpt.optHeader)
	if err != nil {
		return nil, err
	}

	var authTime *AuthTime
	if opt != nil {
		if opt, ok := opt.(*presignedURLTestingOptions); ok {
			authTime = opt.authTime
		}
	}
	if authTime == nil {
		authTime = NewAuthTime(expired)
	}
	authorization := newAuthorization(ak, sk, req, authTime)
	sign := encodeURIComponent(authorization)

	if req.URL.RawQuery == "" {
		req.URL.RawQuery = fmt.Sprintf("sign=%s", sign)
	} else {
		req.URL.RawQuery = fmt.Sprintf("%s&sign=%s", req.URL.RawQuery, sign)
	}
	return req.URL, nil

}

// ObjectPutHeaderOptions the options of header of the put object
type ObjectPutHeaderOptions struct {
	CacheControl       string `header:"Cache-Control,omitempty" url:"-"`
	ContentDisposition string `header:"Content-Disposition,omitempty" url:"-"`
	ContentEncoding    string `header:"Content-Encoding,omitempty" url:"-"`
	ContentType        string `header:"Content-Type,omitempty" url:"-"`
	ContentMD5         string `header:"Content-MD5,omitempty" url:"-"`
	ContentLength      int    `header:"Content-Length,omitempty" url:"-"`
	Expect             string `header:"Expect,omitempty" url:"-"`
	Expires            string `header:"Expires,omitempty" url:"-"`
	XCosContentSHA1    string `header:"x-cos-content-sha1,omitempty" url:"-"`
	// 自定义的 x-cos-meta-* header
	XCosMetaXXX      *http.Header `header:"x-cos-meta-*,omitempty" url:"-"`
	XCosStorageClass string       `header:"x-cos-storage-class,omitempty" url:"-"`
	// 可选值: Normal, Appendable
	//XCosObjectType string `header:"x-cos-object-type,omitempty" url:"-"`
	// Enable Server Side Encryption, Only supported: AES256
	XCosServerSideEncryption string `header:"x-cos-server-side-encryption,omitempty" url:"-" xml:"-"`
}

// ObjectPutOptions the options of put object
type ObjectPutOptions struct {
	*ACLHeaderOptions       `header:",omitempty" url:"-" xml:"-"`
	*ObjectPutHeaderOptions `header:",omitempty" url:"-" xml:"-"`
}

// Put Object请求可以将一个文件（Oject）上传至指定Bucket。
//
// 当 r 不是 bytes.Buffer/bytes.Reader/strings.Reader 时，必须指定 opt.ObjectPutHeaderOptions.ContentLength
//
// https://www.qcloud.com/document/product/436/7749
func (s *ObjectService) Put(ctx context.Context, name string, r io.Reader, opt *ObjectPutOptions) (*Response, error) {
	sendOpt := sendOptions{
		baseURL:   s.client.BaseURL.BucketURL,
		uri:       "/" + encodeURIComponent(name),
		method:    http.MethodPut,
		body:      r,
		optHeader: opt,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return resp, err
}

// PutFromFile put object from local file
// Notice that when use this put large file need set non-body of debug req/resp, otherwise will out of memory
func (s *ObjectService) PutFromFile(ctx context.Context, name string, filePath string, opt *ObjectPutOptions) (*Response, error) {
	fd, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	return s.Put(ctx, name, fd, opt)
}

// ObjectCopyHeaderOptions is the head option of the Copy
type ObjectCopyHeaderOptions struct {
	// When use replace directive to update meta infos
	CacheControl                    string `header:"Cache-Control,omitempty" url:"-"`
	ContentDisposition              string `header:"Content-Disposition,omitempty" url:"-"`
	ContentEncoding                 string `header:"Content-Encoding,omitempty" url:"-"`
	ContentType                     string `header:"Content-Type,omitempty" url:"-"`
	Expires                         string `header:"Expires,omitempty" url:"-"`
	Expect                          string `header:"Expect,omitempty" url:"-"`
	XCosMetadataDirective           string `header:"x-cos-metadata-directive,omitempty" url:"-" xml:"-"`
	XCosCopySourceIfModifiedSince   string `header:"x-cos-copy-source-If-Modified-Since,omitempty" url:"-" xml:"-"`
	XCosCopySourceIfUnmodifiedSince string `header:"x-cos-copy-source-If-Unmodified-Since,omitempty" url:"-" xml:"-"`
	XCosCopySourceIfMatch           string `header:"x-cos-copy-source-If-Match,omitempty" url:"-" xml:"-"`
	XCosCopySourceIfNoneMatch       string `header:"x-cos-copy-source-If-None-Match,omitempty" url:"-" xml:"-"`
	XCosStorageClass                string `header:"x-cos-storage-class,omitempty" url:"-" xml:"-"`
	// 自定义的 x-cos-meta-* header
	XCosMetaXXX              *http.Header `header:"x-cos-meta-*,omitempty" url:"-"`
	XCosCopySource           string       `header:"x-cos-copy-source" url:"-" xml:"-"`
	XCosServerSideEncryption string       `header:"x-cos-server-side-encryption,omitempty" url:"-" xml:"-"`
}

// ObjectCopyOptions is the option of Copy, choose header or body
type ObjectCopyOptions struct {
	*ObjectCopyHeaderOptions `header:",omitempty" url:"-" xml:"-"`
	*ACLHeaderOptions        `header:",omitempty" url:"-" xml:"-"`
}

// ObjectCopyResult is the result of Copy
type ObjectCopyResult struct {
	XMLName      xml.Name `xml:"CopyObjectResult"`
	ETag         string   `xml:"ETag,omitempty"`
	LastModified string   `xml:"LastModified,omitempty"`
}

// Copy 调用 PutObjectCopy 请求实现将一个文件从源路径复制到目标路径。建议文件大小 1M 到 5G，
// 超过 5G 的文件请使用分块上传 Upload - Copy。在拷贝的过程中，文件元属性和 ACL 可以被修改。
//
// 用户可以通过该接口实现文件移动，文件重命名，修改文件属性和创建副本。
//
// 注意：在跨帐号复制的时候，需要先设置被复制文件的权限为公有读，或者对目标帐号赋权，同帐号则不需要。
//
// https://cloud.tencent.com/document/product/436/10881
func (s *ObjectService) Copy(ctx context.Context, name, sourceURL string, opt *ObjectCopyOptions) (*ObjectCopyResult, *Response, error) {
	var res ObjectCopyResult
	if opt == nil {
		opt = new(ObjectCopyOptions)
	}
	if opt.ObjectCopyHeaderOptions == nil {
		opt.ObjectCopyHeaderOptions = new(ObjectCopyHeaderOptions)
	}
	opt.XCosCopySource = encodeURIComponent(sourceURL)

	sendOpt := sendOptions{
		baseURL:   s.client.BaseURL.BucketURL,
		uri:       "/" + encodeURIComponent(name),
		method:    http.MethodPut,
		body:      nil,
		optHeader: opt,
		result:    &res,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	// If the error occurs during the copy operation, the error response is embedded in the 200 OK response. This means that a 200 OK response can contain either a success or an error.
	if err == nil && resp.StatusCode == 200 {
		if res.ETag == "" {
			return &res, resp, errors.New("response 200 OK, but body contains an error")
		}
	}
	return &res, resp, err
}

// Delete Object请求可以将一个文件（Object）删除。
//
// https://www.qcloud.com/document/product/436/7743
func (s *ObjectService) Delete(ctx context.Context, name string) (*Response, error) {
	// When use "" string might call the delete bucket interface
	if len(name) == 0 {
		return nil, errors.New("empty object name")
	}

	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/" + encodeURIComponent(name),
		method:  http.MethodDelete,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return resp, err
}

// ObjectHeadOptions is the option of HeadObject
type ObjectHeadOptions struct {
	IfModifiedSince string `url:"-" header:"If-Modified-Since,omitempty"`
}

// Head Object请求可以取回对应Object的元数据，Head的权限与Get的权限一致
//
// https://www.qcloud.com/document/product/436/7745
func (s *ObjectService) Head(ctx context.Context, name string, opt *ObjectHeadOptions, id ...string) (*Response, error) {
	var u string
	if len(id) == 1 {
		u = fmt.Sprintf("/%s?versionId=%s", encodeURIComponent(name), id[0])
	} else if len(id) == 0 {
		u = "/" + encodeURIComponent(name)
	} else {
		return nil, errors.New("wrong params")
	}

	sendOpt := sendOptions{
		baseURL:   s.client.BaseURL.BucketURL,
		uri:       u,
		method:    http.MethodHead,
		optHeader: opt,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	if resp != nil && resp.Header["X-Cos-Object-Type"] != nil && resp.Header["X-Cos-Object-Type"][0] == "appendable" {
		resp.Header.Add("x-cos-next-append-position", resp.Header["Content-Length"][0])
	}

	return resp, err
}

// ObjectOptionsOptions is the option of object options
type ObjectOptionsOptions struct {
	Origin                      string `url:"-" header:"Origin"`
	AccessControlRequestMethod  string `url:"-" header:"Access-Control-Request-Method"`
	AccessControlRequestHeaders string `url:"-" header:"Access-Control-Request-Headers,omitempty"`
}

// Options Object请求实现跨域访问的预请求。即发出一个 OPTIONS 请求给服务器以确认是否可以进行跨域操作。
//
// 当CORS配置不存在时，请求返回403 Forbidden。
//
// https://www.qcloud.com/document/product/436/8288
func (s *ObjectService) Options(ctx context.Context, name string, opt *ObjectOptionsOptions) (*Response, error) {
	sendOpt := sendOptions{
		baseURL:   s.client.BaseURL.BucketURL,
		uri:       "/" + encodeURIComponent(name),
		method:    http.MethodOptions,
		optHeader: opt,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return resp, err
}

// CASJobParameters support three way: Standard(in 35 hours), Expedited(quick way, in 15 mins), Bulk(in 5-12 hours_
type CASJobParameters struct {
	Tier string `xml:"Tier"`
}

// ObjectRestoreOptions is the option of object restore
type ObjectRestoreOptions struct {
	XMLName xml.Name          `xml:"RestoreRequest"`
	Days    int               `xml:"Days"`
	Tier    *CASJobParameters `xml:"CASJobParameters"`
}

// PutRestore API can recover an object of type archived by COS archive.
//
// https://cloud.tencent.com/document/product/436/12633
func (s *ObjectService) PostRestore(ctx context.Context, name string, opt *ObjectRestoreOptions) (*Response, error) {
	u := fmt.Sprintf("/%s?restore", encodeURIComponent(name))
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     u,
		method:  http.MethodPost,
		body:    opt,
	}
	resp, err := s.client.send(ctx, &sendOpt)

	return resp, err
}

// TODO Append 接口在优化未开放使用
//
// Append请求可以将一个文件（Object）以分块追加的方式上传至 Bucket 中。使用Append Upload的文件必须事前被设定为Appendable。
// 当Appendable的文件被执行Put Object的操作以后，文件被覆盖，属性改变为Normal。
//
// 文件属性可以在Head Object操作中被查询到，当您发起Head Object请求时，会返回自定义Header『x-cos-object-type』，该Header只有两个枚举值：Normal或者Appendable。
//
// 追加上传建议文件大小1M - 5G。如果position的值和当前Object的长度不致，COS会返回409错误。
// 如果Append一个Normal的Object，COS会返回409 ObjectNotAppendable。
//
// Appendable的文件不可以被复制，不参与版本管理，不参与生命周期管理，不可跨区域复制。
//
// 当 r 不是 bytes.Buffer/bytes.Reader/strings.Reader 时，必须指定 opt.ObjectPutHeaderOptions.ContentLength
//
// https://www.qcloud.com/document/product/436/7741
// func (s *ObjectService) Append(ctx context.Context, name string, position int, r io.Reader, opt *ObjectPutOptions) (*Response, error) {
// 	u := fmt.Sprintf("/%s?append&position=%d", encodeURIComponent(name), position)
// 	if position != 0{
// 		opt = nil
// 	}
// 	sendOpt := sendOptions{
// 		baseURL:   s.client.BaseURL.BucketURL,
// 		uri:       u,
// 		method:    http.MethodPost,
// 		optHeader: opt,
// 		body:      r,
// 	}
// 	resp, err := s.client.send(ctx, &sendOpt)
// 	return resp, err
// }

// ObjectDeleteMultiOptions is the option of DeleteMulti
type ObjectDeleteMultiOptions struct {
	XMLName xml.Name `xml:"Delete" header:"-"`
	Quiet   bool     `xml:"Quiet" header:"-"`
	Objects []Object `xml:"Object" header:"-"`
	//XCosSha1 string `xml:"-" header:"x-cos-sha1"`
}

// ObjectDeleteMultiResult is the result of DeleteMulti
type ObjectDeleteMultiResult struct {
	XMLName        xml.Name `xml:"DeleteResult"`
	DeletedObjects []Object `xml:"Deleted,omitempty"`
	Errors         []struct {
		Key     string
		Code    string
		Message string
	} `xml:"Error,omitempty"`
}

// DeleteMulti 请求实现批量删除文件，最大支持单次删除1000个文件。
// 对于返回结果，COS提供Verbose和Quiet两种结果模式。Verbose模式将返回每个Object的删除结果；
// Quiet模式只返回报错的Object信息。
// https://www.qcloud.com/document/product/436/8289
func (s *ObjectService) DeleteMulti(ctx context.Context, opt *ObjectDeleteMultiOptions) (*ObjectDeleteMultiResult, *Response, error) {
	var res ObjectDeleteMultiResult
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.BucketURL,
		uri:     "/?delete",
		method:  http.MethodPost,
		body:    opt,
		result:  &res,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return &res, resp, err
}

// Object is the meta info of the object
type Object struct {
	Key          string `xml:",omitempty"`
	ETag         string `xml:",omitempty"`
	Size         int    `xml:",omitempty"`
	PartNumber   int    `xml:",omitempty"`
	LastModified string `xml:",omitempty"`
	StorageClass string `xml:",omitempty"`
	Owner        *Owner `xml:",omitempty"`
}

// MultiUploadOptions is the option of the multiupload,
// ThreadPoolSize default is one
type MultiUploadOptions struct {
	OptIni         *InitiateMultipartUploadOptions
	PartSize       int64
	ThreadPoolSize int
}

type Chunk struct {
	Number int
	OffSet int64
	Size   int64
}

// jobs
type Jobs struct {
	Name       string
	UploadId   string
	FilePath   string
	RetryTimes int
	Chunk      Chunk
	Data       io.Reader
	Opt        *ObjectUploadPartOptions
}

type Results struct {
	PartNumber int
	Resp       *Response
}

func worker(s *ObjectService, jobs <-chan *Jobs, results chan<- *Results) {
	for j := range jobs {
		fd, err := os.Open(j.FilePath)
		var res Results
		if err != nil {
			res.PartNumber = j.Chunk.Number
			res.Resp = nil
			results <- &res
		}

		fd.Seek(j.Chunk.OffSet, os.SEEK_SET)
		// UploadPart do not support the chunk trsf, so need to add the content-length
		opt := &ObjectUploadPartOptions{
			ContentLength: int(j.Chunk.Size),
		}

		rt := j.RetryTimes
		for {
			resp, err := s.UploadPart(context.Background(), j.Name, j.UploadId, j.Chunk.Number,
				&io.LimitedReader{R: fd, N: j.Chunk.Size}, opt)
			res.PartNumber = j.Chunk.Number
			res.Resp = resp
			if err != nil {
				rt--
				if rt == 0 {
					fd.Close()
					results <- &res
					break
				}
				continue
			}
			fd.Close()
			results <- &res
			break
		}
	}
}

func SplitFileIntoChunks(filePath string, partSize int64) ([]Chunk, int, error) {
	if filePath == "" || partSize <= 0 {
		return nil, 0, errors.New("chunkSize invalid")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	var partNum = stat.Size() / partSize
	// 10000 max part size
	if partNum >= 10000 {
		return nil, 0, errors.New("Too many parts, out of 10000")
	}

	var chunks []Chunk
	var chunk = Chunk{}
	for i := int64(0); i < partNum; i++ {
		chunk.Number = int(i + 1)
		chunk.OffSet = i * partSize
		chunk.Size = partSize
		chunks = append(chunks, chunk)
	}

	if stat.Size()%partSize > 0 {
		chunk.Number = len(chunks) + 1
		chunk.OffSet = int64(len(chunks)) * partSize
		chunk.Size = stat.Size() % partSize
		chunks = append(chunks, chunk)
		partNum++
	}

	return chunks, int(partNum), nil

}

// MultiUpload 为高级upload接口，并发分块上传
// 注意该接口目前只供参考
//
// 需要指定分块大小 partSize >= 1 ,单位为MB
// 同时请确认分块数量不超过10000
//

func (s *ObjectService) MultiUpload(ctx context.Context, name string, filepath string, opt *MultiUploadOptions) (*CompleteMultipartUploadResult, *Response, error) {
	// 1.Get the file chunk
	bufSize := opt.PartSize * 1024 * 1024
	chunks, partNum, err := SplitFileIntoChunks(filepath, bufSize)
	if err != nil {
		return nil, nil, err
	}

	// 2.Init
	optini := opt.OptIni
	res, _, err := s.InitiateMultipartUpload(ctx, name, optini)
	if err != nil {
		return nil, nil, err
	}
	uploadID := res.UploadID
	var poolSize int
	if opt.ThreadPoolSize > 0 {
		poolSize = opt.ThreadPoolSize
	} else {
		// Default is one
		poolSize = 1
	}

	chjobs := make(chan *Jobs, 100)
	chresults := make(chan *Results, 10000)
	optcom := &CompleteMultipartUploadOptions{}

	// 3.Start worker
	for w := 1; w <= poolSize; w++ {
		go worker(s, chjobs, chresults)
	}

	// 4.Push jobs
	for _, chunk := range chunks {
		job := &Jobs{
			Name:       name,
			RetryTimes: 3,
			FilePath:   filepath,
			UploadId:   uploadID,
			Chunk:      chunk,
		}
		chjobs <- job
	}
	close(chjobs)

	// 5.Recv the resp etag to complete
	for i := 1; i <= partNum; i++ {
		res := <-chresults
		// Notice one part fail can not get the etag according.
		if res.Resp == nil {
			// Some part already fail, can not to get the header inside.
			return nil, nil, fmt.Errorf("UploadID %s, part %d failed to get resp content.", uploadID, res.PartNumber)
		}
		// Notice one part fail can not get the etag according.
		etag := res.Resp.Header.Get("ETag")
		optcom.Parts = append(optcom.Parts, Object{
			PartNumber: res.PartNumber, ETag: etag},
		)
	}
	sort.Sort(ObjectList(optcom.Parts))

	v, resp, err := s.CompleteMultipartUpload(context.Background(), name, uploadID, optcom)

	return v, resp, err
}
