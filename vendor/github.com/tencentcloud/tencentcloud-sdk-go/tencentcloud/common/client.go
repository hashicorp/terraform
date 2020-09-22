package common

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

type Client struct {
	region          string
	httpClient      *http.Client
	httpProfile     *profile.HttpProfile
	profile         *profile.ClientProfile
	credential      *Credential
	signMethod      string
	unsignedPayload bool
	debug           bool
}

func (c *Client) Send(request tchttp.Request, response tchttp.Response) (err error) {
	if request.GetDomain() == "" {
		domain := c.httpProfile.Endpoint
		if domain == "" {
			domain = tchttp.GetServiceDomain(request.GetService())
		}
		request.SetDomain(domain)
	}

	if request.GetHttpMethod() == "" {
		request.SetHttpMethod(c.httpProfile.ReqMethod)
	}

	tchttp.CompleteCommonParams(request, c.GetRegion())

	if c.signMethod == "HmacSHA1" || c.signMethod == "HmacSHA256" {
		return c.sendWithSignatureV1(request, response)
	} else {
		return c.sendWithSignatureV3(request, response)
	}
}

func (c *Client) sendWithSignatureV1(request tchttp.Request, response tchttp.Response) (err error) {
	// TODO: not an elegant way, it should be done in common params, but finally it need to refactor
	request.GetParams()["Language"] = c.profile.Language
	err = tchttp.ConstructParams(request)
	if err != nil {
		return err
	}
	err = signRequest(request, c.credential, c.signMethod)
	if err != nil {
		return err
	}
	httpRequest, err := http.NewRequest(request.GetHttpMethod(), request.GetUrl(), request.GetBodyReader())
	if err != nil {
		return err
	}
	if request.GetHttpMethod() == "POST" {
		httpRequest.Header["Content-Type"] = []string{"application/x-www-form-urlencoded"}
	}
	if c.debug {
		outbytes, err := httputil.DumpRequest(httpRequest, true)
		if err != nil {
			log.Printf("[ERROR] dump request failed because %s", err)
			return err
		}
		log.Printf("[DEBUG] http request = %s", outbytes)
	}
	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return err
	}
	err = tchttp.ParseFromHttpResponse(httpResponse, response)
	return err
}

func (c *Client) sendWithSignatureV3(request tchttp.Request, response tchttp.Response) (err error) {
	headers := map[string]string{
		"Host":               request.GetDomain(),
		"X-TC-Action":        request.GetAction(),
		"X-TC-Version":       request.GetVersion(),
		"X-TC-Timestamp":     request.GetParams()["Timestamp"],
		"X-TC-RequestClient": request.GetParams()["RequestClient"],
		"X-TC-Language":      c.profile.Language,
	}
	if c.region != "" {
		headers["X-TC-Region"] = c.region
	}
	if c.credential.Token != "" {
		headers["X-TC-Token"] = c.credential.Token
	}
	if request.GetHttpMethod() == "GET" {
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	} else {
		headers["Content-Type"] = "application/json"
	}

	// start signature v3 process

	// build canonical request string
	httpRequestMethod := request.GetHttpMethod()
	canonicalURI := "/"
	canonicalQueryString := ""
	if httpRequestMethod == "GET" {
		err = tchttp.ConstructParams(request)
		if err != nil {
			return err
		}
		params := make(map[string]string)
		for key, value := range request.GetParams() {
			params[key] = value
		}
		delete(params, "Action")
		delete(params, "Version")
		delete(params, "Nonce")
		delete(params, "Region")
		delete(params, "RequestClient")
		delete(params, "Timestamp")
		canonicalQueryString = tchttp.GetUrlQueriesEncoded(params)
	}
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\n", headers["Content-Type"], headers["Host"])
	signedHeaders := "content-type;host"
	requestPayload := ""
	if httpRequestMethod == "POST" {
		b, err := json.Marshal(request)
		if err != nil {
			return err
		}
		requestPayload = string(b)
	}
	hashedRequestPayload := ""
	if c.unsignedPayload {
		hashedRequestPayload = sha256hex("UNSIGNED-PAYLOAD")
		headers["X-TC-Content-SHA256"] = "UNSIGNED-PAYLOAD"
	} else {
		hashedRequestPayload = sha256hex(requestPayload)
	}
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload)
	//log.Println("canonicalRequest:", canonicalRequest)

	// build string to sign
	algorithm := "TC3-HMAC-SHA256"
	requestTimestamp := headers["X-TC-Timestamp"]
	timestamp, _ := strconv.ParseInt(requestTimestamp, 10, 64)
	t := time.Unix(timestamp, 0).UTC()
	// must be the format 2006-01-02, ref to package time for more info
	date := t.Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, request.GetService())
	hashedCanonicalRequest := sha256hex(canonicalRequest)
	string2sign := fmt.Sprintf("%s\n%s\n%s\n%s",
		algorithm,
		requestTimestamp,
		credentialScope,
		hashedCanonicalRequest)
	//log.Println("string2sign", string2sign)

	// sign string
	secretDate := hmacsha256(date, "TC3"+c.credential.SecretKey)
	secretService := hmacsha256(request.GetService(), secretDate)
	secretKey := hmacsha256("tc3_request", secretService)
	signature := hex.EncodeToString([]byte(hmacsha256(string2sign, secretKey)))
	//log.Println("signature", signature)

	// build authorization
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		c.credential.SecretId,
		credentialScope,
		signedHeaders,
		signature)
	//log.Println("authorization", authorization)

	headers["Authorization"] = authorization
	url := "https://" + request.GetDomain() + request.GetPath()
	if canonicalQueryString != "" {
		url = url + "?" + canonicalQueryString
	}
	httpRequest, err := http.NewRequest(httpRequestMethod, url, strings.NewReader(requestPayload))
	if err != nil {
		return err
	}
	for k, v := range headers {
		httpRequest.Header[k] = []string{v}
	}
	if c.debug {
		outbytes, err := httputil.DumpRequest(httpRequest, true)
		if err != nil {
			log.Printf("[ERROR] dump request failed because %s", err)
			return err
		}
		log.Printf("[DEBUG] http request = %s", outbytes)
	}
	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return err
	}
	err = tchttp.ParseFromHttpResponse(httpResponse, response)
	return err
}

func (c *Client) GetRegion() string {
	return c.region
}

func (c *Client) Init(region string) *Client {
	c.httpClient = &http.Client{}
	c.region = region
	c.signMethod = "TC3-HMAC-SHA256"
	c.debug = false
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	return c
}

func (c *Client) WithSecretId(secretId, secretKey string) *Client {
	c.credential = NewCredential(secretId, secretKey)
	return c
}

func (c *Client) WithCredential(cred *Credential) *Client {
	c.credential = cred
	return c
}

func (c *Client) WithProfile(clientProfile *profile.ClientProfile) *Client {
	c.profile = clientProfile
	c.signMethod = clientProfile.SignMethod
	c.unsignedPayload = clientProfile.UnsignedPayload
	c.httpProfile = clientProfile.HttpProfile
	c.httpClient.Timeout = time.Duration(c.httpProfile.ReqTimeout) * time.Second
	return c
}

func (c *Client) WithSignatureMethod(method string) *Client {
	c.signMethod = method
	return c
}

func (c *Client) WithHttpTransport(transport http.RoundTripper) *Client {
	c.httpClient.Transport = transport
	return c
}

func NewClientWithSecretId(secretId, secretKey, region string) (client *Client, err error) {
	client = &Client{}
	client.Init(region).WithSecretId(secretId, secretKey)
	return
}
