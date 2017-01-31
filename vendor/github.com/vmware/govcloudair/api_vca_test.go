/*
 * @Author: frapposelli
 * @Project: govcloudair
 * @Filename: api_test.go
 * @Last Modified by: frapposelli
 */

package govcloudair

import (
	"net/url"
	"os"
	"testing"

	"github.com/vmware/govcloudair/testutil"
	. "gopkg.in/check.v1"
)

var au, _ = url.ParseRequestURI("http://localhost:4444/api")
var aus, _ = url.ParseRequestURI("http://localhost:4444/api/vchs/services")
var auc, _ = url.ParseRequestURI("http://localhost:4444/api/vchs/compute/00000000-0000-0000-0000-000000000000")
var aucs, _ = url.ParseRequestURI("http://localhost:4444/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000/vcloudsession")

func Test(t *testing.T) { TestingT(t) }

type S struct {
	client *VAClient
	vdc    Vdc
	vapp   *VApp
}

var _ = Suite(&S{})

var testServer = testutil.NewHTTPServer()

var authheader = map[string]string{"x-vchs-authorization": "012345678901234567890123456789"}

func (s *S) SetUpSuite(c *C) {
	testServer.Start()
	var err error

	os.Setenv("VCLOUDAIR_ENDPOINT", "http://localhost:4444/api")

	s.client, err = NewVAClient()
	if err != nil {
		panic(err)
	}

	testServer.ResponseMap(5, testutil.ResponseMap{
		"/api/vchs/sessions":                                                                                            testutil.Response{201, authheader, vaauthorization},
		"/api/vchs/services":                                                                                            testutil.Response{200, nil, vaservices},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000":                                                        testutil.Response{200, nil, vacompute},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000/vcloudsession": testutil.Response{201, nil, vabackend},
		"/api/vdc/00000000-0000-0000-0000-000000000000":                                                                 testutil.Response{200, nil, vdcExample},
	})

	s.vdc, err = s.client.Authenticate("username", "password", "CI123456-789", "VDC12345-6789")
	s.vapp = NewVApp(&s.client.Client)
	_ = testServer.WaitRequests(5)
	testServer.Flush()
	if err != nil {
		panic(err)
	}
}

func (s *S) TearDownTest(c *C) {
	testServer.Flush()
}

func TestClient_vaauthorize(t *testing.T) {
	testServer.Start()
	var err error

	// Set up a working client
	os.Setenv("VCLOUDAIR_ENDPOINT", "http://localhost:4444/api")

	client, err := NewVAClient()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Set up a correct conversation
	testServer.Response(201, authheader, vaauthorization)
	_, err = client.vaauthorize("username", "password")
	_ = testServer.WaitRequest()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Test if token is correctly set on client.
	if client.VAToken != "012345678901234567890123456789" {
		t.Fatalf("VAtoken not set on client: %s", client.VAToken)
	}

	// Test client errors

	// Test a correct response with a wrong status code
	testServer.Response(404, authheader, notfoundErr)
	_, err = client.vaauthorize("username", "password")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}

	// Test an API error
	testServer.Response(500, authheader, vcdError)
	_, err = client.vaauthorize("username", "password")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}

	// Test an API response that doesn't contain the param we're looking for.
	testServer.Response(200, authheader, vaauthorizationErr)
	_, err = client.vaauthorize("username", "password")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}

	// Test an un-parsable response.
	testServer.Response(200, authheader, notfoundErr)
	_, err = client.vaauthorize("username", "password")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}

}

func TestClient_vaacquireservice(t *testing.T) {
	testServer.Start()
	var err error

	// Set up a working client
	os.Setenv("VCLOUDAIR_ENDPOINT", "http://localhost:4444/api")
	client, err := NewVAClient()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	client.VAToken = "012345678901234567890123456789"

	// Test a correct conversation
	testServer.Response(200, nil, vaservices)
	vacomputehref, err := client.vaacquireservice(*aus, "CI123456-789")
	_ = testServer.WaitRequest()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if vacomputehref.String() != "http://localhost:4444/api/vchs/compute/00000000-0000-0000-0000-000000000000" {
		t.Fatalf("VAComputeHREF not set on client: %s", vacomputehref)
	}

	if client.Region != "US - Anywhere" {
		t.Fatalf("Region not set on client: %s", client.Region)
	}

	// Test client errors

	// Test a 404
	testServer.Response(404, nil, notfoundErr)
	_, err = client.vaacquireservice(*aus, "CI123456-789")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}

	// Test an API error
	testServer.Response(500, nil, vcdError)
	_, err = client.vaacquireservice(*aus, "CI123456-789")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}

	// Test an unknown Compute ID
	testServer.Response(200, nil, vaservices)
	_, err = client.vaacquireservice(*aus, "NOTVALID-789")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}

	// Test an un-parsable response
	testServer.Response(200, nil, notfoundErr)
	_, err = client.vaacquireservice(*aus, "CI123456-789")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}

}

