package acme

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xenolf/lego/acme"
)

const testDirResponseText = `
{
  "new-reg": "https://example.com/acme/new-reg",
  "new-authz": "https://example.com/acme/new-authz",
  "new-cert": "https://example.com/acme/new-cert",
  "revoke-cert": "https://example.com/acme/revoke-cert",
  "meta": {
    "terms-of-service": "https://example.com/acme/terms",
    "website": "https://www.example.com/",
    "caa-identities": ["example.com"]
  }
}
`

func newHTTPTestServer(f func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(f))
	return ts
}

func httpDirTestServer() *httptest.Server {
	return newHTTPTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		http.Error(w, testDirResponseText, http.StatusOK)
	})
}

const testPrivateKeyText = `
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA8XXIc0dO5okTzukP2USWm5tbxY6FQzzvBbOpxIfVpdKpZcKV
HfemqCZEIGu/3P3gI6rAYmDRYvLsbKSjKA5EzuUvVxrLzqPZyFI5mzF0gGEzEvYk
Z+mCPLsS5VwaXCySiz6vIBpItw6KOHByt5v8iMtgppGmjWX5N2oeVZ5314xmXFV3
OMlniyC1uLk6Y/bVtv/vK1mOATXP5vejpjBHdk/VYTTXRZp3zSZllKJbtbt2CxY4
eA55oCc9cfF46rNPsAsiH5iGbBFIIDSqscukZ9BtBZUj+kO+63he0SedzppuosKi
i9YtjgG1Mb81vgMFZ/SQeiR5FONWcH61jTSkiQIDAQABAoIBAQDJVYK8zLq3c5k2
sBLtAUnrmhFdm0b3F7neMT7fhrvYtt1U4njgMf2eu7mWpwGmTXI1i007OqudLB2D
QYxh+/PX6DYfFVLXjLwtUpKCGyyfV2z05JTaqFRWO064PKImNWxD+xKfXAtByDfs
c6bT/pcFoT+H5G7R/DNfx3ZfwfD/oo2aUCQT8PrwzQ9cjJuLYzu5Dwma29Cxtajd
Gsdrd09Qkm0PCM3c0FHG7fV3zq5SNw53tP0U0lNzSzpRiS6wmLAPDy3CcKGaj+9K
5YIE3OoQKRFn7hQkHxgnZlBJJU2BOBAOMJA6s28iRNy3pQOzR0M2kqf+YTQk/i13
if2/mvU5AoGBAPtT9XVbOu6U4Q9WyBSi5nI4AG7gHeJtPC2UWUeaCdj5FJlrEkeD
QZTzqT9KUNu5PfwgsCzCeAzZavQKXDXq7yAtIBIC8bK2sIGhM+bz7Nbu9fPrtmV0
uk5Enlpi2Y/pUFrRTn27FghZAEgWWUF2Drq0kThEZka3jXveBZ7KaHnfAoGBAPXy
3TVsw0Y34ZljmbsHAyT90ZG7FnA3PDXXHOZxEIDo89m8OTGeBW4eqhLvKa3t+thM
oUGyWTtrjKLELuGa8fiDpKq1b8NJqQYB4V0NJlfOYZ6G8Q+hrT+jXTC4+Lb7kmJq
tyIODlyg4B0GQLbFBZXc4FkwWZXxDT+JwKynh36XAoGBAOWsGhm+3yH755fO5FUH
cLRcPPkV0fmDfYThlpz6RZmENbDlyfSUHDB0Yuw1i6Lfq6dmb9jXdkG3xidx+EZF
hXTQCAitrBZ3IOG1YOrjakIYaacYdrxMaZzw1A0hXFRJEGeN8r6vYzkJrFo0IijS
LC+upy7WQujJAIB7qoMr0UHdAoGAEHTEikuRsUQR6zJ32cS5WCNHf2m2MaHwfGW9
QEn2Ybm0fzAR35kEIf8ZQBUSg9m1e/18mKm3QLuMeGOKA3xbjlY4kVd8d+OY1JcR
nilAFIXxkCrVPEeEEQr8NENcGNoyTDV5tWSdX2NAO5DsiY4bNpDFzhHnHJo5WbP8
2VCIR1cCgYAtcMtavC0nIPxmMEkpd9k+0qWcYclt73wr71sQ+kGOU1/M4g8SZh2z
QmXDkRpJf+/xpaeknf6bj6x2r7FXfVoG5vNdB+Cdn3uepkRHPLSStTIwpPVTsQVy
aTVLTgFnTNMM8whCrfR4eBwHVJIejHiA3cl5Ocq/J6u4kgtFkfwKaQ==
-----END RSA PRIVATE KEY-----
`

