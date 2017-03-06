package ibmcloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"strings"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/softlayer/softlayer-go/sl"
)

func TestSessionMissingBluemixRequiredParamters(t *testing.T) {
	c := &Config{}
	_, err := newSession(c)

	if err != ErrBluemixParamaterValidation {
		t.Fatalf("Expecting error: %s, instead got: %v", ErrBluemixParamaterValidation, err)
	}
}
func TestSessionCreation_success(t *testing.T) {
	c := &Config{IBMID: "id", IBMIDPassword: "pass", RetryCount: 1, RetryDelay: 1 * time.Second}
	imsToken := "token"
	imsUserid := 1
	stringifiedUserID := strconv.Itoa(imsUserid)
	slUserName := "somesluser"
	slAPIKey := "apikey"
	resetEnv := setEnv(map[string]string{
		"IBMCLOUD_IAM_TOKEN": "",
	}, t)
	defer resetEnv()
	mockedIAMRequest := mockIAMRequest(c, "")
	mockedIAMResponse := &MockedResponse{
		StatusCode:  200,
		ContentType: "application/json",
		Body: map[string]interface{}{
			"ims_token":   imsToken,
			"ims_user_id": imsUserid,
			"token_type":  "bearer",
			"expires_in":  123,
			"expiration":  5,
		},
	}

	mockedSLRequest := mockSoftLayerXMLRPCRequest(stringifiedUserID)
	mockedSLResponse := &MockedResponse{
		StatusCode:  200,
		ContentType: "text/xml",
		Body:        `<?xml version="1.0" encoding="utf-8"?><params><param><value><struct><member><name>username</name><value><string>` + slUserName + `</string></value></member><member><name>apiAuthenticationKeys</name><value><array><data><value><struct><member><name>authenticationKey</name><value><string>` + slAPIKey + `</string></value></member></struct></value></data></array></value></member></struct></value></param></params>`,
	}

	transactionsIAM := []*MockedTransaction{
		{
			Request:  mockedIAMRequest,
			Response: mockedIAMResponse,
		},
	}
	iamServer := getMockIAMServer(transactionsIAM)
	c.IAMEndpoint = iamServer.URL
	defer iamServer.Close()

	transactionsSL := []*MockedTransaction{
		{
			Request:  mockedSLRequest,
			Response: mockedSLResponse,
		},
	}
	slServer := getMockSoftlayerXMLRPCServer(transactionsSL)
	c.SoftlayerXMLRPCEndpoint = slServer.URL + "/xmlrpc/"
	defer slServer.Close()

	s, e := newSession(c)

	if e != nil {
		t.Fatalf("Session creation failed")
	}

	if s.SoftLayerAPIKey != slAPIKey {
		t.Fatalf("API Key mismatch, Expecting (%s), instead got (%s)", slAPIKey, s.SoftLayerAPIKey)
	}

	if s.SoftLayerIMSToken != imsToken {
		t.Fatalf("IMS Token mismatch, Expecting (%s), instead got (%s)", imsToken, s.SoftLayerIMSToken)
	}

	if s.SoftLayerUserName != slUserName {
		t.Fatalf("SL Username mismatch, Expecting (%s), instead got (%s)", slUserName, s.SoftLayerUserName)
	}

	if s.SoftLayerUserID != imsUserid {
		t.Fatalf("SL UserID mismatch, Expecting (%d), instead got (%d)", imsUserid, s.SoftLayerUserID)
	}
}

func TestFetchIMSToken_success_no_iam_token(t *testing.T) {
	c := &Config{IBMID: "id", IBMIDPassword: "pass", RetryCount: 1, RetryDelay: 1 * time.Second}
	mockedRequest := mockIAMRequest(c, "")
	mockedResponse := &MockedResponse{
		StatusCode:  200,
		ContentType: "application/json",
		Body: map[string]interface{}{
			"ims_token":   "token",
			"ims_user_id": 1,
			"token_type":  "bearer",
			"expires_in":  123,
			"expiration":  5,
		},
	}
	transactions := []*MockedTransaction{
		{
			Request:  mockedRequest,
			Response: mockedResponse,
		},
	}
	server := getMockIAMServer(transactions)
	c.IAMEndpoint = server.URL
	defer server.Close()

	imstoken, imsuserid, err := fetchIMSToken(c, "")
	if err != nil {
		t.Fatalf("Error fetching ims token for %s: %v", c.IBMID, err)
	}

	respBody := mockedResponse.Body.(map[string]interface{})
	expectedToken := respBody["ims_token"].(string)
	if imstoken != expectedToken {
		t.Fatalf("IMS Token mismatch, expected (%s), got (%s)", expectedToken, imstoken)
	}

	expectedIMSUserID := respBody["ims_user_id"].(int)
	if imsuserid != expectedIMSUserID {
		t.Fatalf("IMS User ID mismatch, expected (%d), got (%d)", expectedIMSUserID, imsuserid)
	}
}