func TestClient_vaacquirecompute(t *testing.T) {
	testServer.Start()
	var err error

	os.Setenv("VCLOUDAIR_ENDPOINT", "http://localhost:4444/api")
	client, err := NewVAClient()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	client.VAToken = "012345678901234567890123456789"
	client.Region = "US - Anywhere"

	testServer.Response(200, nil, vacompute)
	vavdchref, err := client.vaacquirecompute(*auc, "VDC12345-6789")
	_ = testServer.WaitRequest()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if vavdchref.String() != "http://localhost:4444/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000/vcloudsession" {
		t.Fatalf("VAVDCHREF not set on client: %s", vavdchref)
	}

	// Test client errors

	// Test a 404
	testServer.Response(404, nil, notfoundErr)
	_, err = client.vaacquirecompute(*auc, "VDC12345-6789")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}
	// Test an API error
	testServer.Response(500, nil, vcdError)
	_, err = client.vaacquirecompute(*auc, "VDC12345-6789")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}

	// Test an unknown VDC ID
	testServer.Response(200, nil, vacompute)
	_, err = client.vaacquirecompute(*auc, "INVALID-6789")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}

	// Test an un-parsable response
	testServer.Response(200, nil, notfoundErr)
	_, err = client.vaacquirecompute(*auc, "VDC12345-6789")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}

}

func TestClient_vagetbackendauth(t *testing.T) {
	testServer.Start()
	var err error

	os.Setenv("VCLOUDAIR_ENDPOINT", "http://localhost:4444/api")
	client, err := NewVAClient()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	client.VAToken = "012345678901234567890123456789"
	client.Region = "US - Anywhere"

	testServer.Response(201, nil, vabackend)
	err = client.vagetbackendauth(*aucs, "CI123456-789")
	_ = testServer.WaitRequest()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if client.Client.VCDToken != "01234567890123456789012345678901" {
		t.Fatalf("VCDToken not set on client: %s", client.Client.VCDToken)
	}
	if client.Client.VCDAuthHeader != "x-vcloud-authorization" {
		t.Fatalf("VCDAuthHeader not set on client: %s", client.Client.VCDAuthHeader)
	}
	if client.Client.VCDVDCHREF.String() != "http://localhost:4444/api/vdc/00000000-0000-0000-0000-000000000000" {
		t.Fatalf("VDC not set on client: %s", client.Client.VCDVDCHREF)
	}

	// Test client errors

	// Test a 404
	testServer.Response(404, nil, notfoundErr)
	err = client.vagetbackendauth(*aucs, "VDC12345-6789")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}
	// Test an API error
	testServer.Response(500, nil, vcdError)
	err = client.vagetbackendauth(*aucs, "VDC12345-6789")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}
	// Test an unknown backend VDC IC
	testServer.Response(201, nil, vabackend)
	err = client.vagetbackendauth(*aucs, "INVALID-6789")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}
	// Test an un-parsable response
	testServer.Response(201, nil, notfoundErr)
	err = client.vagetbackendauth(*aucs, "VDC12345-6789")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}
	// Test a botched backend VDC IC
	testServer.Response(201, nil, vabackendErr)
	err = client.vagetbackendauth(*aucs, "VDC12345-6789")
	_ = testServer.WaitRequest()
	if err == nil {
		t.Fatalf("Request error not caught: %v", err)
	}

}

// Env variable tests

