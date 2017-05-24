package google

import (
	"testing"

	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"golang.org/x/oauth2/google"
)

const fakeCredentials = `{
  "type": "service_account",
  "project_id": "gcp-project",
  "private_key_id": "29a54056cee3d6886d9e8515a959af538ab5add9",
  "private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAsGHDAdHZfi81LgVeeMHXYLgNDpcFYhoBykYtTDdNyA5AixID\n8JdKlCmZ6qLNnZrbs4JlBJfmzw6rjUC5bVBFg5NwYVBu3+3Msa4rgLsTGsjPH9rt\nC+QFnFhcmzg3zz8eeXBqJdhw7wmn1Xa9SsC3h6YWveBk98ecyE7yGe8J8xGphjk7\nEQ/KBmRK/EJD0ZwuYW1W4Bv5f5fca7qvi9rCprEmL8//uy0qCwoJj2jU3zc5p72M\npkSZb1XlYxxTEo/h9WCEvWS9pGhy6fJ0sA2RsBHqU4Y5O7MJEei9yu5fVSZUi05f\n/ggfUID+cFEq0Z/A98whKPEBBJ/STdEaqEEkBwIDAQABAoIBAED6EsvF0dihbXbh\ntXbI+h4AT5cTXYFRUV2B0sgkC3xqe65/2YG1Sl0gojoE9bhcxxjvLWWuy/F1Vw93\nS5gQnTsmgpzm86F8yg6euhn3UMdqOJtknDToMITzLFJmOHEZsJFOL1x3ysrUhMan\nsn4qVrIbJn+WfbumBoToSFnzbHflacOh06ZRbYa2bpSPMfGGFtwqQjRadn5+pync\nlCjaupcg209sM0qEk/BDSzHvWL1VgLMdiKBx574TSwS0o569+7vPNt92Ydi7kARo\nreOzkkF4L3xNhKZnmls2eGH6A8cp1KZXoMLFuO+IwvBMA0O29LsUlKJU4PjBrf+7\nwaslnMECgYEA5bJv0L6DKZQD3RCBLue4/mDg0GHZqAhJBS6IcaXeaWeH6PgGZggV\nMGkWnULltJIYFwtaueTfjWqciAeocKx+rqoRjuDMOGgcrEf6Y+b5AqF+IjQM66Ll\nIYPUt3FCIc69z5LNEtyP4DSWsFPJ5UhAoG4QRlDTqT5q0gKHFjeLdeECgYEAxJRk\nkrsWmdmUs5NH9pyhTdEDIc59EuJ8iOqOLzU8xUw6/s2GSClopEFJeeEoIWhLuPY3\nX3bFt4ppl/ksLh05thRs4wXRxqhnokjD3IcGu3l6Gb5QZTYwb0VfN+q2tWVEE8Qc\nPQURheUsM2aP/gpJVQvNsWVmkT0Ijc3J8bR2hucCgYEAjOF4e0ueHu5NwFTTJvWx\nHTRGLwkU+l66ipcT0MCvPW7miRk2s3XZqSuLV0Ekqi/A3sF0D/g0tQPipfwsb48c\n0/wzcLKoDyCsFW7AQG315IswVcIe+peaeYfl++1XZmzrNlkPtrXY+ObIVbXOavZ5\nzOw0xyvj5jYGRnCOci33N4ECgYA91EKx2ABq0YGw3aEj0u31MMlgZ7b1KqFq2wNv\nm7oKgEiJ/hC/P673AsXefNAHeetfOKn/77aOXQ2LTEb2FiEhwNjiquDpL+ywoVxh\nT2LxsmqSEEbvHpUrWlFxn/Rpp3k7ElKjaqWxTHyTii2+BHQ+OKEwq6kQA3deSpy6\n1jz1fwKBgQDLqbdq5FA63PWqApfNVykXukg9MASIcg/0fjADFaHTPDvJjhFutxRP\nppI5Q95P12CQ/eRBZKJnRlkhkL8tfPaWPzzOpCTjID7avRhx2oLmstmYuXx0HluE\ncqXLbAV9WDpIJ3Bpa/S8tWujWhLDmixn2JeAdurWS+naH9U9e4I6Rw==\n-----END RSA PRIVATE KEY-----\n",
  "client_email": "user@gcp-project.iam.gserviceaccount.com",
  "client_id": "103198861025845558729",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://accounts.google.com/o/oauth2/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/user%40gcp-project.iam.gserviceaccount.com"
}`

