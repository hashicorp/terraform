package remote

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	minio "github.com/minio/minio-go"
)

const CONTENT_TYPE = "application/json"

// MinioClient implements the Client interface for the S3-compatible
// Minio Cloud Storage service (also works with Ceph radosgw and AWS)
type MinioClient struct {
	client          *minio.Client
	endpoint        string
	accessKeyID     string
	secretAccessKey string
	bucketName      string
	bucketLocation  string
	objectName      string
	useSSL          bool
}

func minioFactory(conf map[string]string) (Client, error) {
	c := &MinioClient{}

	endpoint, ok := conf["endpoint"]
	if !ok {
		endpoint = os.Getenv("MINIO_ENDPOINT")
		if endpoint == "" {
			return nil, fmt.Errorf("missing 'endpoint' configuration or MINIO_ENDPOINT environment variable")
		}
	}
	c.endpoint = endpoint

	accessKeyID, ok := conf["access_key_id"]
	if !ok {
		accessKeyID = os.Getenv("MINIO_ACCESS_KEY_ID")
		if accessKeyID == "" {
			return nil, fmt.Errorf("missing 'access_key_id' configuration or MINIO_ACCESS_KEY_ID environment variable")
		}
	}
	c.accessKeyID = accessKeyID

	secretAccessKey, ok := conf["secret_access_key"]
	if !ok {
		secretAccessKey = os.Getenv("MINIO_SECRET_ACCESS_KEY")
		if secretAccessKey == "" {
			return nil, fmt.Errorf("missing 'secret_access_key' configuration or MINIO_SECRET_ACCESS_KEY environment variable")
		}
	}
	c.secretAccessKey = secretAccessKey

	bucketName, ok := conf["bucket_name"]
	if !ok {
		bucketName = os.Getenv("MINIO_BUCKET_NAME")
		if bucketName == "" {
			return nil, fmt.Errorf("missing 'bucket_name' configuration or MINIO_BUCKET_NAME environment variable")
		}
	}
	c.bucketName = bucketName

	bucketLocation, ok := conf["bucket_location"]
	if !ok {
		bucketLocation = os.Getenv("MINIO_BUCKET_LOCATION")
	}
	c.bucketLocation = bucketLocation

	objectName, ok := conf["object_name"]
	if !ok {
		objectName = os.Getenv("MINIO_OBJECT_NAME")
		if objectName == "" {
			return nil, fmt.Errorf("missing 'object_name' configuration or MINIO_OBJECT_NAME environment variable")
		}
	}
	c.objectName = objectName

	c.useSSL = true
	useSSL, ok := conf["use_ssl"]
	if !ok {
		useSSL = os.Getenv("MINIO_USE_SSL")
	}
	if useSSL != "" {
		v, err := strconv.ParseBool(useSSL)
		if err != nil {
			return nil, fmt.Errorf("'use_ssl' or 'MINIO_USE_SSL' could not be parsed as bool: %s", err)
		}
		c.useSSL = v
	}

	minioClient, err := minio.New(c.endpoint, c.accessKeyID, c.secretAccessKey, c.useSSL)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Minio client: %v", err)
	}
	c.client = minioClient
	
	return c, nil
}

func (c *MinioClient) Get() (*Payload, error) {
	object, err := c.client.GetObject(c.bucketName, c.objectName)
	if err != nil {
		return nil, fmt.Errorf("GetObject failed: %v", err)
	}
	defer object.Close()

	bytes, err := ioutil.ReadAll(object)
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			// object does not exist
			return nil, nil
		}
		return nil, fmt.Errorf("Failed to ReadAll from object %v %v: %v" , c.bucketName, c.objectName, err)
	}
	
	hash := md5.Sum(bytes)
	payload := &Payload{
		Data: bytes,
		MD5:  hash[:md5.Size],
	}
	return payload, nil
}

func (c *MinioClient) Put(data []byte) error {
	if err := c.ensureBucketExists(); err != nil {
		return fmt.Errorf("Failed to ensure bucket exists: %v", err)
	}

	reader := bytes.NewReader(data)

	_, err := c.client.PutObject(c.bucketName, c.objectName, reader, CONTENT_TYPE)
	if err != nil {
		return fmt.Errorf("Failed to PutObject: %v", err)
	}
	
	return err
}

func (c *MinioClient) Delete() error {
	err := c.client.RemoveObject(c.bucketName, c.objectName)
	if err != nil {
		return fmt.Errorf("Failed to RemoveObject: %v", err)
	}
	
	return err
}

func (c *MinioClient) ensureBucketExists() error {
	found, err := c.client.BucketExists(c.bucketName)
	if err != nil {
		return fmt.Errorf("Failed BucketExists check: %v", err)
	}
	if !found {
		log.Printf("Creating Minio bucket %s at location %s", c.bucketName, c.bucketLocation)
		err = c.client.MakeBucket(c.bucketName, c.bucketLocation)
		if err != nil {
			return fmt.Errorf("Failed to MakeBucket: %v", err)
		}
	}
	return nil
}

