package clientconfig

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"

	yaml "gopkg.in/yaml.v2"
)

// AuthType respresents a valid method of authentication.
type AuthType string

const (
	// AuthPassword defines an unknown version of the password
	AuthPassword AuthType = "password"
	// AuthToken defined an unknown version of the token
	AuthToken AuthType = "token"

	// AuthV2Password defines version 2 of the password
	AuthV2Password AuthType = "v2password"
	// AuthV2Token defines version 2 of the token
	AuthV2Token AuthType = "v2token"

	// AuthV3Password defines version 3 of the password
	AuthV3Password AuthType = "v3password"
	// AuthV3Token defines version 3 of the token
	AuthV3Token AuthType = "v3token"

	// AuthV3ApplicationCredential defines version 3 of the application credential
	AuthV3ApplicationCredential AuthType = "v3applicationcredential"
)

// ClientOpts represents options to customize the way a client is
// configured.
type ClientOpts struct {
	// Cloud is the cloud entry in clouds.yaml to use.
	Cloud string

	// EnvPrefix allows a custom environment variable prefix to be used.
	EnvPrefix string

	// AuthType specifies the type of authentication to use.
	// By default, this is "password".
	AuthType AuthType

	// AuthInfo defines the authentication information needed to
	// authenticate to a cloud when clouds.yaml isn't used.
	AuthInfo *AuthInfo

	// RegionName is the region to create a Service Client in.
	// This will override a region in clouds.yaml or can be used
	// when authenticating directly with AuthInfo.
	RegionName string
}

// LoadCloudsYAML will load a clouds.yaml file and return the full config.
func LoadCloudsYAML() (map[string]Cloud, error) {
	content, err := findAndReadCloudsYAML()
	if err != nil {
		return nil, err
	}

	var clouds Clouds
	err = yaml.Unmarshal(content, &clouds)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %v", err)
	}

	return clouds.Clouds, nil
}

// LoadSecureCloudsYAML will load a secure.yaml file and return the full config.
func LoadSecureCloudsYAML() (map[string]Cloud, error) {
	var secureClouds Clouds

	content, err := findAndReadSecureCloudsYAML()
	if err != nil {
		if err.Error() == "no secure.yaml file found" {
			// secure.yaml is optional so just ignore read error
			return secureClouds.Clouds, nil
		}
		return nil, err
	}

	err = yaml.Unmarshal(content, &secureClouds)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %v", err)
	}

	return secureClouds.Clouds, nil
}

// LoadPublicCloudsYAML will load a public-clouds.yaml file and return the full config.
func LoadPublicCloudsYAML() (map[string]Cloud, error) {
	var publicClouds PublicClouds

	content, err := findAndReadPublicCloudsYAML()
	if err != nil {
		if err.Error() == "no clouds-public.yaml file found" {
			// clouds-public.yaml is optional so just ignore read error
			return publicClouds.Clouds, nil
		}

		return nil, err
	}

	err = yaml.Unmarshal(content, &publicClouds)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %v", err)
	}

	return publicClouds.Clouds, nil
}