// The following values are derived from the output of the `gsutil signurl` command.
// i.e.
// gsutil signurl fake_creds.json gs://tf-test-bucket-6159205297736845881/path/to/file
// URL	                                                HTTP Method     Expiration           Signed URL
// gs://tf-test-bucket-6159205297736845881/path/to/file	GET             2016-08-12 14:03:30  https://storage.googleapis.com/tf-test-bucket-6159205297736845881/path/to/file?GoogleAccessId=user@gcp-project.iam.gserviceaccount.com&Expires=1470967410&Signature=JJvE2Jc%2BeoagyS1qRACKBGUkgLkKjw7cGymHhtB4IzzN3nbXDqr0acRWGy0%2BEpZ3HYNDalEYsK0lR9Q0WCgty5I0JKmPIuo9hOYa1xTNH%2B22xiWsekxGV%2FcA9FXgWpi%2BFt7fBmMk4dhDe%2BuuYc7N79hd0FYuSBNW1Wp32Bluoe4SNkNAB%2BuIDd9KqPzqs09UAbBoz2y4WxXOQnRyR8GAfb8B%2FDtv62gYjtmp%2F6%2Fyr6xj7byWKZdQt8kEftQLTQmP%2F17Efjp6p%2BXo71Q0F9IhAFiqWfp3Ij8hHDSebLcVb2ULXyHNNQpHBOhFgALrFW3I6Uc3WciLEOsBS9Ej3EGdTg%3D%3D

const testUrlPath = "/tf-test-bucket-6159205297736845881/path/to/file"
const testUrlExpires = 1470967410
const testUrlExpectedSignatureBase64Encoded = "JJvE2Jc%2BeoagyS1qRACKBGUkgLkKjw7cGymHhtB4IzzN3nbXDqr0acRWGy0%2BEpZ3HYNDalEYsK0lR9Q0WCgty5I0JKmPIuo9hOYa1xTNH%2B22xiWsekxGV%2FcA9FXgWpi%2BFt7fBmMk4dhDe%2BuuYc7N79hd0FYuSBNW1Wp32Bluoe4SNkNAB%2BuIDd9KqPzqs09UAbBoz2y4WxXOQnRyR8GAfb8B%2FDtv62gYjtmp%2F6%2Fyr6xj7byWKZdQt8kEftQLTQmP%2F17Efjp6p%2BXo71Q0F9IhAFiqWfp3Ij8hHDSebLcVb2ULXyHNNQpHBOhFgALrFW3I6Uc3WciLEOsBS9Ej3EGdTg%3D%3D"
const testUrlExpectedUrl = "https://storage.googleapis.com/tf-test-bucket-6159205297736845881/path/to/file?GoogleAccessId=user@gcp-project.iam.gserviceaccount.com&Expires=1470967410&Signature=JJvE2Jc%2BeoagyS1qRACKBGUkgLkKjw7cGymHhtB4IzzN3nbXDqr0acRWGy0%2BEpZ3HYNDalEYsK0lR9Q0WCgty5I0JKmPIuo9hOYa1xTNH%2B22xiWsekxGV%2FcA9FXgWpi%2BFt7fBmMk4dhDe%2BuuYc7N79hd0FYuSBNW1Wp32Bluoe4SNkNAB%2BuIDd9KqPzqs09UAbBoz2y4WxXOQnRyR8GAfb8B%2FDtv62gYjtmp%2F6%2Fyr6xj7byWKZdQt8kEftQLTQmP%2F17Efjp6p%2BXo71Q0F9IhAFiqWfp3Ij8hHDSebLcVb2ULXyHNNQpHBOhFgALrFW3I6Uc3WciLEOsBS9Ej3EGdTg%3D%3D"

func TestUrlData_Signing(t *testing.T) {
	urlData := &UrlData{
		HttpMethod: "GET",
		Expires:    testUrlExpires,
		Path:       testUrlPath,
	}
	// unescape and decode the expected signature
	expectedSig, err := url.QueryUnescape(testUrlExpectedSignatureBase64Encoded)
	if err != nil {
		t.Error(err)
	}
	expected, err := base64.StdEncoding.DecodeString(expectedSig)
	if err != nil {
		t.Error(err)
	}

	// load fake service account credentials
	cfg, err := google.JWTConfigFromJSON([]byte(fakeCredentials), "")
	if err != nil {
		t.Error(err)
	}

	// create url data signature
	toSign := urlData.SigningString()
	result, err := SignString(toSign, cfg)
	if err != nil {
		t.Error(err)
	}

	// compare to expected value
	if !bytes.Equal(result, expected) {
		t.Errorf("Signatures do not match:\n%x\n%x\n", expected, result)
	}

}

