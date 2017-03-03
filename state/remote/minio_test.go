package remote

import (
	"fmt"
	"testing"
	"time"
)

func TestMinioClient_impl(t *testing.T) {
	var _ Client = new(MinioClient)
}

func TestMinioFactory(t *testing.T) {
	// This test just instantiates the client. Shouldn't make any actual
	// requests nor incur any costs.

	config := make(map[string]string)

	// Empty config is an error
	_, err := minioFactory(config)
	if err == nil {
		t.Fatalf("Empty config should be error")
	}

	config["endpoint"] = "endpoint:1234"
	config["bucket_name"] = "foo"
	config["bucket_location"] = "location"
	config["object_name"] = "bar/tfstate.tf"
	config["use_ssl"] = "false"
	config["access_key_id"] = "bazkey"
	config["secret_access_key"] = "bazsecret"

	client, err := minioFactory(config)
	if err != nil {
		t.Fatalf("Error for valid config")
	}

	minioClient := client.(*MinioClient)

	if minioClient.endpoint != "endpoint:1234" {
		t.Fatalf("Incorrect endpoint was populated")
	}
	if minioClient.bucketName != "foo" {
		t.Fatalf("Incorrect bucketName was populated")
	}
	if minioClient.bucketLocation != "location" {
		t.Fatalf("Incorrect bucketLocation was populated")
	}
	if minioClient.objectName != "bar/tfstate.tf" {
		t.Fatalf("Incorrect objectName was populated")
	}
	if minioClient.accessKeyID != "bazkey" {
		t.Fatalf("Incorrect accessKeyID was populated")
	}
	if minioClient.useSSL {
		t.Fatalf("Incorrect useSSL was populated")
	}
	if minioClient.secretAccessKey != "bazsecret" {
		t.Fatalf("Incorrect secretAccessKey was populated")
	}
}

func TestMinioClient(t *testing.T) {
	// This test creates a bucket in the public Minio server and populates it.

	bucketName := fmt.Sprintf("terraform-remote-minio-test-%x", time.Now().Unix())
	objectName := "testState/tfstate.tf"
	testData := []byte(`testing data`)

	config := make(map[string]string)
	config["bucket_name"] = bucketName
	config["object_name"] = objectName

	// Public Minio credentials for testing and development
	config["endpoint"] = "play.minio.io:9000"
	config["bucket_location"] = "us-east-1"
	config["use_ssl"] = "true"
	config["access_key_id"] = "Q3AM3UQ867SPQQA43P2F" 
	config["secret_access_key"] = "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG"

	client, err := minioFactory(config)
	if err != nil {
		t.Fatalf("Error for valid config")
	}

	minioClient := client.(*MinioClient)

	// Ensure we can perform a PUT request
	err = minioClient.Put(testData)
	if err != nil {
		t.Logf("WARNING: Failed to send test data to Minio bucket. (error was %s)", err)
	}
	
	defer func() {
		err := minioClient.client.RemoveBucket(bucketName)
		if err != nil {
		 	t.Logf("WARNING: Failed to delete the test Minio bucket. It may have been left in your account and may incur storage charges. (error was %s)", err)
		}
	}()

	testClient(t, client)
}
