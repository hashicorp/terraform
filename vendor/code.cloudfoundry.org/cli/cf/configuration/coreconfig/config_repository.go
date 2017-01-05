package coreconfig

import (
	"strings"
	"sync"

	"code.cloudfoundry.org/cli/cf/configuration"
	"code.cloudfoundry.org/cli/cf/models"
	"github.com/blang/semver"
)

type ConfigRepository struct {
	data      *Data
	mutex     *sync.RWMutex
	initOnce  *sync.Once
	persistor configuration.Persistor
	onError   func(error)
}

type CCInfo struct {
	APIVersion               string `json:"api_version"`
	AuthorizationEndpoint    string `json:"authorization_endpoint"`
	LoggregatorEndpoint      string `json:"logging_endpoint"`
	DopplerEndpoint          string `json:"doppler_logging_endpoint"`
	MinCLIVersion            string `json:"min_cli_version"`
	MinRecommendedCLIVersion string `json:"min_recommended_cli_version"`
	SSHOAuthClient           string `json:"app_ssh_oauth_client"`
	RoutingAPIEndpoint       string `json:"routing_endpoint"`
}

func NewRepositoryFromFilepath(filepath string, errorHandler func(error)) Repository {
	if errorHandler == nil {
		return nil
	}
	return NewRepositoryFromPersistor(configuration.NewDiskPersistor(filepath), errorHandler)
}

func NewRepositoryFromPersistor(persistor configuration.Persistor, errorHandler func(error)) Repository {
	data := NewData()
	if !persistor.Exists() {
		//set default plugin repo
		data.PluginRepos = append(data.PluginRepos, models.PluginRepo{
			Name: "CF-Community",
			URL:  "https://plugins.cloudfoundry.org",
		})
	}

	return &ConfigRepository{
		data:      data,
		mutex:     new(sync.RWMutex),
		initOnce:  new(sync.Once),
		persistor: persistor,
		onError:   errorHandler,
	}
}

type Reader interface {
	APIEndpoint() string
	APIVersion() string
	HasAPIEndpoint() bool

	AuthenticationEndpoint() string
	LoggregatorEndpoint() string
	DopplerEndpoint() string
	UaaEndpoint() string
	RoutingAPIEndpoint() string
	AccessToken() string
	SSHOAuthClient() string
	RefreshToken() string

	OrganizationFields() models.OrganizationFields
	HasOrganization() bool

	SpaceFields() models.SpaceFields
	HasSpace() bool

	Username() string
	UserGUID() string
	UserEmail() string
	IsLoggedIn() bool
	IsSSLDisabled() bool
	IsMinAPIVersion(semver.Version) bool
	IsMinCLIVersion(string) bool
	MinCLIVersion() string
	MinRecommendedCLIVersion() string

	AsyncTimeout() uint
	Trace() string

	ColorEnabled() string

	Locale() string

	PluginRepos() []models.PluginRepo
}

//go:generate counterfeiter . ReadWriter

type ReadWriter interface {
	Reader
	ClearSession()
	SetAPIEndpoint(string)
	SetAPIVersion(string)
	SetMinCLIVersion(string)
	SetMinRecommendedCLIVersion(string)
	SetAuthenticationEndpoint(string)
	SetLoggregatorEndpoint(string)
	SetDopplerEndpoint(string)
	SetUaaEndpoint(string)
	SetRoutingAPIEndpoint(string)
	SetAccessToken(string)
	SetSSHOAuthClient(string)
	SetRefreshToken(string)
	SetOrganizationFields(models.OrganizationFields)
	SetSpaceFields(models.SpaceFields)
	SetSSLDisabled(bool)
	SetAsyncTimeout(uint)
	SetTrace(string)
	SetColorEnabled(string)
	SetLocale(string)
	SetPluginRepo(models.PluginRepo)
	UnSetPluginRepo(int)
}

//go:generate counterfeiter . Repository

type Repository interface {
	ReadWriter
	Close()
}

// ACCESS CONTROL

func (c *ConfigRepository) init() {
	c.initOnce.Do(func() {
		err := c.persistor.Load(c.data)
		if err != nil {
			c.onError(err)
		}
	})
}

func (c *ConfigRepository) read(cb func()) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	c.init()

	cb()
}

func (c *ConfigRepository) write(cb func()) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.init()

	cb()

	err := c.persistor.Save(c.data)
	if err != nil {
		c.onError(err)
	}
}

// CLOSERS

func (c *ConfigRepository) Close() {
	c.read(func() {
		// perform a read to ensure write lock has been cleared
	})
}

// GETTERS

func (c *ConfigRepository) APIVersion() (apiVersion string) {
	c.read(func() {
		apiVersion = c.data.APIVersion
	})
	return
}

