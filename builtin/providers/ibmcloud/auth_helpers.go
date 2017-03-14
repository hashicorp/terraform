package ibmcloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"os"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	slservices "github.com/softlayer/softlayer-go/services"
	slsession "github.com/softlayer/softlayer-go/session"
)

const (
	iamServerTokenRequestURI = "/oidc/token"
)

//TODO Define Error types to use for better error handling but that  alongwith below codes might go into the bluemix sdk.

//SoftlayerRestEndpoint rest endpoint of SoftLayer
const SoftlayerRestEndpoint = "https://api.softlayer.com/rest/v3"

//SoftlayerXMLRPCEndpoint rest endpoint of SoftLayer
const SoftlayerXMLRPCEndpoint = "https://api.softlayer.com/xmlrpc/v3"

//ErrBluemixParamaterValidation sums the insufficient credentials to login to Bluemix API
var ErrBluemixParamaterValidation = errors.New("ibmid and ibmid_password are required paramters." +
	"Either mention in the provider block or source them from IBMID/IBMID_PASSWORD environment")

// Session stores the information required for communication with the SoftLayer
// and Bluemix API
// At this point we don't have a bluemix-go client. Once this is in place
// All this code will be a part of that sdk. For now define the required session here
type Session struct {
	// SoftLayer IMS token
	SoftLayerIMSToken string

	// SoftLayer user ID
	SoftLayerUserID int

	// SoftLayer username
	SoftLayerUserName string

	// SoftLayer apikey
	SoftLayerAPIKey string

	// SoftLayerSesssion is the the SoftLayer session used to connect to the SoftLayer API
	SoftLayerSession *slsession.Session
}

//IMSTokenErrorResponse encapsulates the error response from authentication with IAM
type IMSTokenErrorResponse struct {
	Code    string `json:"errorCode"`
	Message string `json:"errorMessage"`
	Details string `json:"errorDetails"`
}

func (err IMSTokenErrorResponse) Error() string {
	return fmt.Sprintf("Code: (%s), Message: (%s), Details: (%s)", err.Code, err.Message, err.Details)
}

//IMSTokenResponse encapsulates the response from authentication with IAM
type IMSTokenResponse struct {
	IMSToken   string `json:"ims_token"`
	IMSUserID  int    `json:"ims_user_id"`
	TokenType  string `json:"token_type"`
	ExpiresIn  int    `json:"expires_in"`
	Expiration int64  `json:"expiration"`
	IMSTokenErrorResponse
}

// newSession creates and returns a pointer to a new session object
// from a provided Config
func newSession(c *Config) (*Session, error) {
	iamToken := os.Getenv("IBMCLOUD_IAM_TOKEN")
	// ibmid/ibmid_password or iam token needs to be provided
	// iam token is meant to be used internally at this point by IBM cloud hence don't
	// show that in error to the regular user of terraform
	if (c.IBMID == "" || c.IBMIDPassword == "") && (iamToken == "") {
		return nil, ErrBluemixParamaterValidation
	}

	imstoken, imsuserid, err := fetchIMSToken(c, iamToken)
	if err != nil {
		return nil, err
	}
	bmxSession := &Session{}
	bmxSession.SoftLayerIMSToken = imstoken
	bmxSession.SoftLayerUserID = imsuserid

	slUsername, slAPIKey, err := fetchSoftLayerAPIKey(c, imsuserid, imstoken)
	if err != nil {
		return nil, err
	}
	bmxSession.SoftLayerUserName = slUsername
	bmxSession.SoftLayerAPIKey = slAPIKey
	softlayerSession := &slsession.Session{
		Endpoint: c.SoftLayerEndpointURL,
		Timeout:  c.SoftLayerTimeout,
		UserName: slUsername,
		APIKey:   slAPIKey,
		Debug:    os.Getenv("TF_LOG") != "",
	}
	bmxSession.SoftLayerSession = softlayerSession
	return bmxSession, nil
}