// testBadCertBundleText is just simply the LE test intermediate cert
const testBadCertBundleText = `
-----BEGIN CERTIFICATE-----
MIIEqzCCApOgAwIBAgIRAIvhKg5ZRO08VGQx8JdhT+UwDQYJKoZIhvcNAQELBQAw
GjEYMBYGA1UEAwwPRmFrZSBMRSBSb290IFgxMB4XDTE2MDUyMzIyMDc1OVoXDTM2
MDUyMzIyMDc1OVowIjEgMB4GA1UEAwwXRmFrZSBMRSBJbnRlcm1lZGlhdGUgWDEw
ggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDtWKySDn7rWZc5ggjz3ZB0
8jO4xti3uzINfD5sQ7Lj7hzetUT+wQob+iXSZkhnvx+IvdbXF5/yt8aWPpUKnPym
oLxsYiI5gQBLxNDzIec0OIaflWqAr29m7J8+NNtApEN8nZFnf3bhehZW7AxmS1m0
ZnSsdHw0Fw+bgixPg2MQ9k9oefFeqa+7Kqdlz5bbrUYV2volxhDFtnI4Mh8BiWCN
xDH1Hizq+GKCcHsinDZWurCqder/afJBnQs+SBSL6MVApHt+d35zjBD92fO2Je56
dhMfzCgOKXeJ340WhW3TjD1zqLZXeaCyUNRnfOmWZV8nEhtHOFbUCU7r/KkjMZO9
AgMBAAGjgeMwgeAwDgYDVR0PAQH/BAQDAgGGMBIGA1UdEwEB/wQIMAYBAf8CAQAw
HQYDVR0OBBYEFMDMA0a5WCDMXHJw8+EuyyCm9Wg6MHoGCCsGAQUFBwEBBG4wbDA0
BggrBgEFBQcwAYYoaHR0cDovL29jc3Auc3RnLXJvb3QteDEubGV0c2VuY3J5cHQu
b3JnLzA0BggrBgEFBQcwAoYoaHR0cDovL2NlcnQuc3RnLXJvb3QteDEubGV0c2Vu
Y3J5cHQub3JnLzAfBgNVHSMEGDAWgBTBJnSkikSg5vogKNhcI5pFiBh54DANBgkq
hkiG9w0BAQsFAAOCAgEABYSu4Il+fI0MYU42OTmEj+1HqQ5DvyAeyCA6sGuZdwjF
UGeVOv3NnLyfofuUOjEbY5irFCDtnv+0ckukUZN9lz4Q2YjWGUpW4TTu3ieTsaC9
AFvCSgNHJyWSVtWvB5XDxsqawl1KzHzzwr132bF2rtGtazSqVqK9E07sGHMCf+zp
DQVDVVGtqZPHwX3KqUtefE621b8RI6VCl4oD30Olf8pjuzG4JKBFRFclzLRjo/h7
IkkfjZ8wDa7faOjVXx6n+eUQ29cIMCzr8/rNWHS9pYGGQKJiY2xmVC9h12H99Xyf
zWE9vb5zKP3MVG6neX1hSdo7PEAb9fqRhHkqVsqUvJlIRmvXvVKTwNCP3eCjRCCI
PTAvjV+4ni786iXwwFYNz8l3PmPLCyQXWGohnJ8iBm+5nk7O2ynaPVW0U2W+pt2w
SVuvdDM5zGv2f9ltNWUiYZHJ1mmO97jSY/6YfdOUH66iRtQtDkHBRdkNBsMbD+Em
2TgBldtHNSJBfB3pm9FblgOcJ0FSWcUDWJ7vO0+NTXlgrRofRT6pVywzxVo6dND0
WzYlTWeUVsO40xJqhgUQRER9YLOLxJ0O6C8i0xFxAMKOtSdodMB3RIwt7RFQ0uyt
n5Z5MqkYhlMI3J1tPRTp1nEt9fyGspBOO05gi148Qasp+3N+svqKomoQglNoAxU=
-----END CERTIFICATE-----
`

