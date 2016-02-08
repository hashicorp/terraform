package azure

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/Godeps/_workspace/src/github.com/Azure/go-autorest/autorest"
	"github.com/Azure/azure-sdk-for-go/Godeps/_workspace/src/github.com/dgrijalva/jwt-go"
)

const (
	defaultRefresh = 5 * time.Minute
	oauthURL       = "https://login.microsoftonline.com/{tenantID}/oauth2/{requestType}?api-version=1.0"
	tokenBaseDate  = "1970-01-01T00:00:00Z"

	jwtAudienceTemplate = "https://login.microsoftonline.com/%s/oauth2/token"

	// AzureResourceManagerScope is the OAuth scope for the Azure Resource Manager.
	AzureResourceManagerScope = "https://management.azure.com/"
)

var expirationBase time.Time

func init() {
	expirationBase, _ = time.Parse(time.RFC3339, tokenBaseDate)
}

// Token encapsulates the access token used to authorize Azure requests.
type Token struct {
	AccessToken string `json:"access_token"`

	ExpiresIn string `json:"expires_in"`
	ExpiresOn string `json:"expires_on"`
	NotBefore string `json:"not_before"`

	Resource string `json:"resource"`
	Type     string `json:"token_type"`
}

// Expires returns the time.Time when the Token expires.
func (t Token) Expires() time.Time {
	s, err := strconv.Atoi(t.ExpiresOn)
	if err != nil {
		s = -3600
	}
	return expirationBase.Add(time.Duration(s) * time.Second).UTC()
}

// IsExpired returns true if the Token is expired, false otherwise.
func (t Token) IsExpired() bool {
	return t.WillExpireIn(0)
}

// WillExpireIn returns true if the Token will expire after the passed time.Duration interval
// from now, false otherwise.
func (t Token) WillExpireIn(d time.Duration) bool {
	return !t.Expires().After(time.Now().Add(d))
}

// WithAuthorization returns a PrepareDecorator that adds an HTTP Authorization header whose
// value is "Bearer " followed by the AccessToken of the Token.
func (t *Token) WithAuthorization() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			return (autorest.WithBearerAuthorization(t.AccessToken)(p)).Prepare(r)
		})
	}
}

// ServicePrincipalSecret is an interface that allows various secret mechanism to fill the form
// that is submitted when acquiring an oAuth token.
type ServicePrincipalSecret interface {
	SetAuthenticationValues(spt *ServicePrincipalToken, values *url.Values) error
}

// ServicePrincipalTokenSecret implements ServicePrincipalSecret for client_secret type authorization.
type ServicePrincipalTokenSecret struct {
	ClientSecret string
}

// SetAuthenticationValues is a method of the interface ServicePrincipalTokenSecret.
// It will populate the form submitted during oAuth Token Acquisition using the client_secret.
func (tokenSecret *ServicePrincipalTokenSecret) SetAuthenticationValues(spt *ServicePrincipalToken, v *url.Values) error {
	v.Set("client_secret", tokenSecret.ClientSecret)
	return nil
}

// ServicePrincipalCertificateSecret implements ServicePrincipalSecret for generic RSA cert auth with signed JWTs.
type ServicePrincipalCertificateSecret struct {
	Certificate *x509.Certificate
	PrivateKey  *rsa.PrivateKey
}

// SignJwt returns the JWT signed with the certificate's private key.
func (secret *ServicePrincipalCertificateSecret) SignJwt(spt *ServicePrincipalToken) (string, error) {
	hasher := sha1.New()
	_, err := hasher.Write(secret.Certificate.Raw)
	if err != nil {
		return "", err
	}

	thumbprint := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	// The jti (JWT ID) claim provides a unique identifier for the JWT.
	jti := make([]byte, 20)
	_, err = rand.Read(jti)
	if err != nil {
		return "", err
	}

	token := jwt.New(jwt.SigningMethodRS256)
	token.Header["x5t"] = thumbprint
	token.Claims = map[string]interface{}{
		"aud": fmt.Sprintf(jwtAudienceTemplate, spt.tenantID),
		"iss": spt.clientID,
		"sub": spt.clientID,
		"jti": base64.URLEncoding.EncodeToString(jti),
		"nbf": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	}

	signedString, err := token.SignedString(secret.PrivateKey)
	return signedString, nil
}

// SetAuthenticationValues is a method of the interface ServicePrincipalTokenSecret.
// It will populate the form submitted during oAuth Token Acquisition using a JWT signed with a certificate.
func (secret *ServicePrincipalCertificateSecret) SetAuthenticationValues(spt *ServicePrincipalToken, v *url.Values) error {
	jwt, err := secret.SignJwt(spt)
	if err != nil {
		return err
	}

	v.Set("client_assertion", jwt)
	v.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	return nil
}

// ServicePrincipalToken encapsulates a Token created for a Service Principal.
type ServicePrincipalToken struct {
	Token

	secret        ServicePrincipalSecret
	clientID      string
	tenantID      string
	resource      string
	autoRefresh   bool
	refreshWithin time.Duration
	sender        autorest.Sender
}