func TestClient_vaauthorize_env(t *testing.T) {

	os.Setenv("VCLOUDAIR_USERNAME", "username")
	os.Setenv("VCLOUDAIR_PASSWORD", "password")

	testServer.Start()
	var err error
	os.Setenv("VCLOUDAIR_ENDPOINT", "http://localhost:4444/api")
	client, err := NewVAClient()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	testServer.Response(201, authheader, vaauthorization)
	_, err = client.vaauthorize("", "")
	_ = testServer.WaitRequest()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if client.VAToken != "012345678901234567890123456789" {
		t.Fatalf("VAtoken not set on client: %s", client.VAToken)
	}

}

func TestClient_vaacquireservice_env(t *testing.T) {

	os.Setenv("VCLOUDAIR_COMPUTEID", "CI123456-789")

	testServer.Start()
	var err error
	os.Setenv("VCLOUDAIR_ENDPOINT", "http://localhost:4444/api")
	client, err := NewVAClient()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	client.VAToken = "012345678901234567890123456789"

	testServer.Response(200, nil, vaservices)
	vacomputehref, err := client.vaacquireservice(*aus, "")
	_ = testServer.WaitRequest()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if vacomputehref.String() != "http://localhost:4444/api/vchs/compute/00000000-0000-0000-0000-000000000000" {
		t.Fatalf("VAComputeHREF not set on client: %s", vacomputehref)
	}

	if client.Region != "US - Anywhere" {
		t.Fatalf("Region not set on client: %s", client.Region)
	}

}

func TestClient_vaacquirecompute_env(t *testing.T) {

	os.Setenv("VCLOUDAIR_VDCID", "VDC12345-6789")

	testServer.Start()
	var err error
	os.Setenv("VCLOUDAIR_ENDPOINT", "http://localhost:4444/api")
	client, err := NewVAClient()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	client.VAToken = "012345678901234567890123456789"
	client.Region = "US - Anywhere"

	testServer.Response(200, nil, vacompute)
	vavdchref, err := client.vaacquirecompute(*auc, "")
	_ = testServer.WaitRequest()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if vavdchref.String() != "http://localhost:4444/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000/vcloudsession" {
		t.Fatalf("VAVDCHREF not set on client: %s", vavdchref)
	}

}

func TestClient_vagetbackendauth_env(t *testing.T) {

	os.Setenv("VCLOUDAIR_VDCID", "VDC12345-6789")

	testServer.Start()
	var err error
	os.Setenv("VCLOUDAIR_ENDPOINT", "http://localhost:4444/api")
	client, err := NewVAClient()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	client.VAToken = "012345678901234567890123456789"
	client.Region = "US - Anywhere"

	testServer.Response(201, nil, vabackend)
	err = client.vagetbackendauth(*aucs, "")
	_ = testServer.WaitRequest()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if client.Client.VCDToken != "01234567890123456789012345678901" {
		t.Fatalf("VCDToken not set on client: %s", client.Client.VCDToken)
	}
	if client.Client.VCDAuthHeader != "x-vcloud-authorization" {
		t.Fatalf("VCDAuthHeader not set on client: %s", client.Client.VCDAuthHeader)
	}
	if client.Client.VCDVDCHREF.String() != "http://localhost:4444/api/vdc/00000000-0000-0000-0000-000000000000" {
		t.Fatalf("VDC not set on client: %s", client.Client.VCDVDCHREF)
	}

}

func TestClient_NewClient(t *testing.T) {

	var err error
	os.Setenv("VCLOUDAIR_ENDPOINT", "")
	if _, err = NewVAClient(); err != nil {
		t.Fatalf("err: %v", err)
	}

	os.Setenv("VCLOUDAIR_ENDPOINT", "💩")
	if _, err = NewVAClient(); err == nil {
		t.Fatalf("err: %v", err)
	}

}

func TestClient_Disconnect(t *testing.T) {

	testServer.Start()
	var err error

	// Test an authenticated client
	c := makeClient(t)
	testServer.Response(201, nil, "")
	err = c.Disconnect()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Test an empty client
	client, err := NewVAClient()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	err = client.Disconnect()
	if err == nil {
		t.Fatalf("err: %v", err)
	}

}