const testRegJSONText = `
{
	"resource": "reg",
	"id": 224403,
	"key": {
			 "kty": "RSA",
			 "use": "sig",
			 "n": "n4EPtAOCc9AlkeQHPzHStgAbgs7bTZLwUBZdR8_KuKPEHLd4rHVTeT-O-XV2jRojdNhxJWTDvNd7nqQ0VEiZQHz_AJmSCpMaJMRBSFKrKb2wqVwGU_NsYOYL-QtiWN2lbzcEe6XC0dApr5ydQLrHqkHHig3RBordaZ6Aj-oBHqFEHYpPe7Tpe-OfVfHd1E6cS6M1FZcD1NNLYD5lFHpPI9bTwJlsde3uhGqC0ZCuEHg8lhzwOHrtIQbS0FVbb9k3-tVTU4fg_3L_vniUFAKwuCLqKnS2BYwdq_mzSnbLY7h_qixoR7jig3__kRhuaxwUkRz5iaiQkqgc5gHdrNP5zw",
			 "e": "AQAB"
	},
	"contact": ["mailto:nobody@example.com"],
	"agreement": "https://letsencrypt.org/documents/LE-SA-v1.0.1-July-27-2015.pdf"
}
`

func testRegData() *acme.RegistrationResource {
	reg := acme.RegistrationResource{}
	body := acme.Registration{}
	err := json.Unmarshal([]byte(testRegJSONText), &body)
	if err != nil {
		panic(fmt.Errorf("Error reading JSON for registration body: %s", err.Error()))
	}
	reg.Body = body
	reg.URI = "https://acme-staging.api.letsencrypt.org/acme/reg/123456789"
	reg.NewAuthzURL = "https://acme-staging.api.letsencrypt.org/acme/new-authz"
	reg.TosURL = "https://letsencrypt.org/documents/LE-SA-v1.0.1-July-27-2015.pdf"
	return &reg
}

func registrationResourceData() *schema.ResourceData {
	r := &schema.Resource{
		Schema: registrationSchemaFull(),
	}
	d := r.TestResourceData()

	d.SetId("regurl")
	d.Set("server_url", "https://acme-staging.api.letsencrypt.org/directory")
	d.Set("account_key_pem", testPrivateKeyText)
	d.Set("email_address", "nobody@example.com")
	d.Set("registration_body", testRegJSONText)
	d.Set("registration_url", "https://acme-staging.api.letsencrypt.org/acme/reg/123456789")
	d.Set("registration_new_authz_url", "https://acme-staging.api.letsencrypt.org/acme/new-authz")
	d.Set("registration_tos_url", "https://letsencrypt.org/documents/LE-SA-v1.0.1-July-27-2015.pdf")

	return d
}

func blankBaseResource() *schema.ResourceData {
	r := &schema.Resource{
		Schema: baseACMESchema(),
	}
	d := r.TestResourceData()
	d.Set("account_key_pem", testPrivateKeyText)
	return d
}

func blankCertificateResource() *schema.ResourceData {
	r := &schema.Resource{
		Schema: certificateSchemaFull(),
	}
	d := r.TestResourceData()
	return d
}

func TestACME_registrationSchemaFull(t *testing.T) {
	m := registrationSchemaFull()
	fields := []string{"email_address", "registration_body", "registration_url", "registration_new_authz_url", "registration_tos_url"}
	for _, v := range fields {
		if _, ok := m[v]; ok == false {
			t.Fatalf("Expected %s to be present", v)
		}
	}
}