func fetchIMSToken(c *Config, iamToken string) (imstoken string, imsuserid int, err error) {
	log.Printf("[INFO] Fetching the IMS token...")
	var bodyAsValues url.Values
	if iamToken != "" {
		bodyAsValues = url.Values{
			"grant_type":    {"urn:ibm:params:oauth:grant-type:derive"},
			"access_token":  {iamToken},
			"response_type": {"ims_portal"},
		}
	} else {
		bodyAsValues = url.Values{
			"grant_type":    {"password"},
			"username":      {c.IBMID},
			"password":      {c.IBMIDPassword},
			"ims_account":   {c.SoftLayerAccountNumber},
			"response_type": {"cloud_iam,ims_portal"},
		}
	}
	authURL := fmt.Sprintf("%s%s", c.IAMEndpoint, iamServerTokenRequestURI)
	headers := map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Authorization": "Basic Yng6Yng=",
	}
	//retry parameters
	count := c.RetryCount + 1
	delay := c.RetryDelay

	httpClient := cleanhttp.DefaultClient()
	httpClient.Timeout = 60 * time.Second

	for count > 0 {
		var req *http.Request
		req, err = http.NewRequest("POST", authURL, strings.NewReader(bodyAsValues.Encode()))
		if err != nil {
			return "", 0, fmt.Errorf("Failed to compose Auth request to %s: %v", authURL, err)
		}
		for k, v := range headers {
			req.Header.Add(k, v)
		}
		var response *http.Response
		response, err = httpClient.Do(req)
		if err != nil {
			log.Println("[WARNING] Error occurred while aquiring the IMS token:  ", err)
			err = fmt.Errorf("Client request to fetch IMS token failed %v", err)
			time.Sleep(delay)
			count--
			continue
		}
		if response.StatusCode != 200 {
			log.Printf("[WARNING] Response Status: %s", response.Status)
			err = fmt.Errorf("Client request to fetch IMS token failed with response code %d", response.StatusCode)
			time.Sleep(delay)
			response.Body.Close()
			count--
			continue
		}
		var responseBody []byte
		responseBody, err = ioutil.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			log.Println("[WARNING] Error occurred while reading the HTTP response body:  ", err)
			err = fmt.Errorf("Couldn't read the server response at %s.%v", authURL, err)
			time.Sleep(delay)
			count--
			continue
		}
		var jsonResponse IMSTokenResponse
		err = json.Unmarshal(responseBody, &jsonResponse)
		if err != nil {
			log.Println("[WARNING] Error occurred while unmarshalling the IMSTokenResponse:  ", err)
			err = fmt.Errorf("Couldn't unmarshall the  IMSTokenResponse %v", err)
			time.Sleep(delay)
			count--
			continue
		}
		if jsonResponse.Code != "" {
			//should never happen as status code is 200 here
			log.Printf("[SEVERE] Permanent failure occured while acquiring IMS token")
			return "", 0, jsonResponse
		}

		log.Printf("[INFO] IMS token aquired")
		imstoken = jsonResponse.IMSToken
		imsuserid = jsonResponse.IMSUserID
		return
	}
	return "", 0, err
}

func fetchSoftLayerAPIKey(c *Config, sluserID int, slToken string) (slUsername, slAPIKey string, err error) {
	sess := &slsession.Session{
		Endpoint: c.SoftlayerXMLRPCEndpoint,
		Timeout:  c.SoftLayerTimeout,
		Debug:    os.Getenv("TF_LOG") != "",
	}
	sess.UserId = sluserID
	sess.AuthToken = slToken
	userService := slservices.GetUserCustomerService(sess)
	cust, err := userService.Id(sess.UserId).Mask("username;apiAuthenticationKeys.authenticationKey").GetObject()
	if err != nil {
		return "", "", err
	}
	if len(cust.ApiAuthenticationKeys) == 0 {
		return "", "", errors.New("Couldn't find any API Keys")
	}
	slAPIKey = *cust.ApiAuthenticationKeys[0].AuthenticationKey
	slUsername = *cust.Username
	return
}