func (c *ConfigRepository) AuthenticationEndpoint() (authEndpoint string) {
	c.read(func() {
		authEndpoint = c.data.AuthorizationEndpoint
	})
	return
}

func (c *ConfigRepository) LoggregatorEndpoint() (logEndpoint string) {
	c.read(func() {
		logEndpoint = c.data.LoggregatorEndPoint
	})
	return
}

func (c *ConfigRepository) DopplerEndpoint() (dopplerEndpoint string) {
	//revert this in v7.0, once CC advertise doppler endpoint, and
	//everyone has migrated from loggregator to doppler
	c.read(func() {
		dopplerEndpoint = c.data.DopplerEndPoint
	})

	if dopplerEndpoint == "" {
		return strings.Replace(c.LoggregatorEndpoint(), "loggregator", "doppler", 1)
	}
	return
}

func (c *ConfigRepository) UaaEndpoint() (uaaEndpoint string) {
	c.read(func() {
		uaaEndpoint = c.data.UaaEndpoint
	})
	return
}

func (c *ConfigRepository) RoutingAPIEndpoint() (routingAPIEndpoint string) {
	c.read(func() {
		routingAPIEndpoint = c.data.RoutingAPIEndpoint
	})
	return
}

func (c *ConfigRepository) APIEndpoint() (apiEndpoint string) {
	c.read(func() {
		apiEndpoint = c.data.Target
	})
	return
}

func (c *ConfigRepository) HasAPIEndpoint() (hasEndpoint bool) {
	c.read(func() {
		hasEndpoint = c.data.APIVersion != "" && c.data.Target != ""
	})
	return
}

func (c *ConfigRepository) AccessToken() (accessToken string) {
	c.read(func() {
		accessToken = c.data.AccessToken
	})
	return
}

func (c *ConfigRepository) SSHOAuthClient() (clientID string) {
	c.read(func() {
		clientID = c.data.SSHOAuthClient
	})
	return
}

func (c *ConfigRepository) RefreshToken() (refreshToken string) {
	c.read(func() {
		refreshToken = c.data.RefreshToken
	})
	return
}

func (c *ConfigRepository) OrganizationFields() (org models.OrganizationFields) {
	c.read(func() {
		org = c.data.OrganizationFields
	})
	return
}

func (c *ConfigRepository) SpaceFields() (space models.SpaceFields) {
	c.read(func() {
		space = c.data.SpaceFields
	})
	return
}

func (c *ConfigRepository) UserEmail() (email string) {
	c.read(func() {
		email = NewTokenInfo(c.data.AccessToken).Email
	})
	return
}

func (c *ConfigRepository) UserGUID() (guid string) {
	c.read(func() {
		guid = NewTokenInfo(c.data.AccessToken).UserGUID
	})
	return
}

func (c *ConfigRepository) Username() (name string) {
	c.read(func() {
		name = NewTokenInfo(c.data.AccessToken).Username
	})
	return
}

func (c *ConfigRepository) IsLoggedIn() (loggedIn bool) {
	c.read(func() {
		loggedIn = c.data.AccessToken != ""
	})
	return
}

func (c *ConfigRepository) HasOrganization() (hasOrg bool) {
	c.read(func() {
		hasOrg = c.data.OrganizationFields.GUID != "" && c.data.OrganizationFields.Name != ""
	})
	return
}

func (c *ConfigRepository) HasSpace() (hasSpace bool) {
	c.read(func() {
		hasSpace = c.data.SpaceFields.GUID != "" && c.data.SpaceFields.Name != ""
	})
	return
}

func (c *ConfigRepository) IsSSLDisabled() (isSSLDisabled bool) {
	c.read(func() {
		isSSLDisabled = c.data.SSLDisabled
	})
	return
}

func (c *ConfigRepository) IsMinAPIVersion(requiredVersion semver.Version) bool {
	var apiVersion string
	c.read(func() {
		apiVersion = c.data.APIVersion
	})

	actualVersion, err := semver.Make(apiVersion)
	if err != nil {
		return false
	}
	return actualVersion.GTE(requiredVersion)
}

func (c *ConfigRepository) IsMinCLIVersion(version string) bool {
	if version == "BUILT_FROM_SOURCE" {
		return true
	}
	var minCLIVersion string
	c.read(func() {
		minCLIVersion = c.data.MinCLIVersion
	})
	if minCLIVersion == "" {
		return true
	}

	actualVersion, err := semver.Make(version)
	if err != nil {
		return false
	}
	requiredVersion, err := semver.Make(minCLIVersion)
	if err != nil {
		return false
	}
	return actualVersion.GTE(requiredVersion)
}

func (c *ConfigRepository) MinCLIVersion() (minCLIVersion string) {
	c.read(func() {
		minCLIVersion = c.data.MinCLIVersion
	})
	return
}