func TestVAClient_Authenticate(t *testing.T) {

	testServer.Start()
	var err error
	os.Setenv("VCLOUDAIR_ENDPOINT", "http://localhost:4444/api")
	client, err := NewVAClient()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Botched auth
	testServer.ResponseMap(5, testutil.ResponseMap{
		"/api/vchs/sessions":                                                                                            testutil.Response{401, nil, vcdError},
		"/api/vchs/services":                                                                                            testutil.Response{200, nil, vaservices},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000":                                                        testutil.Response{200, nil, vacompute},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000/vcloudsession": testutil.Response{201, nil, vabackend},
		"/api/vdc/00000000-0000-0000-0000-000000000000":                                                                 testutil.Response{200, nil, vdcExample},
	})

	_, err = client.Authenticate("username", "password", "CI123456-789", "VDC12345-6789")
	_ = testServer.WaitRequests(1)
	testServer.Flush()
	if err == nil {
		t.Fatalf("Uncatched error: %v", err)
	}

	// Botched services
	testServer.ResponseMap(5, testutil.ResponseMap{
		"/api/vchs/sessions":                                                                                            testutil.Response{201, authheader, vaauthorization},
		"/api/vchs/services":                                                                                            testutil.Response{500, nil, vcdError},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000":                                                        testutil.Response{200, nil, vacompute},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000/vcloudsession": testutil.Response{201, nil, vabackend},
		"/api/vdc/00000000-0000-0000-0000-000000000000":                                                                 testutil.Response{200, nil, vdcExample},
	})

	_, err = client.Authenticate("username", "password", "CI123456-789", "VDC12345-6789")
	_ = testServer.WaitRequests(2)
	testServer.Flush()
	if err == nil {
		t.Fatalf("Uncatched error: %v", err)
	}

	// Botched compute
	testServer.ResponseMap(5, testutil.ResponseMap{
		"/api/vchs/sessions":                                                                                            testutil.Response{201, authheader, vaauthorization},
		"/api/vchs/services":                                                                                            testutil.Response{200, nil, vaservices},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000":                                                        testutil.Response{500, nil, vcdError},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000/vcloudsession": testutil.Response{201, nil, vabackend},
		"/api/vdc/00000000-0000-0000-0000-000000000000":                                                                 testutil.Response{200, nil, vdcExample},
	})

	_, err = client.Authenticate("username", "password", "CI123456-789", "VDC12345-6789")
	_ = testServer.WaitRequests(3)
	testServer.Flush()
	if err == nil {
		t.Fatalf("Uncatched error: %v", err)
	}

	// Botched backend
	testServer.ResponseMap(5, testutil.ResponseMap{
		"/api/vchs/sessions":                                                                                            testutil.Response{201, authheader, vaauthorization},
		"/api/vchs/services":                                                                                            testutil.Response{200, nil, vaservices},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000":                                                        testutil.Response{200, nil, vacompute},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000/vcloudsession": testutil.Response{500, nil, vcdError},
		"/api/vdc/00000000-0000-0000-0000-000000000000":                                                                 testutil.Response{200, nil, vdcExample},
	})

	_, err = client.Authenticate("username", "password", "CI123456-789", "VDC12345-6789")
	_ = testServer.WaitRequests(4)
	testServer.Flush()
	if err == nil {
		t.Fatalf("Uncatched error: %v", err)
	}

	// Botched vdc
	testServer.ResponseMap(5, testutil.ResponseMap{
		"/api/vchs/sessions":                                                                                            testutil.Response{201, authheader, vaauthorization},
		"/api/vchs/services":                                                                                            testutil.Response{200, nil, vaservices},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000":                                                        testutil.Response{200, nil, vacompute},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000/vcloudsession": testutil.Response{201, nil, vabackend},
		"/api/vdc/00000000-0000-0000-0000-000000000000":                                                                 testutil.Response{500, nil, vcdError},
	})

	_, err = client.Authenticate("username", "password", "CI123456-789", "VDC12345-6789")
	_ = testServer.WaitRequests(5)
	testServer.Flush()
	if err == nil {
		t.Fatalf("Uncatched error: %v", err)
	}

}

