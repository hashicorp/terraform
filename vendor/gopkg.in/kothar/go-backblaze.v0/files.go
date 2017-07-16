package backblaze

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pquerna/ffjson/ffjson"
)

// ListFileNames lists the names of all files in a bucket, starting at a given name.
func (b *Bucket) ListFileNames(startFileName string, maxFileCount int) (*ListFilesResponse, error) {
	request := &listFilesRequest{
		BucketID:      b.ID,
		StartFileName: startFileName,
		MaxFileCount:  maxFileCount,
	}
	response := &ListFilesResponse{}

	if err := b.b2.apiRequest("b2_list_file_names", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// UploadFile calls UploadTypedFile with the b2/x-auto contentType
func (b *Bucket) UploadFile(name string, meta map[string]string, file io.Reader) (*File, error) {
	return b.UploadTypedFile(name, "b2/x-auto", meta, file)
}

// UploadTypedFile uploads a file to B2, returning its unique file ID.
// This method computes the hash of the file before passing it to UploadHashedFile
func (b *Bucket) UploadTypedFile(name, contentType string, meta map[string]string, file io.Reader) (*File, error) {

	// Hash the upload
	hash := sha1.New()

	var reader io.Reader
	var contentLength int64
	if r, ok := file.(io.ReadSeeker); ok {
		// If the input is seekable, just hash then seek back to the beginning
		written, err := io.Copy(hash, r)
		if err != nil {
			return nil, err
		}
		r.Seek(0, 0)
		reader = r
		contentLength = written
	} else {
		// If the input is not seekable, buffer it while hashing, and use the buffer as input
		buffer := &bytes.Buffer{}
		r := io.TeeReader(file, buffer)

		written, err := io.Copy(hash, r)
		if err != nil {
			return nil, err
		}
		reader = buffer
		contentLength = written
	}

	sha1Hash := hex.EncodeToString(hash.Sum(nil))
	f, err := b.UploadHashedTypedFile(name, contentType, meta, reader, sha1Hash, contentLength)

	// Retry after non-fatal errors
	if b2err, ok := err.(*B2Error); ok {
		if !b2err.IsFatal() && !b.b2.NoRetry {
			f, err = b.UploadHashedTypedFile(name, contentType, meta, reader, sha1Hash, contentLength)
		}
	}
	return f, err
}

// UploadHashedFile calls UploadHashedTypedFile with the b2/x-auto file type
func (b *Bucket) UploadHashedFile(
	name string, meta map[string]string, file io.Reader,
	sha1Hash string, contentLength int64) (*File, error) {

	return b.UploadHashedTypedFile(name, "b2/x-auto", meta, file, sha1Hash, contentLength)
}

// UploadHashedTypedFile Uploads a file to B2, returning its unique file ID.
//
// This method will not retry if the upload fails, as the reader may have consumed
// some bytes. If the error type is B2Error and IsFatal returns false, you may retry the
// upload and expect it to succeed eventually.
func (b *Bucket) UploadHashedTypedFile(
	name, contentType string, meta map[string]string, file io.Reader,
	sha1Hash string, contentLength int64) (*File, error) {

	auth, err := b.GetUploadAuth()
	if err != nil {
		return nil, err
	}

	if b.b2.Debug {
		fmt.Printf("         Upload: %s/%s\n", b.Name, name)
		fmt.Printf("           SHA1: %s\n", sha1Hash)
		fmt.Printf("  ContentLength: %d\n", contentLength)
		fmt.Printf("    ContentType: %s\n", contentType)
	}

	// Create authorized request
	req, err := http.NewRequest("POST", auth.UploadURL.String(), file)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", auth.AuthorizationToken)

	// Set file metadata
	req.ContentLength = contentLength
	// default content type
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Bz-File-Name", url.QueryEscape(name))
	req.Header.Set("X-Bz-Content-Sha1", sha1Hash)

	if meta != nil {
		for k, v := range meta {
			req.Header.Add("X-Bz-Info-"+url.QueryEscape(k), url.QueryEscape(v))
		}
	}

	resp, err := b.b2.httpClient.Do(req)
	if err != nil {
		auth.Valid = false
		return nil, err
	}

	result := &File{}

	// We are not dealing with the b2 client auth token in this case, hence the nil auth
	if err := b.b2.parseResponse(resp, result, nil); err != nil {
		auth.Valid = false
		return nil, err
	}

	if sha1Hash != result.ContentSha1 {
		return nil, errors.New("SHA1 of uploaded file does not match local hash")
	}

	return result, nil
}

// GetFileInfo retrieves information about one file stored in B2.
func (b *Bucket) GetFileInfo(fileID string) (*File, error) {
	request := &fileRequest{
		ID: fileID,
	}
	response := &File{}

	if err := b.b2.apiRequest("b2_get_file_info", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// DownloadFileByID downloads a file from B2 using its unique ID
func (c *B2) DownloadFileByID(fileID string) (*File, io.ReadCloser, error) {

	request := &fileRequest{
		ID: fileID,
	}
	requestBody, err := ffjson.Marshal(request)
	if err != nil {
		return nil, nil, err
	}

	f, body, err := c.tryDownloadFileByID(requestBody)

	// Retry after non-fatal errors
	if b2err, ok := err.(*B2Error); ok {
		if !b2err.IsFatal() && !c.NoRetry {
			return c.tryDownloadFileByID(requestBody)
		}
	}
	return f, body, err
}

func (c *B2) tryDownloadFileByID(requestBody []byte) (*File, io.ReadCloser, error) {
	resp, auth, err := c.authPost("b2_download_file_by_id", bytes.NewReader(requestBody))
	if err != nil {
		return nil, nil, err
	}
	return c.downloadFile(resp, auth)
}

// FileURL returns a URL which may be used to download the latest version of a file.
// This returned URL will only work for public buckets unless the correct authorization header is provided.
func (b *Bucket) FileURL(fileName string) (string, error) {
	fileURL, _, err := b.internalFileURL(fileName)
	return fileURL, err
}

func (b *Bucket) internalFileURL(fileName string) (string, *authorizationState, error) {
	b.b2.mutex.Lock()
	defer b.b2.mutex.Unlock()

	if !b.b2.auth.isValid() {
		if err := b.b2.internalAuthorizeAccount(); err != nil {
			return "", nil, err
		}
	}
	return b.b2.auth.DownloadURL + "/file/" + b.Name + "/" + fileName, b.b2.auth, nil
}

// DownloadFileByName Downloads one file by providing the name of the bucket and the name of the
// file.
func (b *Bucket) DownloadFileByName(fileName string) (*File, io.ReadCloser, error) {

	f, body, err := b.tryDownloadFileByName(fileName)

	// Retry after non-fatal errors
	if b2err, ok := err.(*B2Error); ok {
		if !b2err.IsFatal() && !b.b2.NoRetry {
			return b.tryDownloadFileByName(fileName)
		}
	}
	return f, body, err
}

func (b *Bucket) tryDownloadFileByName(fileName string) (*File, io.ReadCloser, error) {
	// Locate the file
	fileURL, auth, err := b.internalFileURL(fileName)
	if err != nil {
		return nil, nil, err
	}

	// Make the download request
	req, err := http.NewRequest("GET", fileURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Add("Authorization", auth.AuthorizationToken)

	resp, err := b.b2.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	// Handle the response
	return b.b2.downloadFile(resp, auth)
}

func (c *B2) downloadFile(resp *http.Response, auth *authorizationState) (*File, io.ReadCloser, error) {
	success := false
	defer func() {
		if !success {
			resp.Body.Close()
		}
	}()

	switch resp.StatusCode {
	case 200:
	case 401:
		auth.invalidate()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}
		if err := c.parseError(body); err != nil {
			return nil, nil, err
		}
		return nil, nil, &B2Error{
			Code:    "UNAUTHORIZED",
			Message: "The account ID is wrong, the account does not have B2 enabled, or the application key is not valid",
			Status:  resp.StatusCode,
		}
	default:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}
		if err := c.parseError(body); err != nil {
			return nil, nil, err
		}

		return nil, nil, fmt.Errorf("Unrecognised status code: %d", resp.StatusCode)
	}

	name, err := url.QueryUnescape(resp.Header.Get("X-Bz-File-Name"))
	if err != nil {
		return nil, nil, err
	}

	file := &File{
		ID:          resp.Header.Get("X-Bz-File-Id"),
		Name:        name,
		ContentSha1: resp.Header.Get("X-Bz-Content-Sha1"),
		ContentType: resp.Header.Get("Content-Type"),
		FileInfo:    make(map[string]string),
	}

	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return nil, nil, err
	}
	file.ContentLength = size

	for k, v := range resp.Header {
		if strings.HasPrefix(k, "X-Bz-Info-") {
			key, err := url.QueryUnescape(k[len("X-Bz-Info-"):])
			if err != nil {
				key = k[len("X-Bz-Info-"):]
				log.Printf("Unable to decode key: %q", key)
			}

			value, err := url.QueryUnescape(v[0])
			if err != nil {
				value = v[0]
				log.Printf("Unable to decode value: %q", value)
			}
			file.FileInfo[key] = value
		}
	}

	success = true // Don't close the response body
	return file, resp.Body, nil
}

// ListFileVersions lists all of the versions of all of the files contained in
// one bucket, in alphabetical order by file name, and by reverse of date/time
// uploaded for versions of files with the same name.
func (b *Bucket) ListFileVersions(startFileName, startFileID string, maxFileCount int) (*ListFileVersionsResponse, error) {
	request := &listFileVersionsRequest{
		BucketID:      b.ID,
		StartFileName: startFileName,
		StartFileID:   startFileID,
		MaxFileCount:  maxFileCount,
	}
	response := &ListFileVersionsResponse{}

	if err := b.b2.apiRequest("b2_list_file_versions", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// DeleteFileVersion deletes one version of a file from B2.
//
// If the version you delete is the latest version, and there are older
// versions, then the most recent older version will become the current
// version, and be the one that you'll get when downloading by name. See the
// File Versions page for more details.
func (b *Bucket) DeleteFileVersion(fileName, fileID string) (*FileStatus, error) {
	request := &fileVersionRequest{
		Name: fileName,
		ID:   fileID,
	}
	response := &FileStatus{}

	if err := b.b2.apiRequest("b2_delete_file_version", request, response); err != nil {
		return nil, err
	}

	return response, nil
}

// HideFile hides a file so that downloading by name will not find the file,
// but previous versions of the file are still stored. See File Versions about
// what it means to hide a file.
func (b *Bucket) HideFile(fileName string) (*FileStatus, error) {
	request := &hideFileRequest{
		BucketID: b.ID,
		FileName: fileName,
	}
	response := &FileStatus{}

	if err := b.b2.apiRequest("b2_hide_file", request, response); err != nil {
		return nil, err
	}

	return response, nil
}
