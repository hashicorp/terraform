package ibmcloud

import (
	json "encoding/json"
	"errors"
	"log"

	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	slsession "github.com/softlayer/softlayer-go/session"
)

// DefaultTimeout is default  API timeout if not specified
const DefaultTimeout = time.Second * 60

//DefaultSoftLayerEndpoint is the default API endpoint of SoftLayer
const DefaultSoftLayerEndpoint = "https://api.softlayer.com/rest/v3"

//DefaultRegion is the default Blumix region
const DefaultRegion = "ng"

//ErrorBluemixParamaterValidation sums the insufficient credentials to login to Bluemix API
var ErrorBluemixParamaterValidation = errors.New("Either IBM ID and password (IBMID/IBMID_PASSWORD environment " +
	"variable) or Bluemix identity cookie via BLUEMIX_IDENTITY_COOKIE environment variable are required")

//ErrorSoftLayerParamaterValidation sums the insufficient credentials to login to SoftLayer API
var ErrorSoftLayerParamaterValidation = errors.New("Either softlayer_username and softlayer_api_key (SOFTLAYER_USERNAME and SOFTLAYER_API_KEY environment variable" +
	" or Softlayer Account number (SOFTLAYER_ACCOUNT_NUMBER environment variable) are required")

// Session stores the information required for communication with the SoftLayer
// and Bluemix API
type Session struct {
	// Timeout specifies a time limit for http requests made by this
	// session. Requests that take longer that the specified timeout
	// will result in an error.
	Timeout time.Duration

	// AccessToken is the token secret for token-based authentication
	AccessToken string

	// AccessToken is the token secret for token-based authentication
	RefreshToken string

	// IdentityCookie is used to aquire a SoftLayer token
	IdentityCookie string

	// SoftLayer IMS token
	SoftLayerIMSToken string

	// SoftLayer user ID
	SoftLayerUserID int

	// SoftLayerSesssion is the the SoftLayer session used to connect to the SoftLayer API
	SoftLayerSession *slsession.Session

	IAMClientID string
	IAMSecret   string

	//HTTP Client
	HTTPClient *http.Client
}

// newSession creates and returns a pointer to a new session object
// from a provided Config
func newSession(c *Config) (*Session, error) {

	//These are not exposed in schema at this point
	//and are meant to be used internally by IBM Cloud
	identityCookie := ValueFromEnv("identity_cookie")
	iamClientID := ValueFromEnv("iam_client_id")
	iamSecret := ValueFromEnv("iam_secret")

	// username/password or identity cookie needs to be provided
	// identity cookie is meant to be used internally at this point
	if (c.IBMID == "" || c.Password == "") && (identityCookie == "") {
		return nil, ErrorBluemixParamaterValidation
	}

	// Bluemix timeout
	timeout := c.Timeout
	var timeoutDuration time.Duration
	if timeout != "" {
		timeoutDuration, _ = time.ParseDuration(fmt.Sprintf("%ss", timeout))
	} else {
		timeoutDuration, _ = time.ParseDuration(fmt.Sprintf("%ss", DefaultTimeout))
	}

	if (c.SoftLayerUsername == "" || c.SoftLayerAPIKey == "") && (c.SoftLayerAccountNumber == "") {
		return nil, ErrorSoftLayerParamaterValidation
	}

	bluemixSession := &Session{
		IdentityCookie: identityCookie,
		IAMClientID:    iamClientID,
		IAMSecret:      iamSecret,
		Timeout:        timeoutDuration,
	}

	bluemixSession.HTTPClient = cleanhttp.DefaultClient()
	bluemixSession.HTTPClient.Timeout = timeoutDuration

	err := bluemixSession.authenticate(c)
	if err != nil {
		return nil, err
	}

	if c.SoftLayerEndpointURL == "" {
		c.SoftLayerEndpointURL = DefaultSoftLayerEndpoint
	}

	softlayerSession := slsession.New(
		c.SoftLayerUsername,
		c.SoftLayerAPIKey,
		c.SoftLayerEndpointURL,
		c.SoftLayerTimeout,
	)

	if os.Getenv("TF_LOG") != "" {
		softlayerSession.Debug = true
	}

	// if the SoftLayer IMS account is provided, retrieve the IMS token
	if c.SoftLayerAccountNumber != "" {

		if identityCookie == "" {
			// creates the identity cookie
			err = bluemixSession.createIdentityCookie(c)
		}
		if err != nil {
			return bluemixSession, err
		}
		// obtain an IMS token
		err := bluemixSession.createIMSToken(c)
		if err != nil {
			return bluemixSession, err
		}
		softlayerSession.UserId = bluemixSession.SoftLayerUserID
		softlayerSession.AuthToken = bluemixSession.SoftLayerIMSToken

	}
	bluemixSession.SoftLayerSession = softlayerSession
	return bluemixSession, nil
}

