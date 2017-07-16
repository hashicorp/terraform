//go:generate ffjson $GOFILE

package backblaze

// B2Error encapsulates an error message returned by the B2 API.
//
// Failures to connect to the B2 servers, and networking problems in general can cause errors
type B2Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e B2Error) Error() string {
	return e.Code + ": " + e.Message
}

// IsFatal returns true if this error represents
// an error which can't be recovered from by retrying
func (e *B2Error) IsFatal() bool {
	switch {
	case e.Status == 401: // Unauthorized
		switch e.Code {
		case "expired_auth_token":
			return false
		case "missing_auth_token", "bad_auth_token":
			return true
		default:
			return true
		}
	case e.Status == 408: // Timeout
		return false
	case e.Status >= 500 && e.Status < 600: // Server error
		return false
	default:
		return true
	}
}

type authorizeAccountResponse struct {
	AccountID          string `json:"accountId"`
	APIEndpoint        string `json:"apiUrl"`
	AuthorizationToken string `json:"authorizationToken"`
	DownloadURL        string `json:"downloadUrl"`
}

type accountRequest struct {
	ID string `json:"accountId"`
}

// BucketType defines the security setting for a bucket
type BucketType string

// Buckets can be either public, private, or snapshot
const (
	AllPublic  BucketType = "allPublic"
	AllPrivate BucketType = "allPrivate"
	Snapshot   BucketType = "snapshot"
)

// BucketInfo describes a bucket
type BucketInfo struct {
	ID         string `json:"bucketId"`
	AccountID  string `json:"accountId"`
	Name       string `json:"bucketName"`
	BucketType `json:"bucketType"`
}

type bucketRequest struct {
	ID string `json:"bucketId"`
}

type createBucketRequest struct {
	AccountID  string `json:"accountId"`
	BucketName string `json:"bucketName"`
	BucketType `json:"bucketType"`
}

type deleteBucketRequest struct {
	AccountID string `json:"accountId"`
	BucketID  string `json:"bucketId"`
}

type updateBucketRequest struct {
	ID         string `json:"bucketId"`
	BucketType `json:"bucketType"`
}

type getUploadURLResponse struct {
	BucketID           string `json:"bucketId"`
	UploadURL          string `json:"uploadUrl"`
	AuthorizationToken string `json:"authorizationToken"`
}

type listBucketsResponse struct {
	Buckets []*BucketInfo `json:"buckets"`
}
type fileRequest struct {
	ID string `json:"fileId"`
}

type fileVersionRequest struct {
	Name string `json:"fileName"`
	ID   string `json:"fileId"`
}

// File descibes a file stored in a B2 bucket
type File struct {
	ID            string            `json:"fileId"`
	Name          string            `json:"fileName"`
	AccountID     string            `json:"accountId"`
	BucketID      string            `json:"bucketId"`
	ContentLength int64             `json:"contentLength"`
	ContentSha1   string            `json:"contentSha1"`
	ContentType   string            `json:"contentType"`
	FileInfo      map[string]string `json:"fileInfo"`
}

type listFilesRequest struct {
	BucketID      string `json:"bucketId"`
	StartFileName string `json:"startFileName"`
	MaxFileCount  int    `json:"maxFileCount"`
}

// ListFilesResponse lists a page of files stored in a B2 bucket
type ListFilesResponse struct {
	Files        []FileStatus `json:"files"`
	NextFileName string       `json:"nextFileName"`
}

type listFileVersionsRequest struct {
	BucketID      string `json:"bucketId"`
	StartFileName string `json:"startFileName,omitempty"`
	StartFileID   string `json:"startFileId,omitempty"`
	MaxFileCount  int    `json:"maxFileCount"`
}

// ListFileVersionsResponse lists a page of file versions stored in a B2 bucket
type ListFileVersionsResponse struct {
	Files        []FileStatus `json:"files"`
	NextFileName string       `json:"nextFileName"`
	NextFileID   string       `json:"nextFileId"`
}

type hideFileRequest struct {
	BucketID string `json:"bucketId"`
	FileName string `json:"fileName"`
}

// FileAction indicates the current status of a file in a B2 bucket
type FileAction string

// Files can be either uploads (visible) or hidden.
//
// Hiding a file makes it look like the file has been deleted, without
// removing any of the history. It adds a new version of the file that is a
// marker saying the file is no longer there.
const (
	Upload FileAction = "upload"
	Hide   FileAction = "hide"
)

// FileStatus describes minimal metadata about a file in a B2 bucket.
// It is returned by the ListFileNames and ListFileVersions methods
type FileStatus struct {
	FileAction      `json:"action"`
	ID              string `json:"fileId"`
	Name            string `json:"fileName"`
	Size            int    `json:"size"`
	UploadTimestamp int64  `json:"uploadTimestamp"`
}
