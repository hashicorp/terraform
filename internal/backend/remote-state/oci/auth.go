// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import (
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/common/auth"
	"github.com/zclconf/go-cty/cty"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

var ApiKeyConfigAttributes = [5]string{UserOcidAttrName, FingerprintAttrName, PrivateKeyAttrName, PrivateKeyPathAttrName, PrivateKeyPasswordAttrName}

type ociAuthConfigProvider struct {
	authType           string
	configFileProfile  string
	region             string
	tenancyOcid        string
	userOcid           string
	fingerprint        string
	privateKey         string
	privateKeyPath     string
	privateKeyPassword string
}

func newOciAuthConfigProvider(obj cty.Value) ociAuthConfigProvider {
	p := ociAuthConfigProvider{}

	if authVal, ok := getBackendAttrWithDefault(obj, AuthAttrName, AuthAPIKeySetting); ok {
		p.authType = authVal.AsString()
	}

	if configFileProfileVal, ok := getBackendAttr(obj, ConfigFileProfileAttrName); ok {
		p.configFileProfile = configFileProfileVal.AsString()
	}
	if regionVal, ok := getBackendAttr(obj, RegionAttrName); ok {
		p.region = regionVal.AsString()
	}

	if tenancyOcidVal, ok := getBackendAttr(obj, TenancyOcidAttrName); ok {
		p.tenancyOcid = tenancyOcidVal.AsString()
	}

	if userOcidVal, ok := getBackendAttr(obj, UserOcidAttrName); ok {
		p.userOcid = userOcidVal.AsString()
	}

	if fingerprintVal, ok := getBackendAttr(obj, FingerprintAttrName); ok {
		p.fingerprint = fingerprintVal.AsString()
	}

	if privateKeyVal, ok := getBackendAttr(obj, PrivateKeyAttrName); ok {
		p.privateKey = privateKeyVal.AsString()
	}

	if privateKeyPathVal, ok := getBackendAttr(obj, PrivateKeyPathAttrName); ok {
		p.privateKeyPath = privateKeyPathVal.AsString()
	}

	if privateKeyPasswordVal, ok := getBackendAttr(obj, PrivateKeyPasswordAttrName); ok {
		p.privateKeyPassword = privateKeyPasswordVal.AsString()
	}

	return p
}
func (p ociAuthConfigProvider) AuthType() (common.AuthConfig, error) {
	return common.AuthConfig{
			AuthType:         common.UnknownAuthenticationType,
			IsFromConfigFile: false,
			OboToken:         nil,
		},
		fmt.Errorf("unsupported, keep the interface")
}

func (p ociAuthConfigProvider) TenancyOCID() (string, error) {
	if p.tenancyOcid != "" {
		return p.tenancyOcid, nil
	}
	return "", fmt.Errorf("can not get %s from Terraform backend configuration", TenancyOcidAttrName)
}

func (p ociAuthConfigProvider) UserOCID() (string, error) {
	if p.userOcid != "" {
		return p.userOcid, nil
	}
	return "", fmt.Errorf("can not get %s from Terraform backend configuration", UserOcidAttrName)
}

func (p ociAuthConfigProvider) KeyFingerprint() (string, error) {
	if p.fingerprint != "" {
		return p.fingerprint, nil
	}
	return "", fmt.Errorf("can not get %s from Terraform backend configuration", FingerprintAttrName)
}

func (p ociAuthConfigProvider) Region() (string, error) {
	if p.region != "" {
		return p.region, nil
	}
	return "", fmt.Errorf("can not get %s from Terraform backend configuration", RegionAttrName)
}
func (p ociAuthConfigProvider) KeyID() (string, error) {
	tenancy, err := p.TenancyOCID()
	if err != nil {
		return "", err
	}

	user, err := p.UserOCID()
	if err != nil {
		return "", err
	}

	fingerprint, err := p.KeyFingerprint()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%s", tenancy, user, fingerprint), nil
}

func (p ociAuthConfigProvider) PrivateRSAKey() (key *rsa.PrivateKey, err error) {

	if p.privateKey != "" {
		keyData := strings.ReplaceAll(p.privateKey, "\\n", "\n") // Ensure \n is replaced by actual newlines
		return common.PrivateKeyFromBytesWithPassword([]byte(keyData), []byte(p.privateKeyPassword))
	}

	if p.privateKeyPath != "" {
		resolvedPath := expandPath(p.privateKeyPath)
		pemFileContent, readFileErr := os.ReadFile(resolvedPath)
		if readFileErr != nil {
			return nil, fmt.Errorf("can not read private key from: '%s', Error: %q", p.privateKeyPath, readFileErr)
		}
		return common.PrivateKeyFromBytesWithPassword(pemFileContent, []byte(p.privateKeyPassword))
	}

	return nil, fmt.Errorf("can not get private_key or private_key_path from Terraform configuration")
}