func TestUrlData_SignedUrl(t *testing.T) {
	// load fake service account credentials
	cfg, err := google.JWTConfigFromJSON([]byte(fakeCredentials), "")
	if err != nil {
		t.Error(err)
	}

	urlData := &UrlData{
		HttpMethod: "GET",
		Expires:    testUrlExpires,
		Path:       testUrlPath,
		JwtConfig:  cfg,
	}
	result, err := urlData.SignedUrl()
	if err != nil {
		t.Errorf("Could not generated signed url: %+v", err)
	}
	if result != testUrlExpectedUrl {
		t.Errorf("URL does not match expected value:\n%s\n%s", testUrlExpectedUrl, result)
	}
}

func TestAccStorageSignedUrl_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleSignedUrlConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccGoogleSignedUrlExists("data.google_storage_object_signed_url.blerg"),
				),
			},
		},
	})
}

func TestAccStorageSignedUrl_accTest(t *testing.T) {
	bucketName := fmt.Sprintf("tf-test-bucket-%d", acctest.RandInt())

	headers := map[string]string{
		"x-goog-test":                "foo",
		"x-goog-if-generation-match": "1",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccTestGoogleStorageObjectSignedURL(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccGoogleSignedUrlRetrieval("data.google_storage_object_signed_url.story_url", nil),
					testAccGoogleSignedUrlRetrieval("data.google_storage_object_signed_url.story_url_w_headers", headers),
					testAccGoogleSignedUrlRetrieval("data.google_storage_object_signed_url.story_url_w_content_type", nil),
					testAccGoogleSignedUrlRetrieval("data.google_storage_object_signed_url.story_url_w_md5", nil),
				),
			},
		},
	})
}

func testAccGoogleSignedUrlExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		r := s.RootModule().Resources[n]
		a := r.Primary.Attributes

		if a["signed_url"] == "" {
			return fmt.Errorf("signed_url is empty: %v", a)
		}

		return nil
	}
}

func testAccGoogleSignedUrlRetrieval(n string, headers map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		r := s.RootModule().Resources[n]
		if r == nil {
			return fmt.Errorf("Datasource not found")
		}
		a := r.Primary.Attributes

		if a["signed_url"] == "" {
			return fmt.Errorf("signed_url is empty: %v", a)
		}

		// create HTTP request
		url := a["signed_url"]
		method := a["http_method"]
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return err
		}

		// Add extension headers to request, if provided
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		// content_type is optional, add to test query if provided in datasource config
		contentType := a["content_type"]
		if contentType != "" {
			req.Header.Add("Content-Type", contentType)
		}

		// content_md5 is optional, add to test query if provided in datasource config
		contentMd5 := a["content_md5"]
		if contentMd5 != "" {
			req.Header.Add("Content-MD5", contentMd5)
		}

		// send request using signed url
		client := cleanhttp.DefaultClient()
		response, err := client.Do(req)
		if err != nil {
			return err
		}
		defer response.Body.Close()

		// check content in response, should be our test string or XML with error
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		if string(body) != "once upon a time..." {
			return fmt.Errorf("Got unexpected object contents: %s\n\tURL: %s", string(body), url)
		}

		return nil
	}
}

const testGoogleSignedUrlConfig = `
data "google_storage_object_signed_url" "blerg" {
  bucket = "friedchicken"
  path   = "path/to/file"

}
`

func testAccTestGoogleStorageObjectSignedURL(bucketName string) string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "%s"
}

resource "google_storage_bucket_object" "story" {
  name   = "path/to/file"
  bucket = "${google_storage_bucket.bucket.name}"

  content = "once upon a time..."
}

data "google_storage_object_signed_url" "story_url" {
  bucket = "${google_storage_bucket.bucket.name}"
  path   = "${google_storage_bucket_object.story.name}"

}

data "google_storage_object_signed_url" "story_url_w_headers" {
  bucket = "${google_storage_bucket.bucket.name}"
  path   = "${google_storage_bucket_object.story.name}"
  extension_headers {
  	x-goog-test = "foo"
  	x-goog-if-generation-match = 1
  }
}

data "google_storage_object_signed_url" "story_url_w_content_type" {
  bucket = "${google_storage_bucket.bucket.name}"
  path   = "${google_storage_bucket_object.story.name}"

  content_type = "text/plain"
}

data "google_storage_object_signed_url" "story_url_w_md5" {
  bucket = "${google_storage_bucket.bucket.name}"
  path   = "${google_storage_bucket_object.story.name}"

  content_md5 = "${google_storage_bucket_object.story.md5hash}"
}`, bucketName)
}
