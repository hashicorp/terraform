package backblaze

import (
	"errors"
	"net/url"
)

// Bucket provides access to the files stored in a B2 Bucket
type Bucket struct {
	*BucketInfo

	b2 *B2
}

// UploadAuth encapsulates the details needed to upload a file to B2
type UploadAuth struct {
	AuthorizationToken string
	UploadURL          *url.URL
	Valid              bool
}

// CreateBucket creates a new B2 Bucket in the authorized account.
//
// Buckets can be named. The name must be globally unique. No account can use
// a bucket with the same name. Buckets are assigned a unique bucketId which
// is used when uploading, downloading, or deleting files.
func (b *B2) CreateBucket(bucketName string, bucketType BucketType) (*Bucket, error) {
	request := &createBucketRequest{
		AccountID:  b.AccountID,
		BucketName: bucketName,
		BucketType: bucketType,
	}
	response := &BucketInfo{}

	if err := b.apiRequest("b2_create_bucket", request, response); err != nil {
		return nil, err
	}

	bucket := &Bucket{
		BucketInfo: response,
		b2:         b,
	}

	return bucket, nil
}

// deleteBucket removes the specified bucket from the authorized account. Only
// buckets that contain no version of any files can be deleted.
func (b *B2) deleteBucket(bucketID string) (*Bucket, error) {
	request := &deleteBucketRequest{
		AccountID: b.AccountID,
		BucketID:  bucketID,
	}
	response := &BucketInfo{}

	if err := b.apiRequest("b2_delete_bucket", request, response); err != nil {
		return nil, err
	}

	return &Bucket{
		BucketInfo: response,
		b2:         b,
	}, nil
}

// Delete removes removes the bucket from the authorized account. Only buckets
// that contain no version of any files can be deleted.
func (b *Bucket) Delete() error {
	_, error := b.b2.deleteBucket(b.ID)
	return error
}

// ListBuckets lists buckets associated with an account, in alphabetical order
// by bucket ID.
func (b *B2) ListBuckets() ([]*Bucket, error) {
	request := &accountRequest{
		ID: b.AccountID,
	}
	response := &listBucketsResponse{}

	if err := b.apiRequest("b2_list_buckets", request, response); err != nil {
		return nil, err
	}

	// Construct bucket list
	buckets := make([]*Bucket, len(response.Buckets))
	for i, info := range response.Buckets {
		bucket := &Bucket{
			BucketInfo: info,
			b2:         b,
		}

		switch info.BucketType {
		case AllPublic:
		case AllPrivate:
		case Snapshot:
		default:
			return nil, errors.New("Uncrecognised bucket type: " + string(bucket.BucketType))
		}

		buckets[i] = bucket
	}

	return buckets, nil
}

// updateBucket allows the bucket type to be changed
func (b *B2) updateBucket(bucketID string, bucketType BucketType) (*Bucket, error) {
	request := &updateBucketRequest{
		ID:         bucketID,
		BucketType: bucketType,
	}
	response := &BucketInfo{}

	if err := b.apiRequest("b2_update_bucket", request, response); err != nil {
		return nil, err
	}

	return &Bucket{
		BucketInfo: response,
		b2:         b,
	}, nil
}

// Update allows the bucket type to be changed
func (b *Bucket) Update(bucketType BucketType) error {
	_, error := b.b2.updateBucket(b.ID, bucketType)
	return error
}

// Bucket looks up a bucket for the currently authorized client
func (b *B2) Bucket(bucketName string) (*Bucket, error) {
	buckets, err := b.ListBuckets()
	if err != nil {
		return nil, err
	}

	for _, bucket := range buckets {
		if bucket.Name == bucketName {
			return bucket, nil
		}
	}

	return nil, nil
}

// GetUploadAuth retrieves the URL to use for uploading files.
//
// When you upload a file to B2, you must call b2_get_upload_url first to get
// the URL for uploading directly to the place where the file will be stored.
func (b *Bucket) GetUploadAuth() (*UploadAuth, error) {
	request := &bucketRequest{
		ID: b.ID,
	}

	response := &getUploadURLResponse{}
	if err := b.b2.apiRequest("b2_get_upload_url", request, response); err != nil {
		return nil, err
	}

	// Set bucket auth
	uploadURL, err := url.Parse(response.UploadURL)
	if err != nil {
		return nil, err
	}
	auth := &UploadAuth{
		AuthorizationToken: response.AuthorizationToken,
		UploadURL:          uploadURL,
		Valid:              true,
	}

	return auth, nil
}