//Authenticate against Bluemix
func (s *Session) authenticate(c *Config) error {

	// Create body for token request
	bodyAsValues := url.Values{
		"grant_type": {"password"},
		"username":   {c.IBMID},
		"password":   {c.Password},
	}

	authURL := fmt.Sprintf("%s/oauth/token", c.Endpoint)

	authHeaders := map[string]string{
		"Authorization": "Basic Y2Y6",
		"Content-Type":  "application/x-www-form-urlencoded",
	}

	type AuthenticationErrorResponse struct {
		Code        string `json:"error"`
		Description string `json:"error_description"`
	}

	type AuthenticationResponse struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		AuthenticationErrorResponse
	}

	req, err := http.NewRequest("POST", authURL, strings.NewReader(bodyAsValues.Encode()))
	if err != nil {
		return err
	}
	for k, v := range authHeaders {
		req.Header.Add(k, v)
	}
	response, err := s.HTTPClient.Do(req)

	if err != nil {
		return err
	}

	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)

	// unmarshall the response
	var jsonResponse AuthenticationResponse
	err = json.Unmarshal(responseBody, &jsonResponse)

	s.AccessToken = jsonResponse.AccessToken
	s.RefreshToken = jsonResponse.RefreshToken
	errorCode := jsonResponse.Code
	if errorCode != "" {
		return fmt.Errorf("Bluemix authentication failed %s", string(responseBody))
	}
	return err
}

func (s *Session) createIdentityCookie(c *Config) error {
	bodyAsValues := url.Values{
		"grant_type":    {"password"},
		"username":      {c.IBMID},
		"password":      {c.Password},
		"response_type": {"identity_cookie"},
	}

	authURL := fmt.Sprintf("%s/oauth/token", c.Endpoint)
	authHeaders := map[string]string{
		"Authorization": "Basic Y2Y6",
		"Content-Type":  "application/x-www-form-urlencoded",
	}
	type IdentityCookieErrorResponse struct {
		Code        string `json:"error"`
		Description string `json:"error_description"`
	}

	type IdentityCookieResponse struct {
		Expiration     int64  `json:"expiration"`
		IdentityCookie string `json:"identity_cookie"`
		IdentityCookieErrorResponse
	}

	req, err := http.NewRequest("POST", authURL, strings.NewReader(bodyAsValues.Encode()))
	if err != nil {
		return err
	}
	for k, v := range authHeaders {
		req.Header.Add(k, v)
	}
	response, err := s.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)

	var jsonResponse IdentityCookieResponse
	err = json.Unmarshal(responseBody, &jsonResponse)

	s.IdentityCookie = jsonResponse.IdentityCookie
	errorCode := jsonResponse.Code
	if errorCode != "" {
		return fmt.Errorf("Bluemix createIdentityCookie failed, %s", string(responseBody))
	}
	return err
}

