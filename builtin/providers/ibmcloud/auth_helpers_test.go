package ibmcloud

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestSessionMissingBluemixRequiredParamters(t *testing.T) {
	c := &Config{}
	_, err := newSession(c)

	if err != ErrorBluemixParamaterValidation {
		t.Fatalf("Expecting error: %s, instead got: %v", ErrorBluemixParamaterValidation, err)
	}
}

func TestSessionMissingSoftlayerRequiredParamters(t *testing.T) {
	c := &Config{IBMID: "user", Password: "password"}
	_, err := newSession(c)

	if err != ErrorSoftLayerParamaterValidation {
		t.Fatalf("Expecting error: %s, instead got: %v", ErrorSoftLayerParamaterValidation, err)
	}
}

func TestSessionMissingSoftlayerRequiredParamtersWithIdentityCookie(t *testing.T) {
	c := &Config{}
	resetEnv := setEnv(map[string]string{
		"BM_IDENTITY_COOKIE": "cookie",
	}, t)
	defer resetEnv()
	_, err := newSession(c)

	if err != ErrorSoftLayerParamaterValidation {
		t.Fatalf("Expecting error: %s, instead got: %v", ErrorSoftLayerParamaterValidation, err)
	}
}

func TestSessionAuthenticateError(t *testing.T) {
	c := &Config{IBMID: "user", Password: "password", SoftLayerUsername: "suser", SoftLayerAPIKey: "apikey"}
	m := responseMessages["Authenticate_401"]

	resp := &mockServerResponse{uaaToken: uaaTokenResponse{401, m}}

	mockedServer := mockServer(resp)
	defer mockedServer.Close()
	c.Endpoint = mockedServer.URL

	_, sessErr := newSession(c)

	if sessErr == nil {
		t.Fatal("Expecting Bluemix Authentication error,   got nil")
	}

	errMessage, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("error unmarshalling the authenticate response message %v", err)
	}

	if !strings.Contains(sessErr.Error(), string(errMessage)) {
		t.Fatalf("Expecting Bluemix Authentication failed error,   got %v", sessErr)
	}
}

func TestSessionAuthenticateSuccess(t *testing.T) {
	c := &Config{IBMID: "user", Password: "password", SoftLayerUsername: "suser", SoftLayerAPIKey: "apikey"}
	m := responseMessages["Authenticate_200"]
	resp := &mockServerResponse{uaaToken: uaaTokenResponse{200, m}}
	mockedServer := mockServer(resp)
	defer mockedServer.Close()
	c.Endpoint = mockedServer.URL

	s, err := newSession(c)
	if err != nil {
		t.Fatalf("Expecting no error for bluemix authentication but got %v", err)
	}
	validateTokens(s, m, t)

}

func TestSessionCreateIMSTokenSuccess(t *testing.T) {
	c := &Config{IBMID: "user", Password: "password", SoftLayerAccountNumber: "1234"}
	authMessage := responseMessages["Authenticate_200"]
	identityMessage := responseMessages["Identity_200"]
	iamMessage := responseMessages["IAM_200"]
	resp := &mockServerResponse{
		uaaToken:          uaaTokenResponse{200, authMessage},
		iam:               iamResponse{200, iamMessage},
		uaaIdentityCookie: uaaIdentityCookieResponse{200, identityMessage},
	}
	mockedServer := mockServer(resp)
	defer mockedServer.Close()
	//Use same mocked server for both
	c.Endpoint = mockedServer.URL
	c.IAMEndpoint = mockedServer.URL

	s, err := newSession(c)
	if err != nil {
		t.Fatalf("Expecting no error for session creation but got %v", err)
	}

	expectedIMSToken := iamMessage["ims_token"].(string)
	if s.SoftLayerIMSToken != expectedIMSToken {
		t.Fatalf("Expecting IMSToken to be %s but actual is %s", expectedIMSToken, s.SoftLayerIMSToken)
	}

	expectedIdentityCookie := identityMessage["identity_cookie"].(string)
	if s.IdentityCookie != expectedIdentityCookie {
		t.Fatalf("Expecting IMSToken to be %s but actual is %s", expectedIdentityCookie, s.IdentityCookie)
	}

	validateTokens(s, authMessage, t)
}

func TestSessionCreateIMSTokenFailure(t *testing.T) {
	c := &Config{IBMID: "user", Password: "password", SoftLayerAccountNumber: "1234"}
	authMessage := responseMessages["Authenticate_200"]
	identityMessage := responseMessages["Identity_200"]
	iamMessage := responseMessages["IAM_401"]
	resp := &mockServerResponse{
		uaaToken:          uaaTokenResponse{200, authMessage},
		iam:               iamResponse{401, iamMessage},
		uaaIdentityCookie: uaaIdentityCookieResponse{200, identityMessage},
	}
	mockedServer := mockServer(resp)
	defer mockedServer.Close()
	//Use same mocked server for both
	c.Endpoint = mockedServer.URL
	c.IAMEndpoint = mockedServer.URL

	_, err := newSession(c)
	if err != ErrorIMSTokenRetrieval {
		t.Fatalf("Expecting error %v but got %v", ErrorIMSTokenRetrieval, err)
	}
}

