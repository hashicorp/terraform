package google

import (
	"testing"

	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"golang.org/x/oauth2/google"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
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
	toSign := urlData.CreateSigningString()
	result, err := SignString(toSign, cfg)
	if err != nil {
		t.Error(err)
	}

	// compare to expected value
	if !bytes.Equal(result, expected) {
		t.Errorf("Signatures do not match:\n%x\n%x\n", expected, result)
	}

}

func TestUrlData_BuildUrl(t *testing.T) {
	// unescape and decode the expected signature
	encodedSig, err := url.QueryUnescape(testUrlExpectedSignatureBase64Encoded)
	if err != nil {
		t.Error(err)
	}
	sig, err := base64.StdEncoding.DecodeString(encodedSig)
	if err != nil {
		t.Error(err)
	}

	// load fake service account credentials
	cfg, err := google.JWTConfigFromJSON([]byte(fakeCredentials), "")
	if err != nil {
		t.Error(err)
	}

	urlData := &UrlData{
		HttpMethod: "GET",
		Expires:    testUrlExpires,
		Path:       testUrlPath,
		Signature:  sig,
		JwtConfig:  cfg,
	}
	result := urlData.BuildUrl()
	if result != testUrlExpectedUrl {
		t.Errorf("URL does not match expected value:\n%s\n%s", testUrlExpectedUrl, result)
	}
}

func TestDatasourceSignedUrl_basic(t *testing.T) {
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

func TestDatasourceSignedUrl_accTest(t *testing.T) {
	bucketName := fmt.Sprintf("tf-test-bucket-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccTestGoogleStorageObjectSingedUrl(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccGoogleSignedUrlRetrieval("data.google_storage_object_signed_url.story_url", nil),
				),
			},
		},
	})
}

func TestDatasourceSignedUrl_wHeaders(t *testing.T) {

	headers := map[string]string{
		"x-goog-test": "foo",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccTestGoogleStorageObjectSingedUrl_wHeader(),
				Check: resource.ComposeTestCheckFunc(
					testAccGoogleSignedUrlRetrieval("data.google_storage_object_signed_url.story_url_w_headers", headers),
				),
			},
		},
	})
}

func TestDatasourceSignedUrl_wContentType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccTestGoogleStorageObjectSingedUrl_wContentType(),
				Check: resource.ComposeTestCheckFunc(
					testAccGoogleSignedUrlRetrieval("data.google_storage_object_signed_url.story_url_w_content_type", nil),
				),
			},
		},
	})
}

func TestDatasourceSignedUrl_wMD5(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccTestGoogleStorageObjectSingedUrl_wMD5(),
				Check: resource.ComposeTestCheckFunc(
					testAccGoogleSignedUrlRetrieval("data.google_storage_object_signed_url.story_url_w_md5", nil),
				),
			},
		},
	})
}

// formatRequest generates ascii representation of a request
func formatRequest(r *http.Request) string {
	// Create return string
	var request []string
	request = append(request, "--------")
	// Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	// Loop through headers
	for name, headers := range r.Header {
		//name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}
	request = append(request, "--------")
	// Return the request as a string
	return strings.Join(request, "\n")
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

		url := a["signed_url"]
		fmt.Printf("URL: %s\n", url)
		method := a["http_method"]

		req, _ := http.NewRequest(method, url, nil)

		// Apply custom headers to request
		for k, v := range headers {
			fmt.Printf("Adding Header (%s: %s)\n", k, v)
			req.Header.Set(k, v)
		}

		contentType := a["content_type"]
		if contentType != "" {
			fmt.Printf("Adding Content-Type: %s\n", contentType)
			req.Header.Add("Content-Type", contentType)
		}

		md5Digest := a["md5_digest"]
		if md5Digest != "" {
			fmt.Printf("Adding Content-MD5: %s\n", md5Digest)
			req.Header.Add("Content-MD5", md5Digest)
		}

		// send request to GET object using signed url
		client := cleanhttp.DefaultClient()

		// Print request
		//dump, _ := httputil.DumpRequest(req, true)
		//fmt.Printf("%+q\n", strings.Replace(string(dump), "\\n", "\n", 99))
		fmt.Printf("%s\n", formatRequest(req))

		response, err := client.Do(req)
		if err != nil {
			return err
		}
		defer response.Body.Close()
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

func testAccTestGoogleStorageObjectSingedUrl(bucketName string) string {
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

}`, bucketName)
}

func testAccTestGoogleStorageObjectSingedUrl_wHeader() string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "tf-signurltest-%s"
}

resource "google_storage_bucket_object" "story" {
  name   = "path/to/file"
  bucket = "${google_storage_bucket.bucket.name}"

  content = "once upon a time..."
}

data "google_storage_object_signed_url" "story_url_w_headers" {
  bucket = "${google_storage_bucket.bucket.name}"
  path   = "${google_storage_bucket_object.story.name}"
  http_headers {
  	x-goog-test = "foo"
  }
}`, acctest.RandString(6))
}

func testAccTestGoogleStorageObjectSingedUrl_wContentType() string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "tf-signurltest-%s"
}

resource "google_storage_bucket_object" "story" {
  name   = "path/to/file"
  bucket = "${google_storage_bucket.bucket.name}"

  content = "once upon a time..."
}

data "google_storage_object_signed_url" "story_url_w_content_type" {
  bucket = "${google_storage_bucket.bucket.name}"
  path   = "${google_storage_bucket_object.story.name}"

  content_type = "text/plain"
}`, acctest.RandString(6))
}

func testAccTestGoogleStorageObjectSingedUrl_wMD5() string {
	return fmt.Sprintf(`
resource "google_storage_bucket" "bucket" {
	name = "tf-signurltest-%s"
}

resource "google_storage_bucket_object" "story" {
  name   = "path/to/file"
  bucket = "${google_storage_bucket.bucket.name}"

  content = "once upon a time..."
}

data "google_storage_object_signed_url" "story_url_w_md5" {
  bucket = "${google_storage_bucket.bucket.name}"
  path   = "${google_storage_bucket_object.story.name}"

  md5_digest = "${google_storage_bucket_object.story.md5hash}"
}`, acctest.RandString(6))
}
