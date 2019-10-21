package awsbase

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsCredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

// MockAwsApiServer establishes a httptest server to simulate behaviour of a real AWS API server
func MockAwsApiServer(svcName string, endpoints []*MockEndpoint) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r.Body); err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Error reading from HTTP Request Body: %s", err)
			return
		}
		requestBody := buf.String()

		log.Printf("[DEBUG] Received %s API %q request to %q: %s",
			svcName, r.Method, r.RequestURI, requestBody)

		for _, e := range endpoints {
			if r.Method == e.Request.Method && r.RequestURI == e.Request.Uri && requestBody == e.Request.Body {
				log.Printf("[DEBUG] Mocked %s API responding with %d: %s",
					svcName, e.Response.StatusCode, e.Response.Body)

				w.WriteHeader(e.Response.StatusCode)
				w.Header().Set("Content-Type", e.Response.ContentType)
				w.Header().Set("X-Amzn-Requestid", "1b206dd1-f9a8-11e5-becf-051c60f11c4a")
				w.Header().Set("Date", time.Now().Format(time.RFC1123))

				fmt.Fprintln(w, e.Response.Body)
				return
			}
		}

		w.WriteHeader(400)
	}))

	return ts
}

// GetMockedAwsApiSession establishes an AWS session to a simulated AWS API server for a given service and route endpoints.
func GetMockedAwsApiSession(svcName string, endpoints []*MockEndpoint) (func(), *session.Session, error) {
	ts := MockAwsApiServer(svcName, endpoints)

	sc := awsCredentials.NewStaticCredentials("accessKey", "secretKey", "")

	sess, err := session.NewSession(&aws.Config{
		Credentials:                   sc,
		Region:                        aws.String("us-east-1"),
		Endpoint:                      aws.String(ts.URL),
		CredentialsChainVerboseErrors: aws.Bool(true),
	})

	return ts.Close, sess, err
}

// awsMetadataApiMock establishes a httptest server to mock out the internal AWS Metadata
// service. IAM Credentials are retrieved by the EC2RoleProvider, which makes
// API calls to this internal URL. By replacing the server with a test server,
// we can simulate an AWS environment
func awsMetadataApiMock(responses []*MetadataResponse) func() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Add("Server", "MockEC2")
		log.Printf("[DEBUG] Mocker server received request to %q", r.RequestURI)
		for _, e := range responses {
			if r.RequestURI == e.Uri {
				fmt.Fprintln(w, e.Body)
				return
			}
		}
		w.WriteHeader(400)
	}))

	os.Setenv("AWS_METADATA_URL", ts.URL+"/latest")
	return ts.Close
}

// MockEndpoint represents a basic request and response that can be used for creating simple httptest server routes.
type MockEndpoint struct {
	Request  *MockRequest
	Response *MockResponse
}

// MockRequest represents a basic HTTP request
type MockRequest struct {
	Method string
	Uri    string
	Body   string
}

// MockResponse represents a basic HTTP response.
type MockResponse struct {
	StatusCode  int
	Body        string
	ContentType string
}

// MetadataResponse represents a metadata server response URI and body
type MetadataResponse struct {
	Uri  string `json:"uri"`
	Body string `json:"body"`
}

var ec2metadata_instanceIdEndpoint = &MetadataResponse{
	Uri:  "/latest/meta-data/instance-id",
	Body: "mock-instance-id",
}

var ec2metadata_securityCredentialsEndpoints = []*MetadataResponse{
	{
		Uri:  "/latest/meta-data/iam/security-credentials/",
		Body: "test_role",
	},
	{
		Uri:  "/latest/meta-data/iam/security-credentials/test_role",
		Body: "{\"Code\":\"Success\",\"LastUpdated\":\"2015-12-11T17:17:25Z\",\"Type\":\"AWS-HMAC\",\"AccessKeyId\":\"somekey\",\"SecretAccessKey\":\"somesecret\",\"Token\":\"sometoken\"}",
	},
}

var ec2metadata_iamInfoEndpoint = &MetadataResponse{
	Uri:  "/latest/meta-data/iam/info",
	Body: "{\"Code\": \"Success\",\"LastUpdated\": \"2016-03-17T12:27:32Z\",\"InstanceProfileArn\": \"arn:aws:iam::000000000000:instance-profile/my-instance-profile\",\"InstanceProfileId\": \"AIPAABCDEFGHIJKLMN123\"}",
}

const ec2metadata_iamInfoEndpoint_expectedAccountID = `000000000000`
const ec2metadata_iamInfoEndpoint_expectedPartition = `aws`