func TestACME_certificateSchema(t *testing.T) {
	m := certificateSchemaFull()
	fields := []string{
		"common_name",
		"subject_alternative_names",
		"key_type",
		"certificate_request_pem",
		"min_days_remaining",
		"dns_challenge",
		"http_challenge_port",
		"tls_challenge_port",
		"registration_url",
		"certificate_domain",
		"certificate_url",
		"account_ref",
		"private_key_pem",
		"certificate_pem",
	}
	for _, v := range fields {
		if _, ok := m[v]; ok == false {
			t.Fatalf("Expected %s to be present", v)
		}
	}
}

func TestACME_expandACMEUser(t *testing.T) {
	d := registrationResourceData()
	u, err := expandACMEUser(d)
	if err != nil {
		t.Fatalf("fatal: %s", err.Error())
	}

	if u.GetEmail() != "nobody@example.com" {
		t.Fatalf("Expected email to be nobody@example.com, got %s", u.GetEmail())
	}

	key, err := privateKeyFromPEM([]byte(testPrivateKeyText))
	if err != nil {
		t.Fatalf("fatal: %s", err.Error())
	}

	if reflect.DeepEqual(key, u.GetPrivateKey()) == false {
		t.Fatalf("Expected private key to be %#v, got %#v", key, u.GetPrivateKey())
	}
}

func TestACME_expandACMEUser_badKey(t *testing.T) {
	d := registrationResourceData()
	d.Set("account_key_pem", "bad")
	_, err := expandACMEUser(d)
	if err == nil {
		t.Fatalf("expected error due to bad key")
	}
}

func TestACME_expandACMERegistration(t *testing.T) {
	d := registrationResourceData()
	actual, ok := expandACMERegistration(d)
	if ok == false {
		t.Fatalf("Got error while expanding registration")
	}

	expected := testRegData()
	if reflect.DeepEqual(expected, actual) == false {
		t.Fatalf("Expected %#v, got %#v", expected, actual)
	}
}

func TestACME_expandACMERegistration_noBody(t *testing.T) {
	d := registrationResourceData()
	d.Set("registration_body", "")
	if _, ok := expandACMERegistration(d); ok {
		t.Fatalf("Expected error due to empty body")
	}
}

func TestACME_expandACMERegistration_badBody(t *testing.T) {
	d := registrationResourceData()
	d.Set("registration_body", "foo")
	if _, ok := expandACMERegistration(d); ok {
		t.Fatalf("Expected error due to bad body data")
	}
}

func TestACME_expandACMERegistration_noRegURL(t *testing.T) {
	d := registrationResourceData()
	d.Set("registration_url", "")
	if _, ok := expandACMERegistration(d); ok {
		t.Fatalf("Expected error due to empty reg URL")
	}
}

func TestACME_expandACMERegistration_noNewAuthZ(t *testing.T) {
	d := registrationResourceData()
	d.Set("registration_new_authz_url", "")
	if _, ok := expandACMERegistration(d); ok {
		t.Fatalf("Expected error due to empty new-authz URL")
	}
}

func TestACME_expandACMERegistration_noTOS(t *testing.T) {
	d := registrationResourceData()
	d.Set("registration_tos_url", "")
	if _, ok := expandACMERegistration(d); ok == false {
		t.Fatalf("Blank TOS URL should have been OK")
	}
}

func TestACME_expandACMEClient_badKey(t *testing.T) {
	d := registrationResourceData()
	d.Set("account_key_pem", "bad")
	_, _, err := expandACMEClient(d, "")
	if err == nil {
		t.Fatalf("expected error due to bad key")
	}
}

func TestACME_expandACMEClient_badURL(t *testing.T) {
	d := registrationResourceData()
	d.Set("server_url", "bad://")
	_, _, err := expandACMEClient(d, "")
	if err == nil {
		t.Fatalf("expected error due to bad URL")
	}
}

func TestACME_expandACMEClient_badRegURL(t *testing.T) {
	d := registrationResourceData()
	_, _, err := expandACMEClient(d, "bad://")
	if err == nil {
		t.Fatalf("expected error due to bad reg URL")
	}
}