func TestSessionCreateIdentityCookieFailure(t *testing.T) {
	c := &Config{IBMID: "user", Password: "password", SoftLayerAccountNumber: "1234"}
	authMessage := responseMessages["Authenticate_200"]
	identityMessage := responseMessages["Identity_401"]

	resp := &mockServerResponse{
		uaaToken:          uaaTokenResponse{200, authMessage},
		uaaIdentityCookie: uaaIdentityCookieResponse{401, identityMessage},
	}
	mockedServer := mockServer(resp)
	defer mockedServer.Close()
	c.Endpoint = mockedServer.URL
	_, sessErr := newSession(c)
	if sessErr == nil {
		t.Fatalf("Expecting error for session creation but got %v", sessErr)
	}

	m, err := json.Marshal(identityMessage)
	if err != nil {
		t.Fatalf("error unmarshalling the identity response message %v", err)
	}

	if !strings.Contains(sessErr.Error(), string(m)) {
		t.Fatalf("Expecting Bluemix createIdentityCookie failed error,   got %v", sessErr)
	}

}

func validateTokens(s *Session, m map[string]interface{}, t *testing.T) {

	expectedAccessToken := m["access_token"].(string)
	expectedRefreshToken := m["refresh_token"].(string)

	if s.AccessToken != expectedAccessToken {
		t.Fatalf("Expecting AccessToken to be %s but actual is %s", expectedAccessToken, s.AccessToken)
	}

	if s.RefreshToken != expectedRefreshToken {
		t.Fatalf("Expecting RefreshToken to be %s but actual is %s", expectedRefreshToken, s.RefreshToken)
	}
}

func mockServer(resp *mockServerResponse) *httptest.Server {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		requestBody := buf.String()
		rURI := r.RequestURI

		encoded, statusCode, err := []byte{}, 400, errors.New("Bad Request")
		switch rURI {

		case uaaServerTokenRequestURI:
			if strings.Contains(string(requestBody), "response_type=identity_cookie") {
				encoded, err = json.Marshal(resp.uaaIdentityCookie.message)
				statusCode = resp.uaaIdentityCookie.status

			} else if strings.Contains(string(requestBody), "grant_type=password") {
				encoded, err = json.Marshal(resp.uaaToken.message)
				statusCode = resp.uaaToken.status
			}

		case iamServerTokenRequestURI:
			encoded, err = json.Marshal(resp.iam.message)
			statusCode = resp.iam.status

		}

		if err != nil {
			w.WriteHeader(statusCode)
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write(encoded)
	}))

	return s
}

var responseMessages = map[string]map[string]interface{}{
	"Authenticate_401": map[string]interface{}{
		"error_description": "Authentication failure - your login credentials are invalid.",
		"error":             "unauthorized",
	},
	"Authenticate_200": map[string]interface{}{
		"access_token":  "some_token",
		"token_type":    "bearer",
		"refresh_token": "refresh_token",
	},
	"Identity_401": map[string]interface{}{
		"error_description": "Authentication failure - your login credentials are invalid.",
		"error":             "unauthorized",
	},
	"Identity_200": map[string]interface{}{
		"identity_cookie": "some_cookie",
		"expiration":      12345,
	},
	"IAM_200": map[string]interface{}{
		"ims_token":   "some_ims_token",
		"token_type":  "bearer",
		"ims_user_id": 23,
		"expires_in":  1212,
		"expiration":  1111,
	},
	"IAM_500": map[string]interface{}{
		"error":             "End point erred",
		"error_description": "some error occured",
	},
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

type iamResponse struct {
	status  int
	message map[string]interface{}
}

type uaaTokenResponse struct {
	status  int
	message map[string]interface{}
}

type uaaIdentityCookieResponse struct {
	status  int
	message map[string]interface{}
}

type mockServerResponse struct {
	iam               iamResponse
	uaaToken          uaaTokenResponse
	uaaIdentityCookie uaaIdentityCookieResponse
}

var AllEnvs = []string{
	"IBM_ID",
	"IBMID_PASSWORD",
	"SL_USERNAME",
	"SOFTLAYER_USERNAME",
	"SL_API_KEY",
	"SOFTLAYER_API_KEY",
	"BM_IDENTITY_COOKIE",
	"BLUEMIX_IDENTITY_COOKIE",
	"BM_REGION",
	"BLUEMIX_REGION",
	"BM_TIMEOUT",
	"BLUEMIX_TIMEOUT",
	"BM_IAM_CLIENT_ID",
	"BLUEMIX_IAM_CLIENT_ID",
	"BM_IAM_SECRET",
	"BLUEMIX_IAM_SECRET",
	"SL_ACCOUNT_NUMBER",
	"SOFTLAYER_ACCOUNT_NUMBER",
}