// GetCloudFromYAML will return a cloud entry from a clouds.yaml file.
func GetCloudFromYAML(opts *ClientOpts) (*Cloud, error) {
	clouds, err := LoadCloudsYAML()
	if err != nil {
		return nil, fmt.Errorf("unable to load clouds.yaml: %s", err)
	}

	// Determine which cloud to use.
	// First see if a cloud name was explicitly set in opts.
	var cloudName string
	if opts != nil && opts.Cloud != "" {
		cloudName = opts.Cloud
	}

	// Next see if a cloud name was specified as an environment variable.
	// This is supposed to override an explicit opts setting.
	envPrefix := "OS_"
	if opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if v := os.Getenv(envPrefix + "CLOUD"); v != "" {
		cloudName = v
	}

	var cloud *Cloud
	if cloudName != "" {
		v, ok := clouds[cloudName]
		if !ok {
			return nil, fmt.Errorf("cloud %s does not exist in clouds.yaml", cloudName)
		}
		cloud = &v
	}

	// If a cloud was not specified, and clouds only contains
	// a single entry, use that entry.
	if cloudName == "" && len(clouds) == 1 {
		for _, v := range clouds {
			cloud = &v
		}
	}

	var cloudIsInCloudsYaml bool
	if cloud == nil {
		// not an immediate error as it might still be defined in secure.yaml
		cloudIsInCloudsYaml = false
	} else {
		cloudIsInCloudsYaml = true
	}

	publicClouds, err := LoadPublicCloudsYAML()
	if err != nil {
		return nil, fmt.Errorf("unable to load clouds-public.yaml: %s", err)
	}

	var profileName = defaultIfEmpty(cloud.Profile, cloud.Cloud)
	if profileName != "" {
		publicCloud, ok := publicClouds[profileName]
		if !ok {
			return nil, fmt.Errorf("cloud %s does not exist in clouds-public.yaml", profileName)
		}
		cloud, err = mergeClouds(cloud, publicCloud)
		if err != nil {
			return nil, fmt.Errorf("Could not merge information from clouds.yaml and clouds-public.yaml for cloud %s", profileName)
		}
	}

	secureClouds, err := LoadSecureCloudsYAML()
	if err != nil {
		return nil, fmt.Errorf("unable to load secure.yaml: %s", err)
	}

	if secureClouds != nil {
		// If no entry was found in clouds.yaml, no cloud name was specified,
		// and only one secureCloud entry exists, use that as the cloud entry.
		if !cloudIsInCloudsYaml && cloudName == "" && len(secureClouds) == 1 {
			for _, v := range secureClouds {
				cloud = &v
			}
		}

		secureCloud, ok := secureClouds[cloudName]
		if !ok && cloud == nil {
			// cloud == nil serves two purposes here:
			// if no entry in clouds.yaml was found and
			// if a single-entry secureCloud wasn't used.
			// At this point, no entry could be determined at all.
			return nil, fmt.Errorf("Could not find cloud %s", cloudName)
		}

		// If secureCloud has content and it differs from the cloud entry,
		// merge the two together.
		if !reflect.DeepEqual((Cloud{}), secureCloud) && !reflect.DeepEqual(cloud, secureCloud) {
			cloud, err = mergeClouds(secureCloud, cloud)
			if err != nil {
				return nil, fmt.Errorf("unable to merge information from clouds.yaml and secure.yaml")
			}
		}
	}

	// Default is to verify SSL API requests
	if cloud.Verify == nil {
		iTrue := true
		cloud.Verify = &iTrue
	}

	// TODO: this is where reading vendor files should go be considered when not found in
	// clouds-public.yml
	// https://github.com/openstack/openstacksdk/tree/master/openstack/config/vendors

	return cloud, nil
}

// AuthOptions creates a gophercloud.AuthOptions structure with the
// settings found in a specific cloud entry of a clouds.yaml file or
// based on authentication settings given in ClientOpts.
//
// This attempts to be a single point of entry for all OpenStack authentication.
//
// See http://docs.openstack.org/developer/os-client-config and
// https://github.com/openstack/os-client-config/blob/master/os_client_config/config.py.
func AuthOptions(opts *ClientOpts) (*gophercloud.AuthOptions, error) {
	cloud := new(Cloud)

	// If no opts were passed in, create an empty ClientOpts.
	if opts == nil {
		opts = new(ClientOpts)
	}

	// Determine if a clouds.yaml entry should be retrieved.
	// Start by figuring out the cloud name.
	// First check if one was explicitly specified in opts.
	var cloudName string
	if opts.Cloud != "" {
		cloudName = opts.Cloud
	}

	// Next see if a cloud name was specified as an environment variable.
	envPrefix := "OS_"
	if opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if v := os.Getenv(envPrefix + "CLOUD"); v != "" {
		cloudName = v
	}

	// If a cloud name was determined, try to look it up in clouds.yaml.
	if cloudName != "" {
		// Get the requested cloud.
		var err error
		cloud, err = GetCloudFromYAML(opts)
		if err != nil {
			return nil, err
		}
	}

	// If cloud.AuthInfo is nil, then no cloud was specified.
	if cloud.AuthInfo == nil {
		// If opts.Auth is not nil, then try using the auth settings from it.
		if opts.AuthInfo != nil {
			cloud.AuthInfo = opts.AuthInfo
		}

		// If cloud.AuthInfo is still nil, then set it to an empty Auth struct
		// and rely on environment variables to do the authentication.
		if cloud.AuthInfo == nil {
			cloud.AuthInfo = new(AuthInfo)
		}
	}

	identityAPI := determineIdentityAPI(cloud, opts)
	switch identityAPI {
	case "2.0", "2":
		return v2auth(cloud, opts)
	case "3":
		return v3auth(cloud, opts)
	}

	return nil, fmt.Errorf("Unable to build AuthOptions")
}

