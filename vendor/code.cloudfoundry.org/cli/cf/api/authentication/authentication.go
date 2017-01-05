package authentication

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	. "code.cloudfoundry.org/cli/cf/i18n"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . TokenRefresher

type TokenRefresher interface {
	RefreshAuthToken() (updatedToken string, apiErr error)
}

//go:generate counterfeiter . Repository

type Repository interface {
	net.RequestDumperInterface

	RefreshAuthToken() (updatedToken string, apiErr error)
	Authenticate(credentials map[string]string) (apiErr error)
	Authorize(token string) (string, error)
	GetLoginPromptsAndSaveUAAServerURL() (map[string]coreconfig.AuthPrompt, error)
}

type UAARepository struct {
	config  coreconfig.ReadWriter
	gateway net.Gateway
	dumper  net.RequestDumper
}

var ErrPreventRedirect = errors.New("prevent-redirect")

func NewUAARepository(gateway net.Gateway, config coreconfig.ReadWriter, dumper net.RequestDumper) UAARepository {
	return UAARepository{
		config:  config,
		gateway: gateway,
		dumper:  dumper,
	}
}

func (uaa UAARepository) Authorize(token string) (string, error) {
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			uaa.DumpRequest(req)
			return ErrPreventRedirect
		},
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: uaa.config.IsSSLDisabled(),
			},
			Proxy:               http.ProxyFromEnvironment,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	authorizeURL, err := url.Parse(uaa.config.UaaEndpoint())
	if err != nil {
		return "", err
	}

	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("grant_type", "authorization_code")
	values.Set("client_id", uaa.config.SSHOAuthClient())

	authorizeURL.Path = "/oauth/authorize"
	authorizeURL.RawQuery = values.Encode()

	authorizeReq, err := http.NewRequest("GET", authorizeURL.String(), nil)
	if err != nil {
		return "", err
	}

	authorizeReq.Header.Add("authorization", token)

	resp, err := httpClient.Do(authorizeReq)
	if resp != nil {
		uaa.DumpResponse(resp)
	}
	if err == nil {
		return "", errors.New(T("Authorization server did not redirect with one time code"))
	}

	if netErr, ok := err.(*url.Error); !ok || netErr.Err != ErrPreventRedirect {
		return "", errors.New(T("Error requesting one time code from server: {{.Error}}", map[string]interface{}{"Error": err.Error()}))
	}

	loc, err := resp.Location()
	if err != nil {
		return "", errors.New(T("Error getting the redirected location: {{.Error}}", map[string]interface{}{"Error": err.Error()}))
	}

	codes := loc.Query()["code"]
	if len(codes) != 1 {
		return "", errors.New(T("Unable to acquire one time code from authorization response"))
	}

	return codes[0], nil
}

func (uaa UAARepository) Authenticate(credentials map[string]string) error {
	data := url.Values{
		"grant_type": {"password"},
		"scope":      {""},
	}
	for key, val := range credentials {
		data[key] = []string{val}
	}

	err := uaa.getAuthToken(data)
	if err != nil {
		httpError, ok := err.(errors.HTTPError)
		if ok {
			switch {
			case httpError.StatusCode() == http.StatusUnauthorized:
				return errors.New(T("Credentials were rejected, please try again."))
			case httpError.StatusCode() >= http.StatusInternalServerError:
				return errors.New(T("The targeted API endpoint could not be reached."))
			}
		}

		return err
	}

	return nil
}

func (uaa UAARepository) DumpRequest(req *http.Request) {
	uaa.dumper.DumpRequest(req)
}

func (uaa UAARepository) DumpResponse(res *http.Response) {
	uaa.dumper.DumpResponse(res)
}

type LoginResource struct {
	Prompts map[string][]string
	Links   map[string]string
}

var knownAuthPromptTypes = map[string]coreconfig.AuthPromptType{
	"text":     coreconfig.AuthPromptTypeText,
	"password": coreconfig.AuthPromptTypePassword,
}

func (r *LoginResource) parsePrompts() (prompts map[string]coreconfig.AuthPrompt) {
	prompts = make(map[string]coreconfig.AuthPrompt)
	for key, val := range r.Prompts {
		prompts[key] = coreconfig.AuthPrompt{
			Type:        knownAuthPromptTypes[val[0]],
			DisplayName: val[1],
		}
	}
	return
}

func (uaa UAARepository) GetLoginPromptsAndSaveUAAServerURL() (prompts map[string]coreconfig.AuthPrompt, apiErr error) {
	url := fmt.Sprintf("%s/login", uaa.config.AuthenticationEndpoint())
	resource := &LoginResource{}
	apiErr = uaa.gateway.GetResource(url, resource)

	prompts = resource.parsePrompts()
	if resource.Links["uaa"] == "" {
		uaa.config.SetUaaEndpoint(uaa.config.AuthenticationEndpoint())
	} else {
		uaa.config.SetUaaEndpoint(resource.Links["uaa"])
	}
	return
}

func (uaa UAARepository) RefreshAuthToken() (string, error) {
	data := url.Values{
		"refresh_token": {uaa.config.RefreshToken()},
		"grant_type":    {"refresh_token"},
		"scope":         {""},
	}

	apiErr := uaa.getAuthToken(data)
	updatedToken := uaa.config.AccessToken()

	return updatedToken, apiErr
}

func (uaa UAARepository) getAuthToken(data url.Values) error {
	type uaaErrorResponse struct {
		Code        string `json:"error"`
		Description string `json:"error_description"`
	}

	type AuthenticationResponse struct {
		AccessToken  string           `json:"access_token"`
		TokenType    string           `json:"token_type"`
		RefreshToken string           `json:"refresh_token"`
		Error        uaaErrorResponse `json:"error"`
	}

	path := fmt.Sprintf("%s/oauth/token", uaa.config.AuthenticationEndpoint())
	request, err := uaa.gateway.NewRequest("POST", path, "Basic "+base64.StdEncoding.EncodeToString([]byte("cf:")), strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("%s: %s", T("Failed to start oauth request"), err.Error())
	}
	request.HTTPReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response := new(AuthenticationResponse)
	_, err = uaa.gateway.PerformRequestForJSONResponse(request, &response)

	switch err.(type) {
	case nil:
	case errors.HTTPError:
		return err
	case *errors.InvalidTokenError:
		return errors.New(T("Authentication has expired.  Please log back in to re-authenticate.\n\nTIP: Use `cf login -a <endpoint> -u <user> -o <org> -s <space>` to log back in and re-authenticate."))
	default:
		return fmt.Errorf("%s: %s", T("auth request failed"), err.Error())
	}

	// TODO: get the actual status code
	if response.Error.Code != "" {
		return errors.NewHTTPError(0, response.Error.Code, response.Error.Description)
	}

	uaa.config.SetAccessToken(fmt.Sprintf("%s %s", response.TokenType, response.AccessToken))
	uaa.config.SetRefreshToken(response.RefreshToken)

	return nil
}
