package remote

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
)

type S3RemoteClient struct {
	Bucket *s3.Bucket
	Path   string
}

func getRegion(conf map[string]string) (aws.Region, error) {
	regionName, ok := conf["region"]
	if !ok || regionName == "" {
		regionName = os.Getenv("AWS_DEFAULT_REGION")
		if regionName == "" {
			return aws.Region{}, fmt.Errorf("AWS region not set")
		}
	}

	region, ok := aws.Regions[regionName]
	if !ok {
		return aws.Region{}, fmt.Errorf("AWS region set in configuration '%v' doesn't exist", regionName)
	}
	return region, nil
}

func getBucketAndPath(address string) (string, string, error) {
	re := regexp.MustCompile("^s3://([^/]+)/(.+)$")
	matches := re.FindStringSubmatch(address)
	if len(matches) < 3 {
		return "", "", fmt.Errorf("Address for s3 should be of form: s3://<bucket_name>/<path>")
	}
	return matches[1], matches[2], nil
}

func NewS3RemoteClient(conf map[string]string) (*S3RemoteClient, error) {
	client := &S3RemoteClient{}

	auth, err := aws.GetAuth(conf["access_token"], conf["secret_token"], "", time.Now())
	if err != nil {
		return nil, err
	}

	region, err := getRegion(conf)
	if err != nil {
		return nil, err
	}

	address, ok := conf["address"]
	if !ok {
		return nil, fmt.Errorf("'address' configuration not set for S3 remote state storage backend")
	}
	bucketName, path, err := getBucketAndPath(address)
	if err != nil {
		return nil, err
	}
	client.Bucket = s3.New(auth, region).Bucket(bucketName)
	client.Path = path

	return client, nil
}

func (c *S3RemoteClient) GetState() (*RemoteStatePayload, error) {
	resp, err := c.Bucket.GetResponse(c.Path)
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	if err != nil {
		switch err.(type) {
		case *s3.Error:
			s3Err := err.(*s3.Error)

			// FIXME copied from Atlas
			// Handle the common status codes
			switch s3Err.StatusCode {
			case http.StatusNoContent:
				return nil, nil
			case http.StatusNotFound:
				return nil, nil
			case http.StatusUnauthorized:
				return nil, ErrRequireAuth
			case http.StatusForbidden:
				return nil, ErrInvalidAuth
			case http.StatusInternalServerError:
				return nil, ErrRemoteInternal
			default:
				return nil, fmt.Errorf("Unexpected HTTP response code %d", s3Err.StatusCode)
			}
		default:
			return nil, err
		}
	}

	// Read in the body
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, fmt.Errorf("Failed to read remote state: %v", err)
	}

	// Create the payload
	payload := &RemoteStatePayload{
		State: buf.Bytes(),
	}

	// Check for the MD5
	if raw := resp.Header.Get("Content-MD5"); raw != "" {
		md5, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("Failed to decode Content-MD5 '%s': %v", raw, err)
		}
		payload.MD5 = md5

	} else {
		// Generate the MD5
		hash := md5.Sum(payload.State)
		payload.MD5 = hash[:md5.Size]
	}

	return payload, nil
}

func (c *S3RemoteClient) PutState(state []byte, force bool) error {
	// Generate the MD5
	hash := md5.Sum(state)
	b64 := base64.StdEncoding.EncodeToString(hash[:md5.Size])

	options := s3.Options{
		ContentMD5: b64,
	}

	err := c.Bucket.Put(c.Path, state, "application/json", s3.Private, options)
	switch err.(type) {
	case *s3.Error:
		s3Err := err.(*s3.Error)

		// Handle the error codes
		switch s3Err.StatusCode {
		case http.StatusUnauthorized:
			return ErrRequireAuth
		case http.StatusForbidden:
			return ErrInvalidAuth
		case http.StatusInternalServerError:
			return ErrRemoteInternal
		default:
			return fmt.Errorf("Unexpected HTTP response code %d", s3Err.StatusCode)
		}
	default:
		return err
	}
}

func (c *S3RemoteClient) DeleteState() error {
	err := c.Bucket.Del(c.Path)
	switch err.(type) {
	case *s3.Error:
		s3Err := err.(*s3.Error)
		// Handle the error codes
		switch s3Err.StatusCode {
		case http.StatusNoContent:
			return nil
		case http.StatusNotFound:
			return nil
		case http.StatusUnauthorized:
			return ErrRequireAuth
		case http.StatusForbidden:
			return ErrInvalidAuth
		case http.StatusInternalServerError:
			return ErrRemoteInternal
		default:
			return fmt.Errorf("Unexpected HTTP response code %d", s3Err.StatusCode)
		}
	default:
		return err
	}
}