func (c *ConfigRepository) MinRecommendedCLIVersion() (minRecommendedCLIVersion string) {
	c.read(func() {
		minRecommendedCLIVersion = c.data.MinRecommendedCLIVersion
	})
	return
}

func (c *ConfigRepository) AsyncTimeout() (timeout uint) {
	c.read(func() {
		timeout = c.data.AsyncTimeout
	})
	return
}

func (c *ConfigRepository) Trace() (trace string) {
	c.read(func() {
		trace = c.data.Trace
	})
	return
}

func (c *ConfigRepository) ColorEnabled() (enabled string) {
	c.read(func() {
		enabled = c.data.ColorEnabled
	})
	return
}

func (c *ConfigRepository) Locale() (locale string) {
	c.read(func() {
		locale = c.data.Locale
	})
	return
}

func (c *ConfigRepository) PluginRepos() (repos []models.PluginRepo) {
	c.read(func() {
		repos = c.data.PluginRepos
	})
	return
}

// SETTERS

func (c *ConfigRepository) ClearSession() {
	c.write(func() {
		c.data.AccessToken = ""
		c.data.RefreshToken = ""
		c.data.OrganizationFields = models.OrganizationFields{}
		c.data.SpaceFields = models.SpaceFields{}
	})
}

func (c *ConfigRepository) SetAPIEndpoint(endpoint string) {
	c.write(func() {
		c.data.Target = endpoint
	})
}

func (c *ConfigRepository) SetAPIVersion(version string) {
	c.write(func() {
		c.data.APIVersion = version
	})
}

func (c *ConfigRepository) SetMinCLIVersion(version string) {
	c.write(func() {
		c.data.MinCLIVersion = version
	})
}

func (c *ConfigRepository) SetMinRecommendedCLIVersion(version string) {
	c.write(func() {
		c.data.MinRecommendedCLIVersion = version
	})
}

func (c *ConfigRepository) SetAuthenticationEndpoint(endpoint string) {
	c.write(func() {
		c.data.AuthorizationEndpoint = endpoint
	})
}

func (c *ConfigRepository) SetLoggregatorEndpoint(endpoint string) {
	c.write(func() {
		c.data.LoggregatorEndPoint = endpoint
	})
}

func (c *ConfigRepository) SetDopplerEndpoint(endpoint string) {
	c.write(func() {
		c.data.DopplerEndPoint = endpoint
	})
}

func (c *ConfigRepository) SetUaaEndpoint(uaaEndpoint string) {
	c.write(func() {
		c.data.UaaEndpoint = uaaEndpoint
	})
}

func (c *ConfigRepository) SetRoutingAPIEndpoint(routingAPIEndpoint string) {
	c.write(func() {
		c.data.RoutingAPIEndpoint = routingAPIEndpoint
	})
}

func (c *ConfigRepository) SetAccessToken(token string) {
	c.write(func() {
		c.data.AccessToken = token
	})
}

func (c *ConfigRepository) SetSSHOAuthClient(clientID string) {
	c.write(func() {
		c.data.SSHOAuthClient = clientID
	})
}

func (c *ConfigRepository) SetRefreshToken(token string) {
	c.write(func() {
		c.data.RefreshToken = token
	})
}

func (c *ConfigRepository) SetOrganizationFields(org models.OrganizationFields) {
	c.write(func() {
		c.data.OrganizationFields = org
	})
}

func (c *ConfigRepository) SetSpaceFields(space models.SpaceFields) {
	c.write(func() {
		c.data.SpaceFields = space
	})
}

func (c *ConfigRepository) SetSSLDisabled(disabled bool) {
	c.write(func() {
		c.data.SSLDisabled = disabled
	})
}

func (c *ConfigRepository) SetAsyncTimeout(timeout uint) {
	c.write(func() {
		c.data.AsyncTimeout = timeout
	})
}

func (c *ConfigRepository) SetTrace(value string) {
	c.write(func() {
		c.data.Trace = value
	})
}

func (c *ConfigRepository) SetColorEnabled(enabled string) {
	c.write(func() {
		c.data.ColorEnabled = enabled
	})
}

func (c *ConfigRepository) SetLocale(locale string) {
	c.write(func() {
		c.data.Locale = locale
	})
}

func (c *ConfigRepository) SetPluginRepo(repo models.PluginRepo) {
	c.write(func() {
		c.data.PluginRepos = append(c.data.PluginRepos, repo)
	})
}

func (c *ConfigRepository) UnSetPluginRepo(index int) {
	c.write(func() {
		c.data.PluginRepos = append(c.data.PluginRepos[:index], c.data.PluginRepos[index+1:]...)
	})
}