func makeClient(t *testing.T) VAClient {

	testServer.Start()
	var err error
	os.Setenv("VCLOUDAIR_ENDPOINT", "http://localhost:4444/api")
	client, err := NewVAClient()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	testServer.ResponseMap(5, testutil.ResponseMap{
		"/api/vchs/sessions":                                                                                            testutil.Response{201, authheader, vaauthorization},
		"/api/vchs/services":                                                                                            testutil.Response{200, nil, vaservices},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000":                                                        testutil.Response{200, nil, vacompute},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000/vcloudsession": testutil.Response{201, nil, vabackend},
		"/api/vdc/00000000-0000-0000-0000-000000000000":                                                                 testutil.Response{200, nil, vdcExample},
	})

	_, err = client.Authenticate("username", "password", "CI123456-789", "VDC12345-6789")

	_ = testServer.WaitRequests(5)
	testServer.Flush()

	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if client.VAToken != "012345678901234567890123456789" {
		t.Fatalf("VAtoken not set on client: %s", client.VAToken)
	}

	if client.Region != "US - Anywhere" {
		t.Fatalf("Region not set on client: %s", client.Region)
	}

	if client.Client.VCDToken != "01234567890123456789012345678901" {
		t.Fatalf("VCDToken not set on client: %s", client.Client.VCDToken)
	}
	if client.Client.VCDAuthHeader != "x-vcloud-authorization" {
		t.Fatalf("VCDAuthHeader not set on client: %s", client.Client.VCDAuthHeader)
	}
	if client.Client.VCDVDCHREF.String() != "http://localhost:4444/api/vdc/00000000-0000-0000-0000-000000000000" {
		t.Fatalf("VDC not set on client: %s", client.Client.VCDVDCHREF)
	}

	return *client
}

func TestClient_parseErr(t *testing.T) {
	testServer.Start()
	var err error
	os.Setenv("VCLOUDAIR_ENDPOINT", "http://localhost:4444/api")
	client, err := NewVAClient()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// I'M A TEAPOT!
	testServer.ResponseMap(5, testutil.ResponseMap{
		"/api/vchs/sessions":                                                                                            testutil.Response{201, authheader, vaauthorization},
		"/api/vchs/services":                                                                                            testutil.Response{200, nil, vaservices},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000":                                                        testutil.Response{200, nil, vacompute},
		"/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000/vcloudsession": testutil.Response{418, nil, notfoundErr},
		"/api/vdc/00000000-0000-0000-0000-000000000000":                                                                 testutil.Response{200, nil, vdcExample},
	})

	_, err = client.Authenticate("username", "password", "CI123456-789", "VDC12345-6789")

	_ = testServer.WaitRequests(4)
	testServer.Flush()

	if err == nil {
		t.Fatalf("Uncatched error: %v", err)
	}

}

func TestClient_NewRequest(t *testing.T) {
	c := makeClient(t)

	params := map[string]string{
		"foo": "bar",
		"baz": "bar",
	}

	uri, _ := url.ParseRequestURI("http://localhost:4444/api/bar")

	req := c.Client.NewRequest(params, "POST", *uri, nil)

	encoded := req.URL.Query()
	if encoded.Get("foo") != "bar" {
		t.Fatalf("bad: %v", encoded)
	}

	if encoded.Get("baz") != "bar" {
		t.Fatalf("bad: %v", encoded)
	}

	if req.URL.String() != "http://localhost:4444/api/bar?baz=bar&foo=bar" {
		t.Fatalf("bad base url: %v", req.URL.String())
	}

	if req.Header.Get("x-vcloud-authorization") != "01234567890123456789012345678901" {
		t.Fatalf("bad auth header: %v", req.Header)
	}

	if req.Method != "POST" {
		t.Fatalf("bad method: %v", req.Method)
	}

}

// status: 404
var notfoundErr = `
	<html>
		<head><title>404 Not Found</title></head>
		<body bgcolor="white">
			<center><h1>404 Not Found</h1></center>
			<hr><center>nginx/1.4.6 (Ubuntu)</center>
		</body>
	</html>
	`

// status: 201
var vaauthorization = `
	<?xml version="1.0" ?>
	<Session href="http://localhost:4444/api/vchs/session" type="application/xml;class=vnd.vmware.vchs.session" xmlns="http://www.vmware.com/vchs/v5.6" xmlns:tns="http://www.vmware.com/vchs/v5.6" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
	    <Link href="http://localhost:4444/api/vchs/services" rel="down" type="application/xml;class=vnd.vmware.vchs.servicelist"/>
	    <Link href="http://localhost:4444/api/vchs/session" rel="remove"/>
	</Session>
	`