func determineIdentityAPI(cloud *Cloud, opts *ClientOpts) string {
	var identityAPI string
	if cloud.IdentityAPIVersion != "" {
		identityAPI = cloud.IdentityAPIVersion
	}

	envPrefix := "OS_"
	if opts != nil && opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if v := os.Getenv(envPrefix + "IDENTITY_API_VERSION"); v != "" {
		identityAPI = v
	}

	if identityAPI == "" {
		if cloud.AuthInfo != nil {
			if strings.Contains(cloud.AuthInfo.AuthURL, "v2.0") {
				identityAPI = "2.0"
			}

			if strings.Contains(cloud.AuthInfo.AuthURL, "v3") {
				identityAPI = "3"
			}
		}
	}

	if identityAPI == "" {
		switch cloud.AuthType {
		case AuthV2Password:
			identityAPI = "2.0"
		case AuthV2Token:
			identityAPI = "2.0"
		case AuthV3Password:
			identityAPI = "3"
		case AuthV3Token:
			identityAPI = "3"
		case AuthV3ApplicationCredential:
			identityAPI = "3"
		}
	}

	// If an Identity API version could not be determined,
	// default to v3.
	if identityAPI == "" {
		identityAPI = "3"
	}

	return identityAPI
}

// v2auth creates a v2-compatible gophercloud.AuthOptions struct.
func v2auth(cloud *Cloud, opts *ClientOpts) (*gophercloud.AuthOptions, error) {
	// Environment variable overrides.
	envPrefix := "OS_"
	if opts != nil && opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if cloud.AuthInfo.AuthURL == "" {
		if v := os.Getenv(envPrefix + "AUTH_URL"); v != "" {
			cloud.AuthInfo.AuthURL = v
		}
	}

	if cloud.AuthInfo.Token == "" {
		if v := os.Getenv(envPrefix + "TOKEN"); v != "" {
			cloud.AuthInfo.Token = v
		}

		if v := os.Getenv(envPrefix + "AUTH_TOKEN"); v != "" {
			cloud.AuthInfo.Token = v
		}
	}

	if cloud.AuthInfo.Username == "" {
		if v := os.Getenv(envPrefix + "USERNAME"); v != "" {
			cloud.AuthInfo.Username = v
		}
	}

	if cloud.AuthInfo.Password == "" {
		if v := os.Getenv(envPrefix + "PASSWORD"); v != "" {
			cloud.AuthInfo.Password = v
		}
	}

	if cloud.AuthInfo.ProjectID == "" {
		if v := os.Getenv(envPrefix + "TENANT_ID"); v != "" {
			cloud.AuthInfo.ProjectID = v
		}

		if v := os.Getenv(envPrefix + "PROJECT_ID"); v != "" {
			cloud.AuthInfo.ProjectID = v
		}
	}

	if cloud.AuthInfo.ProjectName == "" {
		if v := os.Getenv(envPrefix + "TENANT_NAME"); v != "" {
			cloud.AuthInfo.ProjectName = v
		}

		if v := os.Getenv(envPrefix + "PROJECT_NAME"); v != "" {
			cloud.AuthInfo.ProjectName = v
		}
	}

	ao := &gophercloud.AuthOptions{
		IdentityEndpoint: cloud.AuthInfo.AuthURL,
		TokenID:          cloud.AuthInfo.Token,
		Username:         cloud.AuthInfo.Username,
		Password:         cloud.AuthInfo.Password,
		TenantID:         cloud.AuthInfo.ProjectID,
		TenantName:       cloud.AuthInfo.ProjectName,
	}

	return ao, nil
}

