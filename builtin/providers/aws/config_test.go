package aws

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsCredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

// getMockedAwsApiSession establishes a httptest server to simulate behaviour
// of a real AWS API server
func getMockedAwsApiSession(svcName string, endpoints []*awsMockEndpoint) (func(), *session.Session, error) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
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
		return
	}))

	sc := awsCredentials.NewStaticCredentials("accessKey", "secretKey", "")

	sess, err := session.NewSession(&aws.Config{
		Credentials:                   sc,
		Region:                        aws.String("us-east-1"),
		Endpoint:                      aws.String(ts.URL),
		CredentialsChainVerboseErrors: aws.Bool(true),
	})

	return ts.Close, sess, err
}

type awsMockEndpoint struct {
	Request  *awsMockRequest
	Response *awsMockResponse
}

type awsMockRequest struct {
	Method string
	Uri    string
	Body   string
}

type awsMockResponse struct {
	StatusCode  int
	Body        string
	ContentType string
}