var vaauthorizationErr = `
	<?xml version="1.0" ?>
	<Session href="http://localhost:4444/api/vchs/session" type="application/xml;class=vnd.vmware.vchs.session" xmlns="http://www.vmware.com/vchs/v5.6" xmlns:tns="http://www.vmware.com/vchs/v5.6" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
	    <Link href="http://localhost:4444/api/vchs/session" rel="remove"/>
	</Session>
	`

// status: 200
var vaservices = `
	<?xml version="1.0" ?>
	<Services href="http://localhost:4444/api/vchs/services" type="application/xml;class=vnd.vmware.vchs.servicelist" xmlns="http://www.vmware.com/vchs/v5.6" xmlns:tns="http://www.vmware.com/vchs/v5.6" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
	    <Service href="http://localhost:4444/api/vchs/compute/00000000-0000-0000-0000-000000000000" region="US - Anywhere" serviceId="CI123456-789" serviceType="compute:vpc" type="application/xml;class=vnd.vmware.vchs.compute"/>
	</Services>
	`

// status: 200
var vacompute = `
	<?xml version="1.0" ?>
	<Compute href="http://localhost:4444/api/vchs/compute/00000000-0000-0000-0000-000000000000" serviceId="CI123456-789" serviceType="compute:vpc" type="application/xml;class=vnd.vmware.vchs.compute" xmlns="http://www.vmware.com/vchs/v5.6" xmlns:tns="http://www.vmware.com/vchs/v5.6" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
	    <Link href="http://localhost:4444/api/vchs/services" name="services" rel="up" type="application/xml;class=vnd.vmware.vchs.servicelist"/>
	    <VdcRef href="http://localhost:4444/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000" name="VDC12345-6789" status="Active" type="application/xml;class=vnd.vmware.vchs.vdcref">
	        <Link href="http://localhost:4444/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000/vcloudsession" name="VDC12345-6789" rel="down" type="application/xml;class=vnd.vmware.vchs.vcloudsession"/>
	    </VdcRef>
	</Compute>
	`

// status: 201
var vabackend = `
<?xml version="1.0" ?>
<VCloudSession href="http://localhost:4444/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000/vcloudsession" name="CI123456-789-session" type="application/xml;class=vnd.vmware.vchs.vcloudsession" xmlns="http://www.vmware.com/vchs/v5.6" xmlns:tns="http://www.vmware.com/vchs/v5.6" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <Link href="http://localhost:4444/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000" name="vdc" rel="up" type="application/xml;class=vnd.vmware.vchs.vdcref"/>
    <VdcLink authorizationHeader="x-vcloud-authorization" authorizationToken="01234567890123456789012345678901" href="http://localhost:4444/api/vdc/00000000-0000-0000-0000-000000000000" name="CI123456-789" rel="vcloudvdc" type="application/vnd.vmware.vcloud.vdc+xml"/>
</VCloudSession>
	`

// status: 201
var vabackendErr = `
<?xml version="1.0" ?>
<VCloudSession href="http://localhost:4444/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000/vcloudsession" name="CI123456-789-session" type="application/xml;class=vnd.vmware.vchs.vcloudsession" xmlns="http://www.vmware.com/vchs/v5.6" xmlns:tns="http://www.vmware.com/vchs/v5.6" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <Link href="http://localhost:4444/api/vchs/compute/00000000-0000-0000-0000-000000000000/vdc/00000000-0000-0000-0000-000000000000" name="vdc" rel="up" type="application/xml;class=vnd.vmware.vchs.vdcref"/>
    <VdcLink authorizationHeader="x-vcloud-authorization" authorizationToken="01234567890123456789012345678901" href="http://$£$%£%$:4444/api/vdc/00000000-0000-0000-0000-000000000000" name="CI123456-789" rel="vcloudvdc" type="application/vnd.vmware.vcloud.vdc+xml"/>
</VCloudSession>
	`

var vcdError = `
<Error xmlns="http://www.vmware.com/vcloud/v1.5" message="Error Message" majorErrorCode="500" minorErrorCode="Server Error" vendorSpecificErrorCode="NoSpecificError" stackTrace="Hello my name is Stack Trace"/>
	`
