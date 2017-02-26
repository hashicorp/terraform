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
	"github.com/hashicorp/terraform/helper/schema"
	slsession "github.com/softlayer/softlayer-go/session"
)

const (
	uaaServerTokenRequestURI = "/oauth/token"
	iamServerTokenRequestURI = "/oidc/token"
)

//ErrorBluemixParamaterValidation sums the insufficient credentials to login to Bluemix API
var ErrorBluemixParamaterValidation = errors.New("Either ibmid and password (IBMID/IBMID_PASSWORD environment " +
	"variable) or Bluemix identity cookie via BLUEMIX_IDENTITY_COOKIE environment variable are required")

//ErrorSoftLayerParamaterValidation sums the insufficient credentials to login to SoftLayer API
var ErrorSoftLayerParamaterValidation = errors.New("Either softlayer_username and softlayer_api_key " +
	"(SOFTLAYER_USERNAME and SOFTLAYER_API_KEY environment variable  or softlayer_account_number " +
	"(SOFTLAYER_ACCOUNT_NUMBER environment variable) are required")

//ErrorIMSTokenRetrieval announces the error in retrieving IMS Token
var ErrorIMSTokenRetrieval = errors.New("[ERROR] Failed to retrieve the IMS token")

// Session stores the information required for communication with the SoftLayer
// and Bluemix API
// At this point we don't have a bluemix-go client. Once this is in place
// All this code will be a part of that sdk. For now define the required session here
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

	identityCookie, err := valueFromEnv("identity_cookie")
	if err != nil {
		return nil, err
	}

	// ibmid/password or identity cookie needs to be provided
	// identity cookie is meant to be used internally at this point
	if (c.IBMID == "" || c.Password == "") && (identityCookie == "") {
		return nil, ErrorBluemixParamaterValidation
	}

	// either SoftLayer username/password or the SoftLayer account number must be provided
	if (c.SoftLayerUsername == "" || c.SoftLayerAPIKey == "") && (c.SoftLayerAccountNumber == "") {
		return nil, ErrorSoftLayerParamaterValidation
	}

	iamClientID, err := valueFromEnv("iam_client_id")
	if err != nil {
		return nil, err
	}
	iamSecret, err := valueFromEnv("iam_secret")
	if err != nil {
		return nil, err
	}

	// Bluemix timeout
	timeout := c.Timeout
	timeoutDuration, _ := time.ParseDuration(fmt.Sprintf("%ss", timeout))

	bluemixSession := &Session{
		IdentityCookie: identityCookie,
		IAMClientID:    iamClientID,
		IAMSecret:      iamSecret,
		Timeout:        timeoutDuration,
	}

	bluemixSession.HTTPClient = cleanhttp.DefaultClient()
	bluemixSession.HTTPClient.Timeout = timeoutDuration

	err = bluemixSession.authenticate(c)
	if err != nil {
		return nil, err
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
			err = bluemixSession.createIdentityCookie(c)
		}
		if err != nil {
			return bluemixSession, err
		}
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

	authURL := fmt.Sprintf("%s%s", c.Endpoint, uaaServerTokenRequestURI)

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
		return fmt.Errorf("error occurred while composing request to retrieve acess token: %s ", err)
	}
	for k, v := range authHeaders {
		req.Header.Add(k, v)
	}
	response, err := s.HTTPClient.Do(req)

	if err != nil {
		return fmt.Errorf("error occurred while retrieving acces token: %s ", err)
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

	authURL := fmt.Sprintf("%s%s", c.Endpoint, uaaServerTokenRequestURI)
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
		return fmt.Errorf("error occurred while composing request to create identity cookie: %s ", err)
	}
	for k, v := range authHeaders {
		req.Header.Add(k, v)
	}
	response, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("error occurred while creating identity cookie: %s ", err)
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

	authURL := fmt.Sprintf("%s%s", c.IAMEndpoint, iamServerTokenRequestURI)

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

	//retry parameters
	count := 5
	delay := 4000 * time.Millisecond

	for count > 0 {
		req, _ := http.NewRequest("POST", authURL, strings.NewReader(bodyAsValues.Encode()))
		for k, v := range authHeaders {
			req.Header.Add(k, v)
		}
		req.SetBasicAuth(s.IAMClientID, s.IAMSecret)

		response, err := s.HTTPClient.Do(req)
		if err != nil {
			log.Println("[WARNING] Error occurred while aquiring the IMS token:  ", err)
			time.Sleep(delay)
			count--
			continue
		}
		if response.StatusCode != 200 {
			log.Printf("[WARNING] Response Status: %s", response.Status)
			time.Sleep(delay)
			response.Body.Close()
			count--
			continue
		}
		responseBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println("[WARNING] Error occurred while reading the HTTP response body:  ", err)
			time.Sleep(delay)
			count--
			continue
		}
		var jsonResponse IMSTokenResponse
		err = json.Unmarshal(responseBody, &jsonResponse)

		if err != nil {
			log.Println("[WARNING] Error occurred while unmarshalling the IMSTokenResponse:  ", err)
			time.Sleep(delay)
			count--
			continue
		}
		if jsonResponse.IMSToken == "" {
			log.Printf("[WARNING] Retrying to aquire the IMS token...")
			time.Sleep(delay)
			count--
			continue
		}
		log.Printf("[INFO] IMS token aquired")
		s.SoftLayerIMSToken = jsonResponse.IMSToken
		s.SoftLayerUserID = jsonResponse.IMSUserID
		response.Body.Close()
		return nil
	}
	return ErrorIMSTokenRetrieval
}

func valueFromEnv(paramName string) (string, error) {

	switch paramName {

	case "identity_cookie":
		//These envs not exposed in schema at this point and are meant to be used internally by IBM Cloud
		identityCookie, err := schema.MultiEnvDefaultFunc([]string{"BM_IDENTITY_COOKIE", "BLUEMIX_IDENTITY_COOKIE"}, "")()
		if err != nil {
			return "", err
		}
		return identityCookie.(string), nil

	case "iam_client_id":
		//These envs not exposed in schema at this point and are meant to be used internally by IBM Cloud
		iamClientID, err := schema.MultiEnvDefaultFunc([]string{"BM_IAM_CLIENT_ID", "BLUEMIX_IAM_CLIENT_ID"}, "")()
		if err != nil {
			return "", err
		}

		return iamClientID.(string), nil

	case "iam_secret":
		//These envs not exposed in schema at this point and are meant to be used internally by IBM Cloud
		iamSecret, err := schema.MultiEnvDefaultFunc([]string{"BM_IAM_SECRET", "BLUEMIX_IAM_SECRET"}, "")()
		if err != nil {
			return "", err
		}
		return iamSecret.(string), nil

	default:
		return "", fmt.Errorf("Invalid parameter provided to fetch from Environment variables %s", paramName)
	}

}