// v3auth creates a v3-compatible gophercloud.AuthOptions struct.
func v3auth(cloud *Cloud, opts *ClientOpts) (*gophercloud.AuthOptions, error) {
	// Environment variable overrides.
	envPrefix := "OS_"
	if opts != nil && opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if cloud.AuthInfo.AuthURL == "" {
		if v := os.Getenv(envPrefix + "AUTH_URL"); v != "" {
			cloud.AuthInfo.AuthURL = v
		}
	}

	if cloud.AuthInfo.Token == "" {
		if v := os.Getenv(envPrefix + "TOKEN"); v != "" {
			cloud.AuthInfo.Token = v
		}

		if v := os.Getenv(envPrefix + "AUTH_TOKEN"); v != "" {
			cloud.AuthInfo.Token = v
		}
	}

	if cloud.AuthInfo.Username == "" {
		if v := os.Getenv(envPrefix + "USERNAME"); v != "" {
			cloud.AuthInfo.Username = v
		}
	}

	if cloud.AuthInfo.UserID == "" {
		if v := os.Getenv(envPrefix + "USER_ID"); v != "" {
			cloud.AuthInfo.UserID = v
		}
	}

	if cloud.AuthInfo.Password == "" {
		if v := os.Getenv(envPrefix + "PASSWORD"); v != "" {
			cloud.AuthInfo.Password = v
		}
	}

	if cloud.AuthInfo.ProjectID == "" {
		if v := os.Getenv(envPrefix + "TENANT_ID"); v != "" {
			cloud.AuthInfo.ProjectID = v
		}

		if v := os.Getenv(envPrefix + "PROJECT_ID"); v != "" {
			cloud.AuthInfo.ProjectID = v
		}
	}

	if cloud.AuthInfo.ProjectName == "" {
		if v := os.Getenv(envPrefix + "TENANT_NAME"); v != "" {
			cloud.AuthInfo.ProjectName = v
		}

		if v := os.Getenv(envPrefix + "PROJECT_NAME"); v != "" {
			cloud.AuthInfo.ProjectName = v
		}
	}

	if cloud.AuthInfo.DomainID == "" {
		if v := os.Getenv(envPrefix + "DOMAIN_ID"); v != "" {
			cloud.AuthInfo.DomainID = v
		}
	}

	if cloud.AuthInfo.DomainName == "" {
		if v := os.Getenv(envPrefix + "DOMAIN_NAME"); v != "" {
			cloud.AuthInfo.DomainName = v
		}
	}

	if cloud.AuthInfo.DefaultDomain == "" {
		if v := os.Getenv(envPrefix + "DEFAULT_DOMAIN"); v != "" {
			cloud.AuthInfo.DefaultDomain = v
		}
	}

	if cloud.AuthInfo.ProjectDomainID == "" {
		if v := os.Getenv(envPrefix + "PROJECT_DOMAIN_ID"); v != "" {
			cloud.AuthInfo.ProjectDomainID = v
		}
	}

	if cloud.AuthInfo.ProjectDomainName == "" {
		if v := os.Getenv(envPrefix + "PROJECT_DOMAIN_NAME"); v != "" {
			cloud.AuthInfo.ProjectDomainName = v
		}
	}

	if cloud.AuthInfo.UserDomainID == "" {
		if v := os.Getenv(envPrefix + "USER_DOMAIN_ID"); v != "" {
			cloud.AuthInfo.UserDomainID = v
		}
	}

	if cloud.AuthInfo.UserDomainName == "" {
		if v := os.Getenv(envPrefix + "USER_DOMAIN_NAME"); v != "" {
			cloud.AuthInfo.UserDomainName = v
		}
	}

	if cloud.AuthInfo.ApplicationCredentialID == "" {
		if v := os.Getenv(envPrefix + "APPLICATION_CREDENTIAL_ID"); v != "" {
			cloud.AuthInfo.ApplicationCredentialID = v
		}
	}

	if cloud.AuthInfo.ApplicationCredentialName == "" {
		if v := os.Getenv(envPrefix + "APPLICATION_CREDENTIAL_NAME"); v != "" {
			cloud.AuthInfo.ApplicationCredentialName = v
		}
	}

	if cloud.AuthInfo.ApplicationCredentialSecret == "" {
		if v := os.Getenv(envPrefix + "APPLICATION_CREDENTIAL_SECRET"); v != "" {
			cloud.AuthInfo.ApplicationCredentialSecret = v
		}
	}

	// Build a scope and try to do it correctly.
	// https://github.com/openstack/os-client-config/blob/master/os_client_config/config.py#L595
	scope := new(gophercloud.AuthScope)

	// Application credentials don't support scope
	if isApplicationCredential(cloud.AuthInfo) {
		// If Domain* is set, but UserDomain* or ProjectDomain* aren't,
		// then use Domain* as the default setting.
		cloud = setDomainIfNeeded(cloud)
	} else {
		if !isProjectScoped(cloud.AuthInfo) {
			if cloud.AuthInfo.DomainID != "" {
				scope.DomainID = cloud.AuthInfo.DomainID
			} else if cloud.AuthInfo.DomainName != "" {
				scope.DomainName = cloud.AuthInfo.DomainName
			}
		} else {
			// If Domain* is set, but UserDomain* or ProjectDomain* aren't,
			// then use Domain* as the default setting.
			cloud = setDomainIfNeeded(cloud)

			if cloud.AuthInfo.ProjectID != "" {
				scope.ProjectID = cloud.AuthInfo.ProjectID
			} else {
				scope.ProjectName = cloud.AuthInfo.ProjectName
				scope.DomainID = cloud.AuthInfo.ProjectDomainID
				scope.DomainName = cloud.AuthInfo.ProjectDomainName
			}
		}
	}

	ao := &gophercloud.AuthOptions{
		Scope:                       scope,
		IdentityEndpoint:            cloud.AuthInfo.AuthURL,
		TokenID:                     cloud.AuthInfo.Token,
		Username:                    cloud.AuthInfo.Username,
		UserID:                      cloud.AuthInfo.UserID,
		Password:                    cloud.AuthInfo.Password,
		TenantID:                    cloud.AuthInfo.ProjectID,
		TenantName:                  cloud.AuthInfo.ProjectName,
		DomainID:                    cloud.AuthInfo.UserDomainID,
		DomainName:                  cloud.AuthInfo.UserDomainName,
		ApplicationCredentialID:     cloud.AuthInfo.ApplicationCredentialID,
		ApplicationCredentialName:   cloud.AuthInfo.ApplicationCredentialName,
		ApplicationCredentialSecret: cloud.AuthInfo.ApplicationCredentialSecret,
	}

	// If an auth_type of "token" was specified, then make sure
	// Gophercloud properly authenticates with a token. This involves
	// unsetting a few other auth options. The reason this is done
	// here is to wait until all auth settings (both in clouds.yaml
	// and via environment variables) are set and then unset them.
	if strings.Contains(string(cloud.AuthType), "token") || ao.TokenID != "" {
		ao.Username = ""
		ao.Password = ""
		ao.UserID = ""
		ao.DomainID = ""
		ao.DomainName = ""
	}

	// Check for absolute minimum requirements.
	if ao.IdentityEndpoint == "" {
		err := gophercloud.ErrMissingInput{Argument: "auth_url"}
		return nil, err
	}

	return ao, nil
}