func TestFetchIMSToken_success_with_iam_token(t *testing.T) {
	c := &Config{RetryCount: 1, RetryDelay: 1 * time.Second}

	iamToken := "some_access_token"
	mockedRequest := mockIAMRequest(c, iamToken)
	mockedResponse := &MockedResponse{
		StatusCode:  200,
		ContentType: "application/json",
		Body: map[string]interface{}{
			"ims_token":   "token",
			"ims_user_id": 1,
			"token_type":  "bearer",
			"expires_in":  123,
			"expiration":  5,
		},
	}
	transactions := []*MockedTransaction{
		{
			Request:  mockedRequest,
			Response: mockedResponse,
		},
	}
	server := getMockIAMServer(transactions)
	c.IAMEndpoint = server.URL
	defer server.Close()

	imstoken, imsuserid, err := fetchIMSToken(c, iamToken)
	if err != nil {
		t.Fatalf("Error fetching ims token for %s: %v", c.IBMID, err)
	}

	respBody := mockedResponse.Body.(map[string]interface{})
	expectedToken := respBody["ims_token"].(string)
	if imstoken != expectedToken {
		t.Fatalf("IMS Token mismatch, expected (%s), got (%s)", expectedToken, imstoken)
	}

	expectedIMSUserID := respBody["ims_user_id"].(int)
	if imsuserid != expectedIMSUserID {
		t.Fatalf("IMS User ID mismatch, expected (%d), got (%d)", expectedIMSUserID, imsuserid)
	}
}

func TestFetchIMSToken_failure_invalid_ibmid_crdentials(t *testing.T) {
	c := &Config{IBMID: "id", IBMIDPassword: "pass", RetryCount: 1, RetryDelay: 1 * time.Second}
	mockedRequest := mockIAMRequest(c, "")
	mockedResponse := &MockedResponse{
		StatusCode:  401,
		ContentType: "application/json",
		Body: IMSTokenErrorResponse{
			Code:    "BXNIM0602E",
			Message: "The credentials you provided are incorrect",
			Details: "The credentials you entered for the user '" + c.IBMID + "' are incorrect",
		},
	}
	transactions := []*MockedTransaction{
		{
			Request:  mockedRequest,
			Response: mockedResponse,
		},
	}
	server := getMockIAMServer(transactions)
	c.IAMEndpoint = server.URL
	defer server.Close()

	_, _, err := fetchIMSToken(c, "")
	if err == nil {
		t.Fatalf("Expecting Authorization error but no error occured")
	}

	if !strings.Contains(err.Error(), "Client request to fetch IMS token failed with response code 401") {
		t.Fatalf("Auhtorization failure Code 401 is not present in the error message: %v", err)
	}
}

func TestFetchIMSToken_failure_invalid_json_response_from_iam(t *testing.T) {
	c := &Config{IBMID: "id", IBMIDPassword: "pass", RetryCount: 1, RetryDelay: 1 * time.Second}
	mockedRequest := mockIAMRequest(c, "")
	mockedResponse := &MockedResponse{
		StatusCode:      200,
		ContentType:     "application/json",
		SendInvalidJSON: true,
	}
	transactions := []*MockedTransaction{
		{
			Request:  mockedRequest,
			Response: mockedResponse,
		},
	}
	server := getMockIAMServer(transactions)
	c.IAMEndpoint = server.URL
	defer server.Close()

	_, _, err := fetchIMSToken(c, "")
	if err == nil {
		t.Fatalf("Expecting unmarshalling error but no error occured")
	}

	if !strings.Contains(err.Error(), "Couldn't unmarshall the  IMSTokenResponse invalid character") {
		t.Fatalf("unmarshalling error is not present in the error message: %v", err)
	}
}

