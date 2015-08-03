package remote

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
)

func TestS3Client_impl(t *testing.T) {
	var _ Client = new(S3Client)
}

func TestS3Factory(t *testing.T) {
	// This test just instantiates the client. Shouldn't make any actual
	// requests nor incur any costs.

	config := make(map[string]string)

	// Empty config is an error
	_, err := s3Factory(config)
	if err == nil {
		t.Fatalf("Empty config should be error")
	}

	config["region"] = "us-west-1"
	config["bucket"] = "foo"
	config["key"] = "bar"
	config["encrypt"] = "1"

	// For this test we'll provide the credentials as config. The
	// acceptance tests implicitly test passing credentials as
	// environment variables.
	config["access_key"] = "bazkey"
	config["secret_key"] = "bazsecret"

	client, err := s3Factory(config)
	if err != nil {
		t.Fatalf("Error for valid config")
	}

	s3Client := client.(*S3Client)

	if *s3Client.nativeClient.Config.Region != "us-west-1" {
		t.Fatalf("Incorrect region was populated")
	}
	if s3Client.bucketName != "foo" {
		t.Fatalf("Incorrect bucketName was populated")
	}
	if s3Client.keyName != "bar" {
		t.Fatalf("Incorrect keyName was populated")
	}

	credentials, err := s3Client.nativeClient.Config.Credentials.Get()
	if err != nil {
		t.Fatalf("Error when requesting credentials")
	}
	if credentials.AccessKeyID != "bazkey" {
		t.Fatalf("Incorrect Access Key Id was populated")
	}
	if credentials.SecretAccessKey != "bazsecret" {
		t.Fatalf("Incorrect Secret Access Key was populated")
	}
}

func TestS3Client(t *testing.T) {
	// This test creates a bucket in S3 and populates it.
	// It may incur costs, so it will only run if AWS credential environment
	// variables are present.

	accessKeyId := os.Getenv("AWS_ACCESS_KEY_ID")
	if accessKeyId == "" {
		t.Skipf("skipping; AWS_ACCESS_KEY_ID must be set")
	}

	regionName := os.Getenv("AWS_DEFAULT_REGION")
	if regionName == "" {
		regionName = "us-west-2"
	}

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"
	testData := []byte(`testing data`)

	config := make(map[string]string)
	config["region"] = regionName
	config["bucket"] = bucketName
	config["key"] = keyName
	config["encrypt"] = "1"

	client, err := s3Factory(config)
	if err != nil {
		t.Fatalf("Error for valid config")
	}

	s3Client := client.(*S3Client)
	nativeClient := s3Client.nativeClient

	createBucketReq := &s3.CreateBucketInput{
		Bucket: &bucketName,
	}

	// Be clear about what we're doing in case the user needs to clean
	// this up later.
	t.Logf("Creating S3 bucket %s in %s", bucketName, regionName)
	_, err = nativeClient.CreateBucket(createBucketReq)
	if err != nil {
		t.Skipf("Failed to create test S3 bucket, so skipping")
	}

	// Ensure we can perform a PUT request with the encryption header
	err = s3Client.Put(testData)
	if err != nil {
		t.Logf("WARNING: Failed to send test data to S3 bucket. (error was %s)", err)
	}

	defer func() {
		deleteBucketReq := &s3.DeleteBucketInput{
			Bucket: &bucketName,
		}

		_, err := nativeClient.DeleteBucket(deleteBucketReq)
		if err != nil {
			t.Logf("WARNING: Failed to delete the test S3 bucket. It has been left in your AWS account and may incur storage charges. (error was %s)", err)
		}
	}()

	testClient(t, client)
}
