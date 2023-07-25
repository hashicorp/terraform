package s3

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	servicemocks "github.com/hashicorp/aws-sdk-go-base"
)

// TODO: replace with `aws-sdk-go-base/v2/servicemocks.InitSessionTestEnv`
func initSessionTestEnv() (oldEnv []string) {
	oldEnv = stashEnv()
	os.Setenv("AWS_CONFIG_FILE", "file_not_exists")
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", "file_not_exists")

	return oldEnv
}

// TODO: replace with `aws-sdk-go-base/v2/servicemocks.StashEnv`
func stashEnv() []string {
	env := os.Environ()
	os.Clearenv()
	return env
}

// TODO: replace with `aws-sdk-go-base/v2/servicemocks.PopEnv`
func popEnv(env []string) {
	os.Clearenv()

	for _, e := range env {
		p := strings.SplitN(e, "=", 2)
		k, v := p[0], ""
		if len(p) > 1 {
			v = p[1]
		}
		os.Setenv(k, v)
	}
}

// TODO: replace with `aws-sdk-go-base/v2/servicemocks.AwsMetadataApiMock`
// awsMetadataApiMock establishes a httptest server to mock out the internal AWS Metadata
// service. IAM Credentials are retrieved by the EC2RoleProvider, which makes
// API calls to this internal URL. By replacing the server with a test server,
// we can simulate an AWS environment
func awsMetadataApiMock(responses []*servicemocks.MetadataResponse) func() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Add("Server", "MockEC2")
		log.Printf("[DEBUG] Mock EC2 metadata server received request: %s", r.RequestURI)
		for _, e := range responses {
			if r.RequestURI == e.Uri {
				fmt.Fprintln(w, e.Body)
				return
			}
		}
		w.WriteHeader(http.StatusBadRequest)
	}))

	os.Setenv("AWS_METADATA_URL", ts.URL+"/latest")
	return ts.Close
}

// TODO: replace with `aws-sdk-go-base/v2/servicemocks.Ec2metadata_securityCredentialsEndpoints`
var ec2metadata_securityCredentialsEndpoints = []*servicemocks.MetadataResponse{
	{
		Uri:  "/latest/api/token",
		Body: "Ec2MetadataApiToken",
	},
	{
		Uri:  "/latest/meta-data/iam/security-credentials/",
		Body: "test_role",
	},
	{
		Uri:  "/latest/meta-data/iam/security-credentials/test_role",
		Body: "{\"Code\":\"Success\",\"LastUpdated\":\"2015-12-11T17:17:25Z\",\"Type\":\"AWS-HMAC\",\"AccessKeyId\":\"Ec2MetadataAccessKey\",\"SecretAccessKey\":\"Ec2MetadataSecretKey\",\"Token\":\"Ec2MetadataSessionToken\"}",
	},
}

// TODO: replace with `aws-sdk-go-base/v2/servicemocks.Ec2metadata_iamInfoEndpoint`
var ec2metadata_instanceIdEndpoint = &servicemocks.MetadataResponse{
	Uri:  "/latest/meta-data/instance-id",
	Body: "mock-instance-id",
}

var ec2metadata_iamInfoEndpoint = &servicemocks.MetadataResponse{
	Uri:  "/latest/meta-data/iam/info",
	Body: "{\"Code\": \"Success\",\"LastUpdated\": \"2016-03-17T12:27:32Z\",\"InstanceProfileArn\": \"arn:aws:iam::000000000000:instance-profile/my-instance-profile\",\"InstanceProfileId\": \"AIPAABCDEFGHIJKLMN123\"}",
}

// TODO: replace with `aws-sdk-go-base/v2/servicemocks.EcsCredentialsApiMock`
func ecsCredentialsApiMock() func() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("Server", "MockECS")
		log.Printf("[DEBUG] Mock ECS credentials server received request: %s", r.RequestURI)
		if r.RequestURI == "/creds" {
			_ = json.NewEncoder(w).Encode(map[string]string{
				"AccessKeyId":     servicemocks.MockEcsCredentialsAccessKey,
				"Expiration":      time.Now().UTC().Format(time.RFC3339),
				"RoleArn":         "arn:aws:iam::000000000000:role/EcsCredentials",
				"SecretAccessKey": servicemocks.MockEcsCredentialsSecretKey,
				"Token":           servicemocks.MockEcsCredentialsSessionToken,
			})
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))

	os.Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", ts.URL+"/creds")
	return ts.Close
}

// TODO: replace with `aws-sdk-go-base/v2/servicemocks.Ec2metadata_instanceIdentityEndpoint`
func ec2metadata_instanceIdentityEndpoint(region string) *servicemocks.MetadataResponse {
	return &servicemocks.MetadataResponse{
		Uri: "/latest/dynamic/instance-identity/document",
		Body: fmt.Sprintf(`{
	"version": "2017-09-30",
	"instanceId": "mock-instance-id",
	"region": %q
}`, region),
	}
}