func (p ociAuthConfigProvider) getConfigProviders() ([]common.ConfigurationProvider, error) {
	var configProviders []common.ConfigurationProvider
	logger.Debug(fmt.Sprintf("Using %s authentication", p.authType))
	switch strings.ToLower(p.authType) {
	case strings.ToLower(AuthAPIKeySetting):
		// No additional config providers needed
	case strings.ToLower(AuthInstancePrincipalSetting):

		logger.Debug("Attempting to authenticate using instance principal credentials")
		if p.region == "" {
			return nil, fmt.Errorf("unable to determine region from Terraform backend configuration while using Instance Principal")
		}

		// Used to modify InstancePrincipal auth clients so that `accept_local_certs` is honored for auth clients as well
		instancePrincipalAuthClientModifier := func(client common.HTTPRequestDispatcher) (common.HTTPRequestDispatcher, error) {
			if acceptLocalCerts := getEnvSettingWithBlankDefault(AcceptLocalCerts); acceptLocalCerts != "" {
				if value, err := strconv.ParseBool(acceptLocalCerts); err == nil {
					modifiedClient := buildHttpClient()
					modifiedClient.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify = value
					return modifiedClient, nil
				}
			}
			return client, nil
		}

		cfg, err := auth.InstancePrincipalConfigurationForRegionWithCustomClient(common.StringToRegion(p.region), instancePrincipalAuthClientModifier)
		if err != nil {
			return nil, err
		}
		logger.Debug(" Configuration provided by: %s", cfg)

		configProviders = append(configProviders, cfg)
	case strings.ToLower(AuthInstancePrincipalWithCertsSetting):
		logger.Debug("Attempting to authenticate using instance principal with certificates")

		if p.region == "" {
			return nil, fmt.Errorf("unable to determine region from Terraform backend configuration while using Instance Principal with certificates")
		}

		defaultCertsDir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("can not get working directory for current os platform")
		}

		certsDir := filepath.Clean(getEnvSettingWithDefault("test_certificates_location", defaultCertsDir))
		leafCertificateBytes, err := getCertificateFileBytes(filepath.Join(certsDir, "ip_cert.pem"))
		if err != nil {
			return nil, fmt.Errorf("can not read leaf certificate from %s", filepath.Join(certsDir, "ip_cert.pem"))
		}

		leafPrivateKeyBytes, err := getCertificateFileBytes(filepath.Join(certsDir, "ip_key.pem"))
		if err != nil {
			return nil, fmt.Errorf("can not read leaf private key from %s", filepath.Join(certsDir, "ip_key.pem"))
		}

		leafPassphraseBytes := []byte{}
		if _, err := os.Stat(certsDir + "/leaf_passphrase"); !os.IsNotExist(err) {
			leafPassphraseBytes, err = getCertificateFileBytes(filepath.Join(certsDir + "leaf_passphrase"))
			if err != nil {
				return nil, fmt.Errorf("can not read leafPassphraseBytes from %s", filepath.Join(certsDir+"leaf_passphrase"))
			}
		}

		intermediateCertificateBytes, err := getCertificateFileBytes(filepath.Join(certsDir, "intermediate.pem"))
		if err != nil {
			return nil, fmt.Errorf("can not read intermediate certificate from %s", filepath.Join(certsDir, "intermediate.pem"))
		}

		intermediateCertificatesBytes := [][]byte{
			intermediateCertificateBytes,
		}

		cfg, err := auth.InstancePrincipalConfigurationWithCerts(common.StringToRegion(p.region), leafCertificateBytes, leafPassphraseBytes, leafPrivateKeyBytes, intermediateCertificatesBytes)
		if err != nil {
			return nil, err
		}
		logger.Debug(" Configuration provided by: %s", cfg)

		configProviders = append(configProviders, cfg)

	case strings.ToLower(AuthSecurityToken):

		if p.region == "" {
			return nil, fmt.Errorf("can not get %s from Terraform configuration (SecurityToken)", RegionAttrName)
		}
		// if region is part of the provider block make sure it is part of the final configuration too, and overwrites the region in the profile. +
		regionProvider := common.NewRawConfigurationProvider("", "", p.region, "", "", nil)
		configProviders = append(configProviders, regionProvider)

		if p.configFileProfile == "" {
			return nil, fmt.Errorf("missing profile in provider block %v", ConfigFileProfileAttrName)
		}

		defaultPath := path.Join(getHomeFolder(), DefaultConfigDirName, DefaultConfigFileName)
		if err := checkProfile(p.configFileProfile, defaultPath); err != nil {
			return nil, err
		}
		securityTokenBasedAuthConfigProvider, err := common.ConfigurationProviderForSessionTokenWithProfile(defaultPath, p.configFileProfile, p.privateKeyPassword)
		if err != nil {
			return nil, fmt.Errorf("could not create security token based auth config provider %v", err)
		}
		configProviders = append(configProviders, securityTokenBasedAuthConfigProvider)
	case strings.ToLower(ResourcePrincipal):
		var err error
		var resourcePrincipalAuthConfigProvider auth.ConfigurationProviderWithClaimAccess

		if p.region == "" {
			logger.Debug("did not get %s from Terraform configuration (ResourcePrincipal), falling back to environment variable", RegionAttrName)
			resourcePrincipalAuthConfigProvider, err = auth.ResourcePrincipalConfigurationProvider()
		} else {
			resourcePrincipalAuthConfigProvider, err = auth.ResourcePrincipalConfigurationProviderForRegion(common.StringToRegion(p.region))
		}
		if err != nil {
			return nil, err
		}
		configProviders = append(configProviders, resourcePrincipalAuthConfigProvider)
	case strings.ToLower(AuthOKEWorkloadIdentity):
		okeWorkloadIdentityConfigProvider, err := auth.OkeWorkloadIdentityConfigurationProvider()
		if err != nil {
			return nil, fmt.Errorf("can not get oke workload indentity based auth config provider %v", err)
		}
		configProviders = append(configProviders, okeWorkloadIdentityConfigProvider)
	default:
		return nil, fmt.Errorf("auth must be one of '%s' or '%s' or '%s' or '%s' or '%s' or '%s'", AuthAPIKeySetting, AuthInstancePrincipalSetting, AuthInstancePrincipalWithCertsSetting, AuthSecurityToken, ResourcePrincipal, AuthOKEWorkloadIdentity)
	}

	return configProviders, nil
}
func (p ociAuthConfigProvider) getSdkConfigProvider() (common.ConfigurationProvider, error) {

	configProviders, err := p.getConfigProviders()
	if err != nil {
		return nil, err
	}

	configProviders = append(configProviders, p)
	//In GoSDK, the first step is to check if AuthType exists,
	//for composite provider, we only check the first provider in the list for the AuthType.
	//Then SDK will based on the AuthType to Create the actual provider if it's a valid value.
	//If not, then SDK will base on the order in the composite provider list to check for necessary info (tenancyid, userID, fingerprint, region, keyID).
	if p.configFileProfile == "" {
		configProviders = append(configProviders, common.DefaultConfigProvider())
	} else {
		defaultPath := path.Join(getHomeFolder(), DefaultConfigDirName, DefaultConfigFileName)
		err := checkProfile(p.configFileProfile, defaultPath)
		if err != nil {
			return nil, err
		}
		configProviders = append(configProviders, common.CustomProfileConfigProvider(defaultPath, p.configFileProfile))
	}
	sdkConfigProvider, err := common.ComposingConfigurationProvider(configProviders)
	if err != nil {
		return nil, err
	}

	return sdkConfigProvider, nil
}
func buildHttpClient() (httpClient *http.Client) {
	httpClient = &http.Client{
		Timeout: getDurationFromEnvVar(HTTPRequestTimeOut, DefaultRequestTimeout),
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: getDurationFromEnvVar(DialContextConnectionTimeout, DefaultConnectionTimeout),
			}).DialContext,
			TLSHandshakeTimeout: getDurationFromEnvVar(TLSHandshakeTimeout, DefaultTLSHandshakeTimeout),
			TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
			Proxy:               http.ProxyFromEnvironment,
		},
	}
	return
}

func getCertificateFileBytes(certificateFileFullPath string) (pemRaw []byte, err error) {
	absFile, err := filepath.Abs(certificateFileFullPath)
	if err != nil {
		return nil, fmt.Errorf("can't form absolute path of %s: %v", certificateFileFullPath, err)
	}

	if pemRaw, err = os.ReadFile(absFile); err != nil {
		return nil, fmt.Errorf("can't read %s: %v", certificateFileFullPath, err)
	}
	return
}