func TestFetchSoftLayerAPIKey_Success(t *testing.T) {
	c := &Config{SoftLayerTimeout: 10 * time.Second}
	slUserID := acctest.RandInt()
	stringifiedUserID := strconv.Itoa(slUserID)
	slAuthToken := "ims_token"
	slUsername := "slusername"
	slAPIKey := "slapikey"

	mockedRequest := mockSoftLayerXMLRPCRequest(`<member><name>userId</name><value><int>` + stringifiedUserID + `</int></value></member>`)
	mockedResponse := &MockedResponse{
		StatusCode:  200,
		ContentType: "text/xml",
		Body:        `<?xml version="1.0" encoding="utf-8"?><params><param><value><struct><member><name>username</name><value><string>` + slUsername + `</string></value></member><member><name>apiAuthenticationKeys</name><value><array><data><value><struct><member><name>authenticationKey</name><value><string>` + slAPIKey + `</string></value></member></struct></value></data></array></value></member></struct></value></param></params>`,
	}
	transactions := []*MockedTransaction{
		{
			Request:  mockedRequest,
			Response: mockedResponse,
		},
	}
	server := getMockSoftlayerXMLRPCServer(transactions)
	c.SoftlayerXMLRPCEndpoint = server.URL + "/xmlrpc/"
	defer server.Close()
	actualUsername, actualAPIKey, err := fetchSoftLayerAPIKey(c, slUserID, slAuthToken)

	if err != nil {
		t.Fatalf("Error fetching softlayer API Key and Username  for %v", err)
	}

	if actualUsername != slUsername {
		t.Fatalf("SoftLayer Username mismatch, expected (%s), got (%s)", slUsername, actualUsername)
	}

	if actualAPIKey != slAPIKey {
		t.Fatalf("SoftLayer API Key mismatch, expected (%s), got (%s)", slAPIKey, actualAPIKey)
	}

}

func TestFetchSoftLayerAPIKey_invalid_userid(t *testing.T) {
	c := &Config{SoftLayerTimeout: 10 * time.Second}
	slUserID := acctest.RandInt()
	invalidUserID := strconv.Itoa(slUserID)
	slAuthToken := "ims_token"

	mockedRequest := mockSoftLayerXMLRPCRequest(`<member><name>userId</name><value><int>` + invalidUserID + `</int></value></member>`)
	mockedResponse := &MockedResponse{
		StatusCode:  200,
		ContentType: "text/xml",
		Body:        `<?xml version="1.0" encoding="iso-8859-1"?><methodResponse><fault><value><struct><member><name>faultCode</name><value><string>SoftLayer_Exception_ObjectNotFound</string></value></member><member><name>faultString</name><value><string>Unable to find object with id of '` + invalidUserID + `'.</string></value></member></struct></value></fault></methodResponse>`,
	}

	transactions := []*MockedTransaction{
		{
			Request:  mockedRequest,
			Response: mockedResponse,
		},
	}
	server := getMockSoftlayerXMLRPCServer(transactions)
	c.SoftlayerXMLRPCEndpoint = server.URL + "/xmlrpc/"
	defer server.Close()

	_, _, err := fetchSoftLayerAPIKey(c, slUserID, slAuthToken)

	if err == nil {
		t.Fatalf("Expecting invalid softlayer user id error but didn't get any error")
	}
	apiErr := err.(sl.Error)
	expectedException := "SoftLayer_Exception_ObjectNotFound"
	if apiErr.Exception != expectedException {
		t.Fatalf("Expecting exception message (%s) got (%v)", expectedException, apiErr.Exception)
	}

	expectedErrMessage := fmt.Sprintf("Unable to find object with id of '%d'.", slUserID)
	if apiErr.Message != expectedErrMessage {
		t.Fatalf("Expecting error message (%s) got (%v)", expectedErrMessage, apiErr.Message)
	}
}

func TestFetchSoftLayerAPIKey_invalid_authentication_token(t *testing.T) {
	c := &Config{SoftLayerTimeout: 10 * time.Second}
	slUserID := acctest.RandInt()
	invalidSlAuthToken := acctest.RandString(65)

	mockedRequest := mockSoftLayerXMLRPCRequest(`<member><name>authToken</name><value><string>` + invalidSlAuthToken + `</string></value></member>`)
	mockedResponse := &MockedResponse{
		StatusCode:  200,
		ContentType: "text/xml",
		Body:        `<?xml version="1.0" encoding="iso-8859-1"?><methodResponse><fault><value><struct><member><name>faultCode</name><value><string>SoftLayer_Exception_InvalidLegacyToken</string></value></member><member><name>faultString</name><value><string>Invalid authentication token.</string></value></member></struct></value></fault></methodResponse>`,
	}

	transactions := []*MockedTransaction{
		{
			Request:  mockedRequest,
			Response: mockedResponse,
		},
	}
	server := getMockSoftlayerXMLRPCServer(transactions)
	c.SoftlayerXMLRPCEndpoint = server.URL + "/xmlrpc/"
	defer server.Close()

	_, _, err := fetchSoftLayerAPIKey(c, slUserID, invalidSlAuthToken)

	apiErr := err.(sl.Error)
	expectedException := "SoftLayer_Exception_InvalidLegacyToken"
	if apiErr.Exception != expectedException {
		t.Fatalf("Expecting exception message (%s) got (%v)", expectedException, apiErr.Exception)
	}

	expectedErrMessage := "Invalid authentication token."
	if apiErr.Message != expectedErrMessage {
		t.Fatalf("Expecting error message (%s) got (%v)", expectedErrMessage, apiErr.Message)
	}

}

