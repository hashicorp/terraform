package authentication

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/crypto/pkcs12"
)

type servicePrincipalClientCertificateAuth struct {
	clientId           string
	clientCertPath     string
	clientCertPassword string
	subscriptionId     string
	tenantId           string
}

func (a servicePrincipalClientCertificateAuth) build(b Builder) (authMethod, error) {
	method := servicePrincipalClientCertificateAuth{
		clientId:           b.ClientID,
		clientCertPath:     b.ClientCertPath,
		clientCertPassword: b.ClientCertPassword,
		subscriptionId:     b.SubscriptionID,
		tenantId:           b.TenantID,
	}
	return method, nil
}

func (a servicePrincipalClientCertificateAuth) isApplicable(b Builder) bool {
	return b.SupportsClientCertAuth && b.ClientCertPath != ""
}

func (a servicePrincipalClientCertificateAuth) name() string {
	return "Service Principal / Client Certificate"
}

func (a servicePrincipalClientCertificateAuth) getAuthorizationToken(sender autorest.Sender, oauthConfig *adal.OAuthConfig, endpoint string) (*autorest.BearerAuthorizer, error) {
	certificateData, err := ioutil.ReadFile(a.clientCertPath)
	if err != nil {
		return nil, fmt.Errorf("Error reading Client Certificate %q: %v", a.clientCertPath, err)
	}

	// Get the certificate and private key from pfx file
	certificate, rsaPrivateKey, err := decodePkcs12(certificateData, a.clientCertPassword)
	if err != nil {
		return nil, fmt.Errorf("Error decoding pkcs12 certificate: %v", err)
	}

	spt, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, a.clientId, certificate, rsaPrivateKey, endpoint)
	if err != nil {
		return nil, err
	}

	spt.SetSender(sender)

	err = spt.Refresh()
	if err != nil {
		return nil, err
	}

	auth := autorest.NewBearerAuthorizer(spt)
	return auth, nil
}

func (a servicePrincipalClientCertificateAuth) populateConfig(c *Config) error {
	c.AuthenticatedAsAServicePrincipal = true
	return nil
}

func (a servicePrincipalClientCertificateAuth) validate() error {
	var err *multierror.Error

	fmtErrorMessage := "A %s must be configured when authenticating as a Service Principal using a Client Certificate."

	if a.subscriptionId == "" {
		err = multierror.Append(err, fmt.Errorf(fmtErrorMessage, "Subscription ID"))
	}

	if a.clientId == "" {
		err = multierror.Append(err, fmt.Errorf(fmtErrorMessage, "Client ID"))
	}

	if a.clientCertPath == "" {
		err = multierror.Append(err, fmt.Errorf(fmtErrorMessage, "Client Certificate Path"))
	} else {
		if strings.HasSuffix(strings.ToLower(a.clientCertPath), ".pfx") {
			// ensure it exists on disk
			_, fileErr := os.Stat(a.clientCertPath)
			if os.IsNotExist(fileErr) {
				err = multierror.Append(err, fmt.Errorf("Error locating Client Certificate specified at %q: %s", a.clientCertPath, fileErr))
			}

			// we're intentionally /not/ checking it's an actual PFX file at this point, as that happens in the getAuthorizationToken
		} else {
			err = multierror.Append(err, fmt.Errorf("The Client Certificate Path is not a *.pfx file: %q", a.clientCertPath))
		}
	}

	if a.tenantId == "" {
		err = multierror.Append(err, fmt.Errorf(fmtErrorMessage, "Tenant ID"))
	}

	return err.ErrorOrNil()
}

func decodePkcs12(pkcs []byte, password string) (*x509.Certificate, *rsa.PrivateKey, error) {
	privateKey, certificate, err := pkcs12.Decode(pkcs, password)
	if err != nil {
		return nil, nil, err
	}

	rsaPrivateKey, isRsaKey := privateKey.(*rsa.PrivateKey)
	if !isRsaKey {
		return nil, nil, fmt.Errorf("PKCS#12 certificate must contain an RSA private key")
	}

	return certificate, rsaPrivateKey, nil
}
