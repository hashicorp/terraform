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

const (
	UAAServerTokenRequestURI = "/oauth/token"
	IAMServerTokenRequestURI = "/oidc/token"
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
	respFunc := func(rURI, requestBody string) (encoded []byte, statusCode int, err error) {
		if strings.Contains(string(requestBody), "grant_type=password") {
			encoded, err = json.Marshal(m)
			statusCode = 401
			return
		}
		return nil, 0, errors.New("Invalid request")
	}
	mockedServer := mockServer(respFunc)
	defer mockedServer.Close()
	c.Endpoint = mockedServer.URL

	_, err := newSession(c)

	if err == nil {
		t.Fatal("Expecting Bluemix Authentication error,   got nil")
	}

	if !strings.Contains(err.Error(), "Bluemix authentication failed ") {
		t.Fatalf("Expecting Bluemix Authentication failed error,   got %v", err)
	}
}

func TestSessionAuthenticateSuccess(t *testing.T) {
	c := &Config{IBMID: "user", Password: "password", SoftLayerUsername: "suser", SoftLayerAPIKey: "apikey"}
	m := responseMessages["Authenticate_200"]
	respFunc := func(rURI, requestBody string) (encoded []byte, statusCode int, err error) {
		if strings.Contains(string(requestBody), "grant_type=password") {
			encoded, err = json.Marshal(m)
			statusCode = 200
			return
		}
		return nil, 0, errors.New("Invalid request")
	}

	mockedServer := mockServer(respFunc)
	defer mockedServer.Close()
	c.Endpoint = mockedServer.URL

	s, err := newSession(c)

	expectedAccessToken := m["access_token"].(string)
	expectedRefreshToken := m["refresh_token"].(string)

	if err != nil {
		t.Fatalf("Expecting no error for bluemix authentication but got %v", err)
	}

	if s.AccessToken != expectedAccessToken {
		t.Fatalf("Expecting AccessToken to be %s but actual is %s", expectedAccessToken, s.AccessToken)
	}

	if s.RefreshToken != expectedRefreshToken {
		t.Fatalf("Expecting RefreshToken to be %s but actual is %s", expectedRefreshToken, s.RefreshToken)
	}
}

func TestSessionCreateIMSTokenSuccess(t *testing.T) {
	c := &Config{IBMID: "user", Password: "password", SoftLayerAccountNumber: "1234"}
	authMessage := responseMessages["Authenticate_200"]
	identityMessage := responseMessages["Identity_200"]
	iamMessage := responseMessages["IAM_200"]

	respFunc := func(rURI string, rBody string) (encoded []byte, statusCode int, err error) {
		if strings.Contains(rURI, UAAServerTokenRequestURI) {
			if strings.Contains(string(rBody), "response_type=identity_cookie") {
				encoded, err = json.Marshal(identityMessage)
				statusCode = 200
				return
			}
			if strings.Contains(string(rBody), "grant_type=password") {
				encoded, err = json.Marshal(authMessage)
				statusCode = 200
				return
			}
		}
		if strings.Contains(rURI, IAMServerTokenRequestURI) {
			encoded, err = json.Marshal(iamMessage)
			statusCode = 200
			return
		}

		return nil, 0, errors.New("Invalid request")
	}

	mockedServer := mockServer(respFunc)
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

}

func mockServer(respf func(rURI, rBody string) (respBody []byte, statusCode int, err error)) *httptest.Server {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		requestBody := buf.String()
		encoded, statusCode, err := respf(r.RequestURI, requestBody)

		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Internal server error"))
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

var AllEnvs = []string{"IBM_ID", "IBMID_PASSWORD", "SL_USERNAME", "SOFTLAYER_USERNAME", "SL_API_KEY", "SOFTLAYER_API_KEY", "BM_IDENTITY_COOKIE",
	"BLUEMIX_IDENTITY_COOKIE", "BM_REGION", "BLUEMIX_REGION", "BM_TIMEOUT", "BLUEMIX_TIMEOUT",
	"BM_IAM_CLIENT_ID", "BLUEMIX_IAM_CLIENT_ID", "BM_IAM_SECRET", "BLUEMIX_IAM_SECRET",
	"SL_ACCOUNT_NUMBER", "SOFTLAYER_ACCOUNT_NUMBER",
}