func getMockIAMServer(m []*MockedTransaction) *httptest.Server {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody interface{}
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		requestBody = r.Form
		log.Printf("[DEBUG] Received API %q request to %q:", r.Method, r.RequestURI)

		for _, t := range m {
			if r.Method == t.Request.Method && r.RequestURI == t.Request.URI && reflect.DeepEqual(t.Request.Body, requestBody) {
				if t.Response.SendInvalidJSON {
					w.WriteHeader(t.Response.StatusCode)
					w.Header().Set("Content-Type", t.Response.ContentType)
					fmt.Fprintln(w, "invalid json")
					return
				}
				b := new(bytes.Buffer)
				if err := json.NewEncoder(b).Encode(t.Response.Body); err != nil {
					w.WriteHeader(500)
					w.Write([]byte(err.Error()))
					return
				}
				w.WriteHeader(t.Response.StatusCode)
				w.Header().Set("Content-Type", t.Response.ContentType)
				w.Write(b.Bytes())
				return
			}
		}
		w.WriteHeader(400)
		return

	}))
	return s
}

func getMockSoftlayerXMLRPCServer(m []*MockedTransaction) *httptest.Server {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		requestBody := buf.String()

		log.Printf("[DEBUG] Received API %q request to %q", r.Method, r.RequestURI)
		for _, t := range m {
			if r.Method == t.Request.Method && r.RequestURI == t.Request.URI && t.Request.ShouldServeRequest(requestBody) {
				w.WriteHeader(t.Response.StatusCode)
				w.Header().Set("Content-Type", t.Response.ContentType)
				w.Write([]byte(t.Response.Body.(string)))
				return
			}
		}
		log.Printf("[DEBUG] Couldn't find any mocked response to give")
		w.WriteHeader(400)
		return

	}))
	return s
}

func setEnv(envs map[string]string, t *testing.T) func() {
	c := getCurrentEnv()
	for k, v := range envs {
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("Error setting env var %s: %s", k, err)
		}
	}
	return func() {
		resetEnv(*c, t)
	}
}

func resetEnv(e map[string]string, t *testing.T) {
	resetHelper := func(env, val string, t *testing.T) {
		if err := os.Setenv(env, val); err != nil {
			t.Fatalf("Error resetting env var %s: %s", env, err)
		}
	}
	for k, v := range e {
		resetHelper(k, v, t)
	}
}

func getCurrentEnv() *map[string]string {
	envs := make(map[string]string)
	for _, v := range AllEnvs {
		envs[v] = os.Getenv(v)
	}
	return &envs
}

type MockedTransaction struct {
	Response *MockedResponse
	Request  *MockedRequest
}

type MockedResponse struct {
	StatusCode      int
	ContentType     string
	Body            interface{}
	SendInvalidJSON bool
	ContentEncoding string
}

type MockedRequest struct {
	Method string
	URI    string
	Body   interface{}
	//Mocked server will invoke this function on mocked request to decide if it should serve the request
	//Useful when we don't want to make this decision based on complicated request bodies
	//Mainly used by the Mocked XML RPC server where xmls contents could be complicated
	ShouldServeRequest func(interface{}) bool
	AcceptType         string
	ContentType        string
}

func mockIAMRequest(c *Config, iamToken string) *MockedRequest {
	if iamToken != "" {
		return &MockedRequest{
			Method:     "POST",
			URI:        iamServerTokenRequestURI,
			AcceptType: "application/json",
			Body: url.Values{
				"grant_type":    {"urn:ibm:params:oauth:grant-type:derive"},
				"access_token":  {iamToken},
				"response_type": {"ims_portal"},
			},
		}
	}
	return &MockedRequest{
		Method:     "POST",
		URI:        iamServerTokenRequestURI,
		AcceptType: "application/json",
		Body: url.Values{
			"grant_type":    {"password"},
			"username":      {c.IBMID},
			"password":      {c.IBMIDPassword},
			"ims_account":   {""},
			"response_type": {"cloud_iam,ims_portal"},
		},
	}
}

func mockSoftLayerXMLRPCRequest(matchString string) *MockedRequest {
	return &MockedRequest{
		Method:      "POST",
		ContentType: "text/xml",
		AcceptType:  "text/xml",
		URI:         "/xmlrpc/SoftLayer_User_Customer",
		ShouldServeRequest: func(xml interface{}) bool {
			return strings.Contains(xml.(string), matchString)
		},
	}
}

var AllEnvs = []string{
	"IBM_ID",
	"IBMID_PASSWORD",
	"BM_REGION",
	"BLUEMIX_REGION",
	"SL_TIMEOUT",
	"SOFTLAYER_TIMEOUT",
	"SL_ACCOUNT_NUMBER",
	"SOFTLAYER_ACCOUNT_NUMBER",
}