func (s *Session) createIMSToken(c *Config) error {
	log.Printf("[INFO] Creating the IMS token...")
	bodyAsValues := url.Values{
		"grant_type":    {"urn:ibm:params:oauth:grant-type:identity-cookie"},
		"cookie":        {s.IdentityCookie},
		"ims_account":   {c.SoftLayerAccountNumber},
		"response_type": {"cloud_iam, ims_portal"},
	}

	authURL := fmt.Sprintf("%s/oidc/token", c.IAMEndpoint)

	authHeaders := map[string]string{
		"Authorization": "Basic Y2Y6",
		"Content-Type":  "application/x-www-form-urlencoded",
	}
	type IMSTokenErrorResponse struct {
		Code        string `json:"error"`
		Description string `json:"error_description"`
	}

	type IMSTokenResponse struct {
		IMSToken   string `json:"ims_token"`
		IMSUserID  int    `json:"ims_user_id"`
		TokenType  string `json:"token_type"`
		ExpiresIn  int    `json:"expires_in"`
		Expiration int64  `json:"expiration"`
		IMSTokenErrorResponse
	}

	count := 10
	for count > 0 {
		req, _ := http.NewRequest("POST", authURL, strings.NewReader(bodyAsValues.Encode()))
		for k, v := range authHeaders {
			req.Header.Add(k, v)
		}
		req.SetBasicAuth(s.IAMClientID, s.IAMSecret)

		response, err := s.HTTPClient.Do(req)
		if err != nil {
			log.Println("[WARNING] Error occurred while aquiring the IMS token:  ", err)
			time.Sleep(1000 * time.Millisecond)
			count--
			continue
		}
		if response.StatusCode != 200 {
			log.Printf("[ERROR] Response Status: %s", response.Status)
			time.Sleep(1000 * time.Millisecond)
			response.Body.Close()
			count--
			continue
		}
		responseBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println("[WARNING] Error occurred while reading the HTTP response body:  ", err)
			time.Sleep(1000 * time.Millisecond)
			count--
			continue
		}
		var jsonResponse IMSTokenResponse
		err = json.Unmarshal(responseBody, &jsonResponse)

		if err != nil {
			log.Println("[WARNING] Error occurred while unmarshalling the IMSTokenResponse:  ", err)
			time.Sleep(1000 * time.Millisecond)
			count--
			continue
		}
		if jsonResponse.IMSToken == "" {
			log.Printf("[WARNING] Retrying to aquire the IMS token...")
			time.Sleep(1000 * time.Millisecond)
			count--
			continue
		}
		log.Printf("[INFO] IMS token aquired")
		s.SoftLayerIMSToken = jsonResponse.IMSToken
		s.SoftLayerUserID = jsonResponse.IMSUserID
		response.Body.Close()
		return nil
	}
	return errors.New("[ERROR] Failed to retrieve the IMS token")
}

// ValueFromEnv will return the value for param from tne environment if it's set, or "" if not set
func ValueFromEnv(paramName string) string {
	var defValue string

	switch paramName {
	case "ibmid":
		defValue = os.Getenv("IBMID")

	case "password":
		defValue = os.Getenv("IBMID_PASSWORD")

	case "softlayer_username":
		// Prioritize SL_USERNAME
		defValue = os.Getenv("SL_USERNAME")
		if defValue == "" {
			defValue = os.Getenv("SOFTLAYER_USERNAME")
		}

	case "softlayer_api_key":
		// Prioritize SL_API_KEY
		defValue = os.Getenv("SL_API_KEY")
		if defValue == "" {
			defValue = os.Getenv("SOFTLAYER_API_KEY")
		}

	case "identity_cookie":
		// Prioritize BM_IDENTITY_COOKIE
		defValue = os.Getenv("BM_IDENTITY_COOKIE")
		if defValue == "" {
			defValue = os.Getenv("BLUEMIX_IDENTITY_COOKIE")
		}

	case "region":
		// Prioritize BM_REGION
		defValue = os.Getenv("BM_REGION")
		if defValue == "" {
			defValue = os.Getenv("BLUEMIX_REGION")
		}
		if defValue == "" {
			defValue = DefaultRegion
		}

	case "timeout":
		// Prioritize BM_TIMEOUT
		defValue = os.Getenv("BM_TIMEOUT")
		if defValue == "" {
			defValue = os.Getenv("BLUEMIX_TIMEOUT")
		}

	case "iam_client_id":
		// Prioritize BM_IAM_CLIENT_ID
		defValue = os.Getenv("BM_IAM_CLIENT_ID")
		if defValue == "" {
			defValue = os.Getenv("BLUEMIX_IAM_CLIENT_ID")
		}

	case "iam_secret":
		// Prioritize BM_IAM_SECRET
		defValue = os.Getenv("BM_IAM_SECRET")
		if defValue == "" {
			defValue = os.Getenv("BLUEMIX_IAM_SECRET")
		}

	case "softlayer_account_number":
		// PRIORITIZE SL_ACCOUNT_NUMBER
		defValue = os.Getenv("SL_ACCOUNT_NUMBER")
		if defValue == "" {
			defValue = os.Getenv("SOFTLAYER_ACCOUNT_NUMBER")
		}
	}

	return defValue
}

func envFallback(value *string, paramName string) {
	if *value == "" {
		*value = ValueFromEnv(paramName)
	}
}