// NewServicePrincipalTokenWithSecret create a ServicePrincipalToken using the supplied ServicePrincipalSecret implementation.
func NewServicePrincipalTokenWithSecret(id string, tenantID string, resource string, secret ServicePrincipalSecret) (*ServicePrincipalToken, error) {
	spt := &ServicePrincipalToken{
		secret:        secret,
		clientID:      id,
		resource:      resource,
		tenantID:      tenantID,
		autoRefresh:   true,
		refreshWithin: defaultRefresh,
		sender:        &http.Client{},
	}
	return spt, nil
}

// NewServicePrincipalToken creates a ServicePrincipalToken from the supplied Service Principal
// credentials scoped to the named resource.
func NewServicePrincipalToken(id string, secret string, tenantID string, resource string) (*ServicePrincipalToken, error) {
	return NewServicePrincipalTokenWithSecret(
		id,
		tenantID,
		resource,
		&ServicePrincipalTokenSecret{
			ClientSecret: secret,
		},
	)
}

// NewServicePrincipalTokenFromCertificate create a ServicePrincipalToken from the supplied pkcs12 bytes.
func NewServicePrincipalTokenFromCertificate(id string, certificate *x509.Certificate, privateKey *rsa.PrivateKey, tenantID string, resource string) (*ServicePrincipalToken, error) {
	return NewServicePrincipalTokenWithSecret(
		id,
		tenantID,
		resource,
		&ServicePrincipalCertificateSecret{
			PrivateKey:  privateKey,
			Certificate: certificate,
		},
	)
}

// EnsureFresh will refresh the token if it will expire within the refresh window (as set by
// RefreshWithin).
func (spt *ServicePrincipalToken) EnsureFresh() error {
	if spt.WillExpireIn(spt.refreshWithin) {
		return spt.Refresh()
	}
	return nil
}

// Refresh obtains a fresh token for the Service Principal.
func (spt *ServicePrincipalToken) Refresh() error {
	p := map[string]interface{}{
		"tenantID":    spt.tenantID,
		"requestType": "token",
	}

	v := url.Values{}
	v.Set("client_id", spt.clientID)
	v.Set("grant_type", "client_credentials")
	v.Set("resource", spt.resource)

	err := spt.secret.SetAuthenticationValues(spt, &v)
	if err != nil {
		return err
	}

	req, err := autorest.Prepare(&http.Request{},
		autorest.AsPost(),
		autorest.AsFormURLEncoded(),
		autorest.WithBaseURL(oauthURL),
		autorest.WithPathParameters(p),
		autorest.WithFormData(v))
	if err != nil {
		return err
	}

	resp, err := autorest.SendWithSender(spt.sender, req)
	if err != nil {
		return autorest.NewErrorWithError(err,
			"azure.ServicePrincipalToken", "Refresh", resp.StatusCode, "Failure sending request for Service Principal %s",
			spt.clientID)
	}

	var newToken Token

	err = autorest.Respond(resp,
		autorest.WithErrorUnlessOK(),
		autorest.ByUnmarshallingJSON(&newToken),
		autorest.ByClosing())
	if err != nil {
		return autorest.NewErrorWithError(err,
			"azure.ServicePrincipalToken", "Refresh", resp.StatusCode, "Failure handling response to Service Principal %s request",
			spt.clientID)
	}

	spt.Token = newToken

	return nil
}

// SetAutoRefresh enables or disables automatic refreshing of stale tokens.
func (spt *ServicePrincipalToken) SetAutoRefresh(autoRefresh bool) {
	spt.autoRefresh = autoRefresh
}

// SetRefreshWithin sets the interval within which if the token will expire, EnsureFresh will
// refresh the token.
func (spt *ServicePrincipalToken) SetRefreshWithin(d time.Duration) {
	spt.refreshWithin = d
	return
}

// SetSender sets the autorest.Sender used when obtaining the Service Principal token. An
// undecorated http.Client is used by default.
func (spt *ServicePrincipalToken) SetSender(s autorest.Sender) {
	spt.sender = s
}

// WithAuthorization returns a PrepareDecorator that adds an HTTP Authorization header whose
// value is "Bearer " followed by the AccessToken of the ServicePrincipalToken.
//
// By default, the token will automatically refresh if nearly expired (as determined by the
// RefreshWithin interval). Use the AutoRefresh method to enable or disable automatically refreshing
// tokens.
func (spt *ServicePrincipalToken) WithAuthorization() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			if spt.autoRefresh {
				err := spt.EnsureFresh()
				if err != nil {
					return r, autorest.NewErrorWithError(err,
						"azure.ServicePrincipalToken", "WithAuthorization", autorest.UndefinedStatusCode, "Failed to refresh Service Principal Token for request to %s",
						r.URL)
				}
			}
			return (autorest.WithBearerAuthorization(spt.AccessToken)(p)).Prepare(r)
		})
	}
}