const iamResponse_GetUser_valid = `<GetUserResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <GetUserResult>
    <User>
      <UserId>AIDACKCEVSQ6C2EXAMPLE</UserId>
      <Path>/division_abc/subdivision_xyz/</Path>
      <UserName>Bob</UserName>
      <Arn>arn:aws:iam::111111111111:user/division_abc/subdivision_xyz/Bob</Arn>
      <CreateDate>2013-10-02T17:01:44Z</CreateDate>
      <PasswordLastUsed>2014-10-10T14:37:51Z</PasswordLastUsed>
    </User>
  </GetUserResult>
  <ResponseMetadata>
    <RequestId>7a62c49f-347e-4fc4-9331-6e8eEXAMPLE</RequestId>
  </ResponseMetadata>
</GetUserResponse>`

const iamResponse_GetUser_valid_expectedAccountID = `111111111111`
const iamResponse_GetUser_valid_expectedPartition = `aws`

const iamResponse_GetUser_unauthorized = `<ErrorResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <Error>
    <Type>Sender</Type>
    <Code>AccessDenied</Code>
    <Message>User: arn:aws:iam::123456789012:user/Bob is not authorized to perform: iam:GetUser on resource: arn:aws:iam::123456789012:user/Bob</Message>
  </Error>
  <RequestId>7a62c49f-347e-4fc4-9331-6e8eEXAMPLE</RequestId>
</ErrorResponse>`

const stsResponse_GetCallerIdentity_valid = `<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <GetCallerIdentityResult>
   <Arn>arn:aws:iam::222222222222:user/Alice</Arn>
    <UserId>AKIAI44QH8DHBEXAMPLE</UserId>
    <Account>222222222222</Account>
  </GetCallerIdentityResult>
  <ResponseMetadata>
    <RequestId>01234567-89ab-cdef-0123-456789abcdef</RequestId>
  </ResponseMetadata>
</GetCallerIdentityResponse>`

const stsResponse_GetCallerIdentity_valid_expectedAccountID = `222222222222`
const stsResponse_GetCallerIdentity_valid_expectedPartition = `aws`

const stsResponse_GetCallerIdentity_unauthorized = `<ErrorResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <Error>
    <Type>Sender</Type>
    <Code>AccessDenied</Code>
    <Message>User: arn:aws:iam::123456789012:user/Bob is not authorized to perform: sts:GetCallerIdentity</Message>
  </Error>
  <RequestId>01234567-89ab-cdef-0123-456789abcdef</RequestId>
</ErrorResponse>`

const iamResponse_GetUser_federatedFailure = `<ErrorResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <Error>
    <Type>Sender</Type>
    <Code>ValidationError</Code>
    <Message>Must specify userName when calling with non-User credentials</Message>
  </Error>
  <RequestId>7a62c49f-347e-4fc4-9331-6e8eEXAMPLE</RequestId>
</ErrorResponse>`

const iamResponse_ListRoles_valid = `<ListRolesResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <ListRolesResult>
    <IsTruncated>true</IsTruncated>
    <Marker>AWceSSsKsazQ4IEplT9o4hURCzBs00iavlEvEXAMPLE</Marker>
    <Roles>
      <member>
        <Path>/</Path>
        <AssumeRolePolicyDocument>%7B%22Version%22%3A%222008-10-17%22%2C%22Statement%22%3A%5B%7B%22Sid%22%3A%22%22%2C%22Effect%22%3A%22Allow%22%2C%22Principal%22%3A%7B%22Service%22%3A%22ec2.amazonaws.com%22%7D%2C%22Action%22%3A%22sts%3AAssumeRole%22%7D%5D%7D</AssumeRolePolicyDocument>
        <RoleId>AROACKCEVSQ6C2EXAMPLE</RoleId>
        <RoleName>elasticbeanstalk-role</RoleName>
        <Arn>arn:aws:iam::444444444444:role/elasticbeanstalk-role</Arn>
        <CreateDate>2013-10-02T17:01:44Z</CreateDate>
      </member>
    </Roles>
  </ListRolesResult>
  <ResponseMetadata>
    <RequestId>7a62c49f-347e-4fc4-9331-6e8eEXAMPLE</RequestId>
  </ResponseMetadata>
</ListRolesResponse>`

const iamResponse_ListRoles_valid_expectedAccountID = `444444444444`
const iamResponse_ListRoles_valid_expectedPartition = `aws`

const iamResponse_ListRoles_unauthorized = `<ErrorResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <Error>
    <Type>Sender</Type>
    <Code>AccessDenied</Code>
    <Message>User: arn:aws:iam::123456789012:user/Bob is not authorized to perform: iam:ListRoles on resource: arn:aws:iam::123456789012:role/</Message>
  </Error>
  <RequestId>7a62c49f-347e-4fc4-9331-6e8eEXAMPLE</RequestId>
</ErrorResponse>`