// AuthenticatedClient is a convenience function to get a new provider client
// based on a clouds.yaml entry.
func AuthenticatedClient(opts *ClientOpts) (*gophercloud.ProviderClient, error) {
	ao, err := AuthOptions(opts)
	if err != nil {
		return nil, err
	}

	return openstack.AuthenticatedClient(*ao)
}

// NewServiceClient is a convenience function to get a new service client.
func NewServiceClient(service string, opts *ClientOpts) (*gophercloud.ServiceClient, error) {
	cloud := new(Cloud)

	// If no opts were passed in, create an empty ClientOpts.
	if opts == nil {
		opts = new(ClientOpts)
	}

	// Determine if a clouds.yaml entry should be retrieved.
	// Start by figuring out the cloud name.
	// First check if one was explicitly specified in opts.
	var cloudName string
	if opts.Cloud != "" {
		cloudName = opts.Cloud
	}

	// Next see if a cloud name was specified as an environment variable.
	envPrefix := "OS_"
	if opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if v := os.Getenv(envPrefix + "CLOUD"); v != "" {
		cloudName = v
	}

	// If a cloud name was determined, try to look it up in clouds.yaml.
	if cloudName != "" {
		// Get the requested cloud.
		var err error
		cloud, err = GetCloudFromYAML(opts)
		if err != nil {
			return nil, err
		}
	}

	// Get a Provider Client
	pClient, err := AuthenticatedClient(opts)
	if err != nil {
		return nil, err
	}

	// Determine the region to use.
	// First, check if the REGION_NAME environment variable is set.
	var region string
	if v := os.Getenv(envPrefix + "REGION_NAME"); v != "" {
		region = v
	}

	// Next, check if the cloud entry sets a region.
	if v := cloud.RegionName; v != "" {
		region = v
	}

	// Finally, see if one was specified in the ClientOpts.
	// If so, this takes precedence.
	if v := opts.RegionName; v != "" {
		region = v
	}

	eo := gophercloud.EndpointOpts{
		Region: region,
	}

	switch service {
	case "clustering":
		return openstack.NewClusteringV1(pClient, eo)
	case "compute":
		return openstack.NewComputeV2(pClient, eo)
	case "container":
		return openstack.NewContainerV1(pClient, eo)
	case "database":
		return openstack.NewDBV1(pClient, eo)
	case "dns":
		return openstack.NewDNSV2(pClient, eo)
	case "identity":
		identityVersion := "3"
		if v := cloud.IdentityAPIVersion; v != "" {
			identityVersion = v
		}

		switch identityVersion {
		case "v2", "2", "2.0":
			return openstack.NewIdentityV2(pClient, eo)
		case "v3", "3":
			return openstack.NewIdentityV3(pClient, eo)
		default:
			return nil, fmt.Errorf("invalid identity API version")
		}
	case "image":
		return openstack.NewImageServiceV2(pClient, eo)
	case "load-balancer":
		return openstack.NewLoadBalancerV2(pClient, eo)
	case "network":
		return openstack.NewNetworkV2(pClient, eo)
	case "object-store":
		return openstack.NewObjectStorageV1(pClient, eo)
	case "orchestration":
		return openstack.NewOrchestrationV1(pClient, eo)
	case "sharev2":
		return openstack.NewSharedFileSystemV2(pClient, eo)
	case "volume":
		volumeVersion := "2"
		if v := cloud.VolumeAPIVersion; v != "" {
			volumeVersion = v
		}

		switch volumeVersion {
		case "v1", "1":
			return openstack.NewBlockStorageV1(pClient, eo)
		case "v2", "2":
			return openstack.NewBlockStorageV2(pClient, eo)
		case "v3", "3":
			return openstack.NewBlockStorageV3(pClient, eo)
		default:
			return nil, fmt.Errorf("invalid volume API version")
		}
	}

	return nil, fmt.Errorf("unable to create a service client for %s", service)
}

