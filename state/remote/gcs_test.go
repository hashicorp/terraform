package remote

import (
	"fmt"
	"os"
	"testing"
	"time"

	storage "google.golang.org/api/storage/v1"
)

func TestGCSClient_impl(t *testing.T) {
	var _ Client = new(GCSClient)
}

func TestGCSClient(t *testing.T) {
	// This test creates a bucket in GCS and populates it.
	// It may incur costs, so it will only run if GCS credential environment
	// variables are present.

	projectID := os.Getenv("GOOGLE_PROJECT")
	if projectID == "" {
		t.Skipf("skipping; GOOGLE_PROJECT must be set")
	}

	bucketName := fmt.Sprintf("terraform-remote-gcs-test-%x", time.Now().Unix())
	keyName := "testState"
	testData := []byte(`testing data`)

	config := make(map[string]string)
	config["bucket"] = bucketName
	config["path"] = keyName

	client, err := gcsFactory(config)
	if err != nil {
		t.Fatalf("Error for valid config: %v", err)
	}

	gcsClient := client.(*GCSClient)
	nativeClient := gcsClient.clientStorage

	// Be clear about what we're doing in case the user needs to clean
	// this up later.
	if _, err := nativeClient.Buckets.Get(bucketName).Do(); err == nil {
		fmt.Printf("Bucket %s already exists - skipping buckets.insert call.", bucketName)
	} else {
		// Create a bucket.
		if res, err := nativeClient.Buckets.Insert(projectID, &storage.Bucket{Name: bucketName}).Do(); err == nil {
			fmt.Printf("Created bucket %v at location %v\n\n", res.Name, res.SelfLink)
		} else {
			t.Skipf("Failed to create test GCS bucket, so skipping")
		}
	}

	// Ensure we can perform a PUT request with the encryption header
	err = gcsClient.Put(testData)
	if err != nil {
		t.Logf("WARNING: Failed to send test data to GCS bucket. (error was %s)", err)
	}

	defer func() {
		// Delete the test bucket in the project
		if err := gcsClient.clientStorage.Buckets.Delete(bucketName).Do(); err != nil {
			t.Logf("WARNING: Failed to delete the test GCS bucket. It has been left in your GCE account and may incur storage charges. (error was %s)", err)
		}
	}()

	testClient(t, client)
}