func TestACME_expandACMEClient_noCertData(t *testing.T) {
	c := acme.CertificateResource{}
	_, err := certDaysRemaining(c)
	if err == nil {
		t.Fatalf("expected error due to bad cert data")
	}
}

func TestACME_parsePEMBundle_noData(t *testing.T) {
	b := []byte{}
	_, err := parsePEMBundle(b)
	if err == nil {
		t.Fatalf("expected error due to no PEM data")
	}
}

func TestACME_setDNSChallenge_noProvider(t *testing.T) {
	m := make(map[string]interface{})
	d := blankBaseResource()
	ts := httpDirTestServer()
	d.Set("server_url", ts.URL)
	client, _, err := expandACMEClient(d, "")
	if err != nil {
		t.Fatalf("fatal: %s", err.Error())
	}

	err = setDNSChallenge(client, m)
	if err == nil {
		t.Fatalf("should have errored due to no provider supplied")
	}
}

func TestACME_setDNSChallenge_unsuppotedProvider(t *testing.T) {
	m := map[string]interface{}{
		"provider": "foo",
	}
	d := blankBaseResource()
	ts := httpDirTestServer()
	d.Set("server_url", ts.URL)
	client, _, err := expandACMEClient(d, "")
	if err != nil {
		t.Fatalf("fatal: %s", err.Error())
	}

	err = setDNSChallenge(client, m)
	if err == nil {
		t.Fatalf("should have errored due to unknown provider")
	}
}

func TestACME_saveCertificateResource_badCert(t *testing.T) {
	b := testBadCertBundleText
	c := acme.CertificateResource{
		Certificate: []byte(b),
	}
	d := blankCertificateResource()
	err := saveCertificateResource(d, c)
	if err == nil {
		t.Fatalf("expected error due to bad cert data")
	}
}

func TestACME_certDaysRemaining_CACert(t *testing.T) {
	b := testBadCertBundleText
	c := acme.CertificateResource{
		Certificate: []byte(b),
	}
	_, err := certDaysRemaining(c)
	if err == nil {
		t.Fatalf("expected error due to cert being a CA")
	}
}

func TestACME_splitPEMBundle_noData(t *testing.T) {
	b := []byte{}
	_, _, err := splitPEMBundle(b)
	if err == nil {
		t.Fatalf("expected error due to no PEM data")
	}
}

func TestACME_splitPEMBundle_CAFirst(t *testing.T) {
	b := testBadCertBundleText + testBadCertBundleText
	_, _, err := splitPEMBundle([]byte(b))
	if err == nil {
		t.Fatalf("expected error due to CA cert being first")
	}
}

func TestACME_splitPEMBundle_singleCert(t *testing.T) {
	b := testBadCertBundleText
	_, _, err := splitPEMBundle([]byte(b))
	if err == nil {
		t.Fatalf("expected error due to only one cert being present")
	}
}

func TestACME_validateKeyType(t *testing.T) {
	s := "2048"

	_, errs := validateKeyType(s, "key_type")
	if len(errs) > 0 {
		t.Fatalf("bad: %#v", errs)
	}
}

func TestACME_validateKeyType_invalid(t *testing.T) {
	s := "512"

	_, errs := validateKeyType(s, "key_type")
	if len(errs) < 1 {
		t.Fatalf("should have given an error")
	}
}

func TestACME_validateDNSChallengeConfig(t *testing.T) {
	m := map[string]interface{}{
		"AWS_FOO": "bar",
	}

	_, errs := validateDNSChallengeConfig(m, "config")
	if len(errs) > 0 {
		t.Fatalf("bad: %#v", errs)
	}
}

func TestACME_validateDNSChallengeConfig_invalid(t *testing.T) {
	s := map[string]interface{}{
		"AWS_FOO": 1,
	}

	_, errs := validateDNSChallengeConfig(s, "config")
	if len(errs) < 1 {
		t.Fatalf("should have given an error")
	}
}