// isProjectScoped determines if an auth struct is project scoped.
func isProjectScoped(authInfo *AuthInfo) bool {
	if authInfo.ProjectID == "" && authInfo.ProjectName == "" {
		return false
	}

	return true
}

// setDomainIfNeeded will set a DomainID and DomainName
// to ProjectDomain* and UserDomain* if not already set.
func setDomainIfNeeded(cloud *Cloud) *Cloud {
	if cloud.AuthInfo.DomainID != "" {
		if cloud.AuthInfo.UserDomainID == "" {
			cloud.AuthInfo.UserDomainID = cloud.AuthInfo.DomainID
		}

		if cloud.AuthInfo.ProjectDomainID == "" {
			cloud.AuthInfo.ProjectDomainID = cloud.AuthInfo.DomainID
		}

		cloud.AuthInfo.DomainID = ""
	}

	if cloud.AuthInfo.DomainName != "" {
		if cloud.AuthInfo.UserDomainName == "" {
			cloud.AuthInfo.UserDomainName = cloud.AuthInfo.DomainName
		}

		if cloud.AuthInfo.ProjectDomainName == "" {
			cloud.AuthInfo.ProjectDomainName = cloud.AuthInfo.DomainName
		}

		cloud.AuthInfo.DomainName = ""
	}

	// If Domain fields are still not set, and if DefaultDomain has a value,
	// set UserDomainID and ProjectDomainID to DefaultDomain.
	// https://github.com/openstack/osc-lib/blob/86129e6f88289ef14bfaa3f7c9cdfbea8d9fc944/osc_lib/cli/client_config.py#L117-L146
	if cloud.AuthInfo.DefaultDomain != "" {
		if cloud.AuthInfo.UserDomainName == "" && cloud.AuthInfo.UserDomainID == "" {
			cloud.AuthInfo.UserDomainID = cloud.AuthInfo.DefaultDomain
		}

		if cloud.AuthInfo.ProjectDomainName == "" && cloud.AuthInfo.ProjectDomainID == "" {
			cloud.AuthInfo.ProjectDomainID = cloud.AuthInfo.DefaultDomain
		}
	}

	return cloud
}

// isApplicationCredential determines if an application credential is used to auth.
func isApplicationCredential(authInfo *AuthInfo) bool {
	if authInfo.ApplicationCredentialID == "" && authInfo.ApplicationCredentialName == "" && authInfo.ApplicationCredentialSecret == "" {
		return false
	}
	return true
}
