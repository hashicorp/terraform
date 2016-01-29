// Generated service client for heroku API.
//
// To be able to interact with this API, you have to
// create a new service:
//
//     s := heroku.NewService(nil)
//
// The Service struct has all the methods you need
// to interact with heroku API.
//
package heroku

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime"
	"time"
)

const (
	Version          = "v3"
	DefaultAPIURL    = "https://api.heroku.com"
	DefaultUserAgent = "heroku/" + Version + " (" + runtime.GOOS + "; " + runtime.GOARCH + ")"
)

// Service represents your API.
type Service struct {
	client *http.Client
}

// NewService creates a Service using the given, if none is provided
// it uses http.DefaultClient.
func NewService(c *http.Client) *Service {
	if c == nil {
		c = http.DefaultClient
	}
	return &Service{
		client: c,
	}
}

// NewRequest generates an HTTP request, but does not perform the request.
func (s *Service) NewRequest(method, path string, body interface{}) (*http.Request, error) {
	var ctype string
	var rbody io.Reader
	switch t := body.(type) {
	case nil:
	case string:
		rbody = bytes.NewBufferString(t)
	case io.Reader:
		rbody = t
	default:
		v := reflect.ValueOf(body)
		if !v.IsValid() {
			break
		}
		if v.Type().Kind() == reflect.Ptr {
			v = reflect.Indirect(v)
			if !v.IsValid() {
				break
			}
		}
		j, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rbody = bytes.NewReader(j)
		ctype = "application/json"
	}
	req, err := http.NewRequest(method, DefaultAPIURL+path, rbody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", DefaultUserAgent)
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	return req, nil
}

// Do sends a request and decodes the response into v.
func (s *Service) Do(v interface{}, method, path string, body interface{}, lr *ListRange) error {
	req, err := s.NewRequest(method, path, body)
	if err != nil {
		return err
	}
	if lr != nil {
		lr.SetHeader(req)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch t := v.(type) {
	case nil:
	case io.Writer:
		_, err = io.Copy(t, resp.Body)
	default:
		err = json.NewDecoder(resp.Body).Decode(v)
	}
	return err
}

// Get sends a GET request and decodes the response into v.
func (s *Service) Get(v interface{}, path string, lr *ListRange) error {
	return s.Do(v, "GET", path, nil, lr)
}

// Patch sends a Path request and decodes the response into v.
func (s *Service) Patch(v interface{}, path string, body interface{}) error {
	return s.Do(v, "PATCH", path, body, nil)
}

// Post sends a POST request and decodes the response into v.
func (s *Service) Post(v interface{}, path string, body interface{}) error {
	return s.Do(v, "POST", path, body, nil)
}

// Put sends a PUT request and decodes the response into v.
func (s *Service) Put(v interface{}, path string, body interface{}) error {
	return s.Do(v, "PUT", path, body, nil)
}

// Delete sends a DELETE request.
func (s *Service) Delete(path string) error {
	return s.Do(nil, "DELETE", path, nil, nil)
}

// ListRange describes a range.
type ListRange struct {
	Field      string
	Max        int
	Descending bool
	FirstID    string
	LastID     string
}

// SetHeader set headers on the given Request.
func (lr *ListRange) SetHeader(req *http.Request) {
	var hdrval string
	if lr.Field != "" {
		hdrval += lr.Field + " "
	}
	hdrval += lr.FirstID + ".." + lr.LastID
	if lr.Max != 0 {
		hdrval += fmt.Sprintf("; max=%d", lr.Max)
		if lr.Descending {
			hdrval += ", "
		}
	}
	if lr.Descending {
		hdrval += ", order=desc"
	}
	req.Header.Set("Range", hdrval)
	return
}

// Bool allocates a new int value returns a pointer to it.
func Bool(v bool) *bool {
	p := new(bool)
	*p = v
	return p
}

// Int allocates a new int value returns a pointer to it.
func Int(v int) *int {
	p := new(int)
	*p = v
	return p
}

// Float64 allocates a new float64 value returns a pointer to it.
func Float64(v float64) *float64 {
	p := new(float64)
	*p = v
	return p
}

// String allocates a new string value returns a pointer to it.
func String(v string) *string {
	p := new(string)
	*p = v
	return p
}

// An account represents an individual signed up to use the Heroku
// platform.
type Account struct {
	AllowTracking bool      `json:"allow_tracking"` // whether to allow third party web activity tracking
	Beta          bool      `json:"beta"`           // whether allowed to utilize beta Heroku features
	CreatedAt     time.Time `json:"created_at"`     // when account was created
	Email         string    `json:"email"`          // unique email address of account
	ID            string    `json:"id"`             // unique identifier of an account
	LastLogin     time.Time `json:"last_login"`     // when account last authorized with Heroku
	Name          *string   `json:"name"`           // full name of the account owner
	UpdatedAt     time.Time `json:"updated_at"`     // when account was updated
	Verified      bool      `json:"verified"`       // whether account has been verified with billing information
}

// Info for account.
func (s *Service) AccountInfo() (*Account, error) {
	var account Account
	return &account, s.Get(&account, fmt.Sprintf("/account"), nil)
}

type AccountUpdateOpts struct {
	AllowTracking *bool   `json:"allow_tracking,omitempty"` // whether to allow third party web activity tracking
	Beta          *bool   `json:"beta,omitempty"`           // whether allowed to utilize beta Heroku features
	Name          *string `json:"name,omitempty"`           // full name of the account owner
	Password      string  `json:"password"`                 // current password on the account
}

// Update account.
func (s *Service) AccountUpdate(o struct {
	AllowTracking *bool   `json:"allow_tracking,omitempty"` // whether to allow third party web activity tracking
	Beta          *bool   `json:"beta,omitempty"`           // whether allowed to utilize beta Heroku features
	Name          *string `json:"name,omitempty"`           // full name of the account owner
	Password      string  `json:"password"`                 // current password on the account
}) (*Account, error) {
	var account Account
	return &account, s.Patch(&account, fmt.Sprintf("/account"), o)
}

type AccountChangeEmailOpts struct {
	Email    string `json:"email"`    // unique email address of account
	Password string `json:"password"` // current password on the account
}

// Change Email for account.
func (s *Service) AccountChangeEmail(o struct {
	Email    string `json:"email"`    // unique email address of account
	Password string `json:"password"` // current password on the account
}) (*Account, error) {
	var account Account
	return &account, s.Patch(&account, fmt.Sprintf("/account"), o)
}

type AccountChangePasswordOpts struct {
	NewPassword string `json:"new_password"` // the new password for the account when changing the password
	Password    string `json:"password"`     // current password on the account
}

// Change Password for account.
func (s *Service) AccountChangePassword(o struct {
	NewPassword string `json:"new_password"` // the new password for the account when changing the password
	Password    string `json:"password"`     // current password on the account
}) (*Account, error) {
	var account Account
	return &account, s.Patch(&account, fmt.Sprintf("/account"), o)
}

// An account feature represents a Heroku labs capability that can be
// enabled or disabled for an account on Heroku.
type AccountFeature struct {
	CreatedAt   time.Time `json:"created_at"`  // when account feature was created
	Description string    `json:"description"` // description of account feature
	DocURL      string    `json:"doc_url"`     // documentation URL of account feature
	Enabled     bool      `json:"enabled"`     // whether or not account feature has been enabled
	ID          string    `json:"id"`          // unique identifier of account feature
	Name        string    `json:"name"`        // unique name of account feature
	State       string    `json:"state"`       // state of account feature
	UpdatedAt   time.Time `json:"updated_at"`  // when account feature was updated
}

// Info for an existing account feature.
func (s *Service) AccountFeatureInfo(accountFeatureIdentity string) (*AccountFeature, error) {
	var accountFeature AccountFeature
	return &accountFeature, s.Get(&accountFeature, fmt.Sprintf("/account/features/%v", accountFeatureIdentity), nil)
}

// List existing account features.
func (s *Service) AccountFeatureList(lr *ListRange) ([]*AccountFeature, error) {
	var accountFeatureList []*AccountFeature
	return accountFeatureList, s.Get(&accountFeatureList, fmt.Sprintf("/account/features"), lr)
}

type AccountFeatureUpdateOpts struct {
	Enabled bool `json:"enabled"` // whether or not account feature has been enabled
}

// Update an existing account feature.
func (s *Service) AccountFeatureUpdate(accountFeatureIdentity string, o struct {
	Enabled bool `json:"enabled"` // whether or not account feature has been enabled
}) (*AccountFeature, error) {
	var accountFeature AccountFeature
	return &accountFeature, s.Patch(&accountFeature, fmt.Sprintf("/account/features/%v", accountFeatureIdentity), o)
}

// Add-ons represent add-ons that have been provisioned for an app.
type Addon struct {
	AddonService struct {
		ID   string `json:"id"`   // unique identifier of this addon-service
		Name string `json:"name"` // unique name of this addon-service
	} `json:"addon_service"` // identity of add-on service
	ConfigVars []string  `json:"config_vars"` // config vars associated with this application
	CreatedAt  time.Time `json:"created_at"`  // when add-on was updated
	ID         string    `json:"id"`          // unique identifier of add-on
	Name       string    `json:"name"`        // name of the add-on unique within its app
	Plan       struct {
		ID   string `json:"id"`   // unique identifier of this plan
		Name string `json:"name"` // unique name of this plan
	} `json:"plan"` // identity of add-on plan
	ProviderID string    `json:"provider_id"` // id of this add-on with its provider
	UpdatedAt  time.Time `json:"updated_at"`  // when add-on was updated
}
type AddonCreateOpts struct {
	Config *map[string]string `json:"config,omitempty"` // custom add-on provisioning options
	Plan   string             `json:"plan"`             // unique identifier of this plan
}

// Create a new add-on.
func (s *Service) AddonCreate(appIdentity string, o struct {
	Config *map[string]string `json:"config,omitempty"` // custom add-on provisioning options
	Plan   string             `json:"plan"`             // unique identifier of this plan
}) (*Addon, error) {
	var addon Addon
	return &addon, s.Post(&addon, fmt.Sprintf("/apps/%v/addons", appIdentity), o)
}

// Delete an existing add-on.
func (s *Service) AddonDelete(appIdentity string, addonIdentity string) error {
	return s.Delete(fmt.Sprintf("/apps/%v/addons/%v", appIdentity, addonIdentity))
}

// Info for an existing add-on.
func (s *Service) AddonInfo(appIdentity string, addonIdentity string) (*Addon, error) {
	var addon Addon
	return &addon, s.Get(&addon, fmt.Sprintf("/apps/%v/addons/%v", appIdentity, addonIdentity), nil)
}

// List existing add-ons.
func (s *Service) AddonList(appIdentity string, lr *ListRange) ([]*Addon, error) {
	var addonList []*Addon
	return addonList, s.Get(&addonList, fmt.Sprintf("/apps/%v/addons", appIdentity), lr)
}

type AddonUpdateOpts struct {
	Plan string `json:"plan"` // unique identifier of this plan
}

// Change add-on plan. Some add-ons may not support changing plans. In
// that case, an error will be returned.
func (s *Service) AddonUpdate(appIdentity string, addonIdentity string, o struct {
	Plan string `json:"plan"` // unique identifier of this plan
}) (*Addon, error) {
	var addon Addon
	return &addon, s.Patch(&addon, fmt.Sprintf("/apps/%v/addons/%v", appIdentity, addonIdentity), o)
}

// Add-on services represent add-ons that may be provisioned for apps.
// Endpoints under add-on services can be accessed without
// authentication.
type AddonService struct {
	CreatedAt time.Time `json:"created_at"` // when addon-service was created
	ID        string    `json:"id"`         // unique identifier of this addon-service
	Name      string    `json:"name"`       // unique name of this addon-service
	UpdatedAt time.Time `json:"updated_at"` // when addon-service was updated
}

// Info for existing addon-service.
func (s *Service) AddonServiceInfo(addonServiceIdentity string) (*AddonService, error) {
	var addonService AddonService
	return &addonService, s.Get(&addonService, fmt.Sprintf("/addon-services/%v", addonServiceIdentity), nil)
}

// List existing addon-services.
func (s *Service) AddonServiceList(lr *ListRange) ([]*AddonService, error) {
	var addonServiceList []*AddonService
	return addonServiceList, s.Get(&addonServiceList, fmt.Sprintf("/addon-services"), lr)
}

// An app represents the program that you would like to deploy and run
// on Heroku.
type App struct {
	ArchivedAt                   *time.Time `json:"archived_at"`                    // when app was archived
	BuildpackProvidedDescription *string    `json:"buildpack_provided_description"` // description from buildpack of app
	CreatedAt                    time.Time  `json:"created_at"`                     // when app was created
	GitURL                       string     `json:"git_url"`                        // git repo URL of app
	ID                           string     `json:"id"`                             // unique identifier of app
	Maintenance                  bool       `json:"maintenance"`                    // maintenance status of app
	Name                         string     `json:"name"`                           // unique name of app
	Owner                        struct {
		Email string `json:"email"` // unique email address of account
		ID    string `json:"id"`    // unique identifier of an account
	} `json:"owner"` // identity of app owner
	Region struct {
		ID   string `json:"id"`   // unique identifier of region
		Name string `json:"name"` // unique name of region
	} `json:"region"` // identity of app region
	ReleasedAt *time.Time `json:"released_at"` // when app was released
	RepoSize   *int       `json:"repo_size"`   // git repo size in bytes of app
	SlugSize   *int       `json:"slug_size"`   // slug size in bytes of app
	Stack      struct {
		ID   string `json:"id"`   // unique identifier of stack
		Name string `json:"name"` // unique name of stack
	} `json:"stack"` // identity of app stack
	UpdatedAt time.Time `json:"updated_at"` // when app was updated
	WebURL    string    `json:"web_url"`    // web URL of app
}
type AppCreateOpts struct {
	Name   *string `json:"name,omitempty"`   // unique name of app
	Region *string `json:"region,omitempty"` // unique identifier of region
	Stack  *string `json:"stack,omitempty"`  // unique name of stack
}

// Create a new app.
func (s *Service) AppCreate(o struct {
	Name   *string `json:"name,omitempty"`   // unique name of app
	Region *string `json:"region,omitempty"` // unique identifier of region
	Stack  *string `json:"stack,omitempty"`  // unique name of stack
}) (*App, error) {
	var app App
	return &app, s.Post(&app, fmt.Sprintf("/apps"), o)
}

// Delete an existing app.
func (s *Service) AppDelete(appIdentity string) error {
	return s.Delete(fmt.Sprintf("/apps/%v", appIdentity))
}

// Info for existing app.
func (s *Service) AppInfo(appIdentity string) (*App, error) {
	var app App
	return &app, s.Get(&app, fmt.Sprintf("/apps/%v", appIdentity), nil)
}

// List existing apps.
func (s *Service) AppList(lr *ListRange) ([]*App, error) {
	var appList []*App
	return appList, s.Get(&appList, fmt.Sprintf("/apps"), lr)
}

type AppUpdateOpts struct {
	Maintenance *bool   `json:"maintenance,omitempty"` // maintenance status of app
	Name        *string `json:"name,omitempty"`        // unique name of app
}

// Update an existing app.
func (s *Service) AppUpdate(appIdentity string, o struct {
	Maintenance *bool   `json:"maintenance,omitempty"` // maintenance status of app
	Name        *string `json:"name,omitempty"`        // unique name of app
}) (*App, error) {
	var app App
	return &app, s.Patch(&app, fmt.Sprintf("/apps/%v", appIdentity), o)
}

// An app feature represents a Heroku labs capability that can be
// enabled or disabled for an app on Heroku.
type AppFeature struct {
	CreatedAt   time.Time `json:"created_at"`  // when app feature was created
	Description string    `json:"description"` // description of app feature
	DocURL      string    `json:"doc_url"`     // documentation URL of app feature
	Enabled     bool      `json:"enabled"`     // whether or not app feature has been enabled
	ID          string    `json:"id"`          // unique identifier of app feature
	Name        string    `json:"name"`        // unique name of app feature
	State       string    `json:"state"`       // state of app feature
	UpdatedAt   time.Time `json:"updated_at"`  // when app feature was updated
}

// Info for an existing app feature.
func (s *Service) AppFeatureInfo(appIdentity string, appFeatureIdentity string) (*AppFeature, error) {
	var appFeature AppFeature
	return &appFeature, s.Get(&appFeature, fmt.Sprintf("/apps/%v/features/%v", appIdentity, appFeatureIdentity), nil)
}

// List existing app features.
func (s *Service) AppFeatureList(appIdentity string, lr *ListRange) ([]*AppFeature, error) {
	var appFeatureList []*AppFeature
	return appFeatureList, s.Get(&appFeatureList, fmt.Sprintf("/apps/%v/features", appIdentity), lr)
}

type AppFeatureUpdateOpts struct {
	Enabled bool `json:"enabled"` // whether or not app feature has been enabled
}

// Update an existing app feature.
func (s *Service) AppFeatureUpdate(appIdentity string, appFeatureIdentity string, o struct {
	Enabled bool `json:"enabled"` // whether or not app feature has been enabled
}) (*AppFeature, error) {
	var appFeature AppFeature
	return &appFeature, s.Patch(&appFeature, fmt.Sprintf("/apps/%v/features/%v", appIdentity, appFeatureIdentity), o)
}

// An app setup represents an app on Heroku that is setup using an
// environment, addons, and scripts described in an app.json manifest
// file.
type AppSetup struct {
	App struct {
		ID   string `json:"id"`   // unique identifier of app
		Name string `json:"name"` // unique name of app
	} `json:"app"` // identity of app
	Build struct {
		ID     string `json:"id"`     // unique identifier of build
		Status string `json:"status"` // status of build
	} `json:"build"` // identity and status of build
	CreatedAt      time.Time `json:"created_at"`      // when app setup was created
	FailureMessage *string   `json:"failure_message"` // reason that app setup has failed
	ID             string    `json:"id"`              // unique identifier of app setup
	ManifestErrors []string  `json:"manifest_errors"` // errors associated with invalid app.json manifest file
	Postdeploy     *struct {
		ExitCode int    `json:"exit_code"` // The exit code of the postdeploy script
		Output   string `json:"output"`    // output of the postdeploy script
	} `json:"postdeploy"` // result of postdeploy script
	ResolvedSuccessURL *string   `json:"resolved_success_url"` // fully qualified success url
	Status             string    `json:"status"`               // the overall status of app setup
	UpdatedAt          time.Time `json:"updated_at"`           // when app setup was updated
}
type AppSetupCreateOpts struct {
	App *struct {
		Locked       *bool   `json:"locked,omitempty"`       // are other organization members forbidden from joining this app.
		Name         *string `json:"name,omitempty"`         // unique name of app
		Organization *string `json:"organization,omitempty"` // unique name of organization
		Personal     *bool   `json:"personal,omitempty"`     // force creation of the app in the user account even if a default org
		// is set.
		Region *string `json:"region,omitempty"` // unique name of region
		Stack  *string `json:"stack,omitempty"`  // unique name of stack
	} `json:"app,omitempty"` // optional parameters for created app
	Overrides *struct {
		Env *map[string]string `json:"env,omitempty"` // overrides of the env specified in the app.json manifest file
	} `json:"overrides,omitempty"` // overrides of keys in the app.json manifest file
	SourceBlob struct {
		URL *string `json:"url,omitempty"` // URL of gzipped tarball of source code containing app.json manifest
		// file
	} `json:"source_blob"` // gzipped tarball of source code containing app.json manifest file
}

// Create a new app setup from a gzipped tar archive containing an
// app.json manifest file.
func (s *Service) AppSetupCreate(o struct {
	App *struct {
		Locked       *bool   `json:"locked,omitempty"`       // are other organization members forbidden from joining this app.
		Name         *string `json:"name,omitempty"`         // unique name of app
		Organization *string `json:"organization,omitempty"` // unique name of organization
		Personal     *bool   `json:"personal,omitempty"`     // force creation of the app in the user account even if a default org
		// is set.
		Region *string `json:"region,omitempty"` // unique name of region
		Stack  *string `json:"stack,omitempty"`  // unique name of stack
	} `json:"app,omitempty"` // optional parameters for created app
	Overrides *struct {
		Env *map[string]string `json:"env,omitempty"` // overrides of the env specified in the app.json manifest file
	} `json:"overrides,omitempty"` // overrides of keys in the app.json manifest file
	SourceBlob struct {
		URL *string `json:"url,omitempty"` // URL of gzipped tarball of source code containing app.json manifest
		// file
	} `json:"source_blob"` // gzipped tarball of source code containing app.json manifest file
}) (*AppSetup, error) {
	var appSetup AppSetup
	return &appSetup, s.Post(&appSetup, fmt.Sprintf("/app-setups"), o)
}

// Get the status of an app setup.
func (s *Service) AppSetupInfo(appSetupIdentity string) (*AppSetup, error) {
	var appSetup AppSetup
	return &appSetup, s.Get(&appSetup, fmt.Sprintf("/app-setups/%v", appSetupIdentity), nil)
}

// An app transfer represents a two party interaction for transferring
// ownership of an app.
type AppTransfer struct {
	App struct {
		ID   string `json:"id"`   // unique identifier of app
		Name string `json:"name"` // unique name of app
	} `json:"app"` // app involved in the transfer
	CreatedAt time.Time `json:"created_at"` // when app transfer was created
	ID        string    `json:"id"`         // unique identifier of app transfer
	Owner     struct {
		Email string `json:"email"` // unique email address of account
		ID    string `json:"id"`    // unique identifier of an account
	} `json:"owner"` // identity of the owner of the transfer
	Recipient struct {
		Email string `json:"email"` // unique email address of account
		ID    string `json:"id"`    // unique identifier of an account
	} `json:"recipient"` // identity of the recipient of the transfer
	State     string    `json:"state"`      // the current state of an app transfer
	UpdatedAt time.Time `json:"updated_at"` // when app transfer was updated
}
type AppTransferCreateOpts struct {
	App       string `json:"app"`       // unique identifier of app
	Recipient string `json:"recipient"` // unique email address of account
}

// Create a new app transfer.
func (s *Service) AppTransferCreate(o struct {
	App       string `json:"app"`       // unique identifier of app
	Recipient string `json:"recipient"` // unique email address of account
}) (*AppTransfer, error) {
	var appTransfer AppTransfer
	return &appTransfer, s.Post(&appTransfer, fmt.Sprintf("/account/app-transfers"), o)
}

// Delete an existing app transfer
func (s *Service) AppTransferDelete(appTransferIdentity string) error {
	return s.Delete(fmt.Sprintf("/account/app-transfers/%v", appTransferIdentity))
}

// Info for existing app transfer.
func (s *Service) AppTransferInfo(appTransferIdentity string) (*AppTransfer, error) {
	var appTransfer AppTransfer
	return &appTransfer, s.Get(&appTransfer, fmt.Sprintf("/account/app-transfers/%v", appTransferIdentity), nil)
}

// List existing apps transfers.
func (s *Service) AppTransferList(lr *ListRange) ([]*AppTransfer, error) {
	var appTransferList []*AppTransfer
	return appTransferList, s.Get(&appTransferList, fmt.Sprintf("/account/app-transfers"), lr)
}

type AppTransferUpdateOpts struct {
	State string `json:"state"` // the current state of an app transfer
}

// Update an existing app transfer.
func (s *Service) AppTransferUpdate(appTransferIdentity string, o struct {
	State string `json:"state"` // the current state of an app transfer
}) (*AppTransfer, error) {
	var appTransfer AppTransfer
	return &appTransfer, s.Patch(&appTransfer, fmt.Sprintf("/account/app-transfers/%v", appTransferIdentity), o)
}

// A build represents the process of transforming a code tarball into a
// slug
type Build struct {
	CreatedAt time.Time `json:"created_at"` // when build was created
	ID        string    `json:"id"`         // unique identifier of build
	Slug      *struct {
		ID string `json:"id"` // unique identifier of slug
	} `json:"slug"` // slug created by this build
	SourceBlob struct {
		URL string `json:"url"` // URL where gzipped tar archive of source code for build was
		// downloaded.
		Version *string `json:"version"` // Version of the gzipped tarball.
	} `json:"source_blob"` // location of gzipped tarball of source code used to create build
	Status    string    `json:"status"`     // status of build
	UpdatedAt time.Time `json:"updated_at"` // when build was updated
	User      struct {
		Email string `json:"email"` // unique email address of account
		ID    string `json:"id"`    // unique identifier of an account
	} `json:"user"` // user that started the build
}
type BuildCreateOpts struct {
	SourceBlob struct {
		URL *string `json:"url,omitempty"` // URL where gzipped tar archive of source code for build was
		// downloaded.
		Version *string `json:"version,omitempty"` // Version of the gzipped tarball.
	} `json:"source_blob"` // location of gzipped tarball of source code used to create build
}

// Create a new build.
func (s *Service) BuildCreate(appIdentity string, o struct {
	SourceBlob struct {
		URL *string `json:"url,omitempty"` // URL where gzipped tar archive of source code for build was
		// downloaded.
		Version *string `json:"version,omitempty"` // Version of the gzipped tarball.
	} `json:"source_blob"` // location of gzipped tarball of source code used to create build
}) (*Build, error) {
	var build Build
	return &build, s.Post(&build, fmt.Sprintf("/apps/%v/builds", appIdentity), o)
}

// Info for existing build.
func (s *Service) BuildInfo(appIdentity string, buildIdentity string) (*Build, error) {
	var build Build
	return &build, s.Get(&build, fmt.Sprintf("/apps/%v/builds/%v", appIdentity, buildIdentity), nil)
}

// List existing build.
func (s *Service) BuildList(appIdentity string, lr *ListRange) ([]*Build, error) {
	var buildList []*Build
	return buildList, s.Get(&buildList, fmt.Sprintf("/apps/%v/builds", appIdentity), lr)
}

// A build result contains the output from a build.
type BuildResult struct {
	Build struct {
		ID     string `json:"id"`     // unique identifier of build
		Status string `json:"status"` // status of build
	} `json:"build"` // identity of build
	ExitCode float64 `json:"exit_code"` // status from the build
	Lines    []struct {
		Line   string `json:"line"`   // A line of output from the build.
		Stream string `json:"stream"` // The output stream where the line was sent.
	} `json:"lines"` // A list of all the lines of a build's output.
}

// Info for existing result.
func (s *Service) BuildResultInfo(appIdentity string, buildIdentity string) (*BuildResult, error) {
	var buildResult BuildResult
	return &buildResult, s.Get(&buildResult, fmt.Sprintf("/apps/%v/builds/%v/result", appIdentity, buildIdentity), nil)
}

// A collaborator represents an account that has been given access to an
// app on Heroku.
type Collaborator struct {
	CreatedAt time.Time `json:"created_at"` // when collaborator was created
	ID        string    `json:"id"`         // unique identifier of collaborator
	UpdatedAt time.Time `json:"updated_at"` // when collaborator was updated
	User      struct {
		Email string `json:"email"` // unique email address of account
		ID    string `json:"id"`    // unique identifier of an account
	} `json:"user"` // identity of collaborated account
}
type CollaboratorCreateOpts struct {
	Silent *bool  `json:"silent,omitempty"` // whether to suppress email invitation when creating collaborator
	User   string `json:"user"`             // unique email address of account
}

// Create a new collaborator.
func (s *Service) CollaboratorCreate(appIdentity string, o struct {
	Silent *bool  `json:"silent,omitempty"` // whether to suppress email invitation when creating collaborator
	User   string `json:"user"`             // unique email address of account
}) (*Collaborator, error) {
	var collaborator Collaborator
	return &collaborator, s.Post(&collaborator, fmt.Sprintf("/apps/%v/collaborators", appIdentity), o)
}

// Delete an existing collaborator.
func (s *Service) CollaboratorDelete(appIdentity string, collaboratorIdentity string) error {
	return s.Delete(fmt.Sprintf("/apps/%v/collaborators/%v", appIdentity, collaboratorIdentity))
}

// Info for existing collaborator.
func (s *Service) CollaboratorInfo(appIdentity string, collaboratorIdentity string) (*Collaborator, error) {
	var collaborator Collaborator
	return &collaborator, s.Get(&collaborator, fmt.Sprintf("/apps/%v/collaborators/%v", appIdentity, collaboratorIdentity), nil)
}

// List existing collaborators.
func (s *Service) CollaboratorList(appIdentity string, lr *ListRange) ([]*Collaborator, error) {
	var collaboratorList []*Collaborator
	return collaboratorList, s.Get(&collaboratorList, fmt.Sprintf("/apps/%v/collaborators", appIdentity), lr)
}

// Config Vars allow you to manage the configuration information
// provided to an app on Heroku.
type ConfigVar map[string]string

// Get config-vars for app.
func (s *Service) ConfigVarInfo(appIdentity string) (map[string]string, error) {
	var configVar ConfigVar
	return configVar, s.Get(&configVar, fmt.Sprintf("/apps/%v/config-vars", appIdentity), nil)
}

type ConfigVarUpdateOpts map[string]*string

// Update config-vars for app. You can update existing config-vars by
// setting them again, and remove by setting it to `NULL`.
func (s *Service) ConfigVarUpdate(appIdentity string, o map[string]*string) (map[string]string, error) {
	var configVar ConfigVar
	return configVar, s.Patch(&configVar, fmt.Sprintf("/apps/%v/config-vars", appIdentity), o)
}

// A credit represents value that will be used up before further charges
// are assigned to an account.
type Credit struct {
	Amount    float64   `json:"amount"`     // total value of credit in cents
	Balance   float64   `json:"balance"`    // remaining value of credit in cents
	CreatedAt time.Time `json:"created_at"` // when credit was created
	ExpiresAt time.Time `json:"expires_at"` // when credit will expire
	ID        string    `json:"id"`         // unique identifier of credit
	Title     string    `json:"title"`      // a name for credit
	UpdatedAt time.Time `json:"updated_at"` // when credit was updated
}

// Info for existing credit.
func (s *Service) CreditInfo(creditIdentity string) (*Credit, error) {
	var credit Credit
	return &credit, s.Get(&credit, fmt.Sprintf("/account/credits/%v", creditIdentity), nil)
}

// List existing credits.
func (s *Service) CreditList(lr *ListRange) ([]*Credit, error) {
	var creditList []*Credit
	return creditList, s.Get(&creditList, fmt.Sprintf("/account/credits"), lr)
}

// Domains define what web routes should be routed to an app on Heroku.
type Domain struct {
	CreatedAt time.Time `json:"created_at"` // when domain was created
	Hostname  string    `json:"hostname"`   // full hostname
	ID        string    `json:"id"`         // unique identifier of this domain
	UpdatedAt time.Time `json:"updated_at"` // when domain was updated
}
type DomainCreateOpts struct {
	Hostname string `json:"hostname"` // full hostname
}

// Create a new domain.
func (s *Service) DomainCreate(appIdentity string, o struct {
	Hostname string `json:"hostname"` // full hostname
}) (*Domain, error) {
	var domain Domain
	return &domain, s.Post(&domain, fmt.Sprintf("/apps/%v/domains", appIdentity), o)
}

// Delete an existing domain
func (s *Service) DomainDelete(appIdentity string, domainIdentity string) error {
	return s.Delete(fmt.Sprintf("/apps/%v/domains/%v", appIdentity, domainIdentity))
}

// Info for existing domain.
func (s *Service) DomainInfo(appIdentity string, domainIdentity string) (*Domain, error) {
	var domain Domain
	return &domain, s.Get(&domain, fmt.Sprintf("/apps/%v/domains/%v", appIdentity, domainIdentity), nil)
}

// List existing domains.
func (s *Service) DomainList(appIdentity string, lr *ListRange) ([]*Domain, error) {
	var domainList []*Domain
	return domainList, s.Get(&domainList, fmt.Sprintf("/apps/%v/domains", appIdentity), lr)
}

// Dynos encapsulate running processes of an app on Heroku.
type Dyno struct {
	AttachURL *string `json:"attach_url"` // a URL to stream output from for attached processes or null for
	// non-attached processes
	Command   string    `json:"command"`    // command used to start this process
	CreatedAt time.Time `json:"created_at"` // when dyno was created
	ID        string    `json:"id"`         // unique identifier of this dyno
	Name      string    `json:"name"`       // the name of this process on this dyno
	Release   struct {
		ID      string `json:"id"`      // unique identifier of release
		Version int    `json:"version"` // unique version assigned to the release
	} `json:"release"` // app release of the dyno
	Size  string `json:"size"`  // dyno size (default: "1X")
	State string `json:"state"` // current status of process (either: crashed, down, idle, starting, or
	// up)
	Type      string    `json:"type"`       // type of process
	UpdatedAt time.Time `json:"updated_at"` // when process last changed state
}
type DynoCreateOpts struct {
	Attach  *bool              `json:"attach,omitempty"` // whether to stream output or not
	Command string             `json:"command"`          // command used to start this process
	Env     *map[string]string `json:"env,omitempty"`    // custom environment to add to the dyno config vars
	Size    *string            `json:"size,omitempty"`   // dyno size (default: "1X")
}

// Create a new dyno.
func (s *Service) DynoCreate(appIdentity string, o struct {
	Attach  *bool              `json:"attach,omitempty"` // whether to stream output or not
	Command string             `json:"command"`          // command used to start this process
	Env     *map[string]string `json:"env,omitempty"`    // custom environment to add to the dyno config vars
	Size    *string            `json:"size,omitempty"`   // dyno size (default: "1X")
}) (*Dyno, error) {
	var dyno Dyno
	return &dyno, s.Post(&dyno, fmt.Sprintf("/apps/%v/dynos", appIdentity), o)
}

// Restart dyno.
func (s *Service) DynoRestart(appIdentity string, dynoIdentity string) error {
	return s.Delete(fmt.Sprintf("/apps/%v/dynos/%v", appIdentity, dynoIdentity))
}

// Restart all dynos
func (s *Service) DynoRestartAll(appIdentity string) error {
	return s.Delete(fmt.Sprintf("/apps/%v/dynos", appIdentity))
}

// Info for existing dyno.
func (s *Service) DynoInfo(appIdentity string, dynoIdentity string) (*Dyno, error) {
	var dyno Dyno
	return &dyno, s.Get(&dyno, fmt.Sprintf("/apps/%v/dynos/%v", appIdentity, dynoIdentity), nil)
}

// List existing dynos.
func (s *Service) DynoList(appIdentity string, lr *ListRange) ([]*Dyno, error) {
	var dynoList []*Dyno
	return dynoList, s.Get(&dynoList, fmt.Sprintf("/apps/%v/dynos", appIdentity), lr)
}

// The formation of processes that should be maintained for an app.
// Update the formation to scale processes or change dyno sizes.
// Available process type names and commands are defined by the
// `process_types` attribute for the [slug](#slug) currently released on
// an app.
type Formation struct {
	Command   string    `json:"command"`    // command to use to launch this process
	CreatedAt time.Time `json:"created_at"` // when process type was created
	ID        string    `json:"id"`         // unique identifier of this process type
	Quantity  int       `json:"quantity"`   // number of processes to maintain
	Size      string    `json:"size"`       // dyno size (default: "1X")
	Type      string    `json:"type"`       // type of process to maintain
	UpdatedAt time.Time `json:"updated_at"` // when dyno type was updated
}

// Info for a process type
func (s *Service) FormationInfo(appIdentity string, formationIdentity string) (*Formation, error) {
	var formation Formation
	return &formation, s.Get(&formation, fmt.Sprintf("/apps/%v/formation/%v", appIdentity, formationIdentity), nil)
}

// List process type formation
func (s *Service) FormationList(appIdentity string, lr *ListRange) ([]*Formation, error) {
	var formationList []*Formation
	return formationList, s.Get(&formationList, fmt.Sprintf("/apps/%v/formation", appIdentity), lr)
}

type FormationBatchUpdateOpts struct {
	Updates []struct {
		Process  string  `json:"process"`            // unique identifier of this process type
		Quantity *int    `json:"quantity,omitempty"` // number of processes to maintain
		Size     *string `json:"size,omitempty"`     // dyno size (default: "1X")
	} `json:"updates"` // Array with formation updates. Each element must have "process", the
	// id or name of the process type to be updated, and can optionally
	// update its "quantity" or "size".
}

// Batch update process types
func (s *Service) FormationBatchUpdate(appIdentity string, o struct {
	Updates []struct {
		Process  string  `json:"process"`            // unique identifier of this process type
		Quantity *int    `json:"quantity,omitempty"` // number of processes to maintain
		Size     *string `json:"size,omitempty"`     // dyno size (default: "1X")
	} `json:"updates"` // Array with formation updates. Each element must have "process", the
	// id or name of the process type to be updated, and can optionally
	// update its "quantity" or "size".
}) (*Formation, error) {
	var formation Formation
	return &formation, s.Patch(&formation, fmt.Sprintf("/apps/%v/formation", appIdentity), o)
}

type FormationUpdateOpts struct {
	Quantity *int    `json:"quantity,omitempty"` // number of processes to maintain
	Size     *string `json:"size,omitempty"`     // dyno size (default: "1X")
}

// Update process type
func (s *Service) FormationUpdate(appIdentity string, formationIdentity string, o struct {
	Quantity *int    `json:"quantity,omitempty"` // number of processes to maintain
	Size     *string `json:"size,omitempty"`     // dyno size (default: "1X")
}) (*Formation, error) {
	var formation Formation
	return &formation, s.Patch(&formation, fmt.Sprintf("/apps/%v/formation/%v", appIdentity, formationIdentity), o)
}

// Keys represent public SSH keys associated with an account and are
// used to authorize accounts as they are performing git operations.
type Key struct {
	Comment     string    `json:"comment"`     // comment on the key
	CreatedAt   time.Time `json:"created_at"`  // when key was created
	Email       string    `json:"email"`       // deprecated. Please refer to 'comment' instead
	Fingerprint string    `json:"fingerprint"` // a unique identifying string based on contents
	ID          string    `json:"id"`          // unique identifier of this key
	PublicKey   string    `json:"public_key"`  // full public_key as uploaded
	UpdatedAt   time.Time `json:"updated_at"`  // when key was updated
}
type KeyCreateOpts struct {
	PublicKey string `json:"public_key"` // full public_key as uploaded
}

// Create a new key.
func (s *Service) KeyCreate(o struct {
	PublicKey string `json:"public_key"` // full public_key as uploaded
}) (*Key, error) {
	var key Key
	return &key, s.Post(&key, fmt.Sprintf("/account/keys"), o)
}

// Delete an existing key
func (s *Service) KeyDelete(keyIdentity string) error {
	return s.Delete(fmt.Sprintf("/account/keys/%v", keyIdentity))
}

// Info for existing key.
func (s *Service) KeyInfo(keyIdentity string) (*Key, error) {
	var key Key
	return &key, s.Get(&key, fmt.Sprintf("/account/keys/%v", keyIdentity), nil)
}

// List existing keys.
func (s *Service) KeyList(lr *ListRange) ([]*Key, error) {
	var keyList []*Key
	return keyList, s.Get(&keyList, fmt.Sprintf("/account/keys"), lr)
}

// [Log
// drains](https://devcenter.heroku.com/articles/logging#syslog-drains)
// provide a way to forward your Heroku logs to an external syslog
// server for long-term archiving. This external service must be
// configured to receive syslog packets from Heroku, whereupon its URL
// can be added to an app using this API. Some addons will add a log
// drain when they are provisioned to an app. These drains can only be
// removed by removing the add-on.
type LogDrain struct {
	Addon *struct {
		ID string `json:"id"` // unique identifier of add-on
	} `json:"addon"` // addon that created the drain
	CreatedAt time.Time `json:"created_at"` // when log drain was created
	ID        string    `json:"id"`         // unique identifier of this log drain
	Token     string    `json:"token"`      // token associated with the log drain
	UpdatedAt time.Time `json:"updated_at"` // when log drain was updated
	URL       string    `json:"url"`        // url associated with the log drain
}
type LogDrainCreateOpts struct {
	URL string `json:"url"` // url associated with the log drain
}

// Create a new log drain.
func (s *Service) LogDrainCreate(appIdentity string, o struct {
	URL string `json:"url"` // url associated with the log drain
}) (*LogDrain, error) {
	var logDrain LogDrain
	return &logDrain, s.Post(&logDrain, fmt.Sprintf("/apps/%v/log-drains", appIdentity), o)
}

// Delete an existing log drain. Log drains added by add-ons can only be
// removed by removing the add-on.
func (s *Service) LogDrainDelete(appIdentity string, logDrainIdentity string) error {
	return s.Delete(fmt.Sprintf("/apps/%v/log-drains/%v", appIdentity, logDrainIdentity))
}

// Info for existing log drain.
func (s *Service) LogDrainInfo(appIdentity string, logDrainIdentity string) (*LogDrain, error) {
	var logDrain LogDrain
	return &logDrain, s.Get(&logDrain, fmt.Sprintf("/apps/%v/log-drains/%v", appIdentity, logDrainIdentity), nil)
}

// List existing log drains.
func (s *Service) LogDrainList(appIdentity string, lr *ListRange) ([]*LogDrain, error) {
	var logDrainList []*LogDrain
	return logDrainList, s.Get(&logDrainList, fmt.Sprintf("/apps/%v/log-drains", appIdentity), lr)
}

// A log session is a reference to the http based log stream for an app.
type LogSession struct {
	CreatedAt  time.Time `json:"created_at"`  // when log connection was created
	ID         string    `json:"id"`          // unique identifier of this log session
	LogplexURL string    `json:"logplex_url"` // URL for log streaming session
	UpdatedAt  time.Time `json:"updated_at"`  // when log session was updated
}
type LogSessionCreateOpts struct {
	Dyno   *string `json:"dyno,omitempty"`   // dyno to limit results to
	Lines  *int    `json:"lines,omitempty"`  // number of log lines to stream at once
	Source *string `json:"source,omitempty"` // log source to limit results to
	Tail   *bool   `json:"tail,omitempty"`   // whether to stream ongoing logs
}

// Create a new log session.
func (s *Service) LogSessionCreate(appIdentity string, o struct {
	Dyno   *string `json:"dyno,omitempty"`   // dyno to limit results to
	Lines  *int    `json:"lines,omitempty"`  // number of log lines to stream at once
	Source *string `json:"source,omitempty"` // log source to limit results to
	Tail   *bool   `json:"tail,omitempty"`   // whether to stream ongoing logs
}) (*LogSession, error) {
	var logSession LogSession
	return &logSession, s.Post(&logSession, fmt.Sprintf("/apps/%v/log-sessions", appIdentity), o)
}

// OAuth authorizations represent clients that a Heroku user has
// authorized to automate, customize or extend their usage of the
// platform. For more information please refer to the [Heroku OAuth
// documentation](https://devcenter.heroku.com/articles/oauth)
type OAuthAuthorization struct {
	AccessToken *struct {
		ExpiresIn *int `json:"expires_in"` // seconds until OAuth token expires; may be `null` for tokens with
		// indefinite lifetime
		ID    string `json:"id"`    // unique identifier of OAuth token
		Token string `json:"token"` // contents of the token to be used for authorization
	} `json:"access_token"` // access token for this authorization
	Client *struct {
		ID          string `json:"id"`           // unique identifier of this OAuth client
		Name        string `json:"name"`         // OAuth client name
		RedirectURI string `json:"redirect_uri"` // endpoint for redirection after authorization with OAuth client
	} `json:"client"` // identifier of the client that obtained this authorization, if any
	CreatedAt time.Time `json:"created_at"` // when OAuth authorization was created
	Grant     *struct {
		Code      string `json:"code"`       // grant code received from OAuth web application authorization
		ExpiresIn int    `json:"expires_in"` // seconds until OAuth grant expires
		ID        string `json:"id"`         // unique identifier of OAuth grant
	} `json:"grant"` // this authorization's grant
	ID           string `json:"id"` // unique identifier of OAuth authorization
	RefreshToken *struct {
		ExpiresIn *int `json:"expires_in"` // seconds until OAuth token expires; may be `null` for tokens with
		// indefinite lifetime
		ID    string `json:"id"`    // unique identifier of OAuth token
		Token string `json:"token"` // contents of the token to be used for authorization
	} `json:"refresh_token"` // refresh token for this authorization
	Scope     []string  `json:"scope"`      // The scope of access OAuth authorization allows
	UpdatedAt time.Time `json:"updated_at"` // when OAuth authorization was updated
}
type OAuthAuthorizationCreateOpts struct {
	Client      *string `json:"client,omitempty"`      // unique identifier of this OAuth client
	Description *string `json:"description,omitempty"` // human-friendly description of this OAuth authorization
	ExpiresIn   *int    `json:"expires_in,omitempty"`  // seconds until OAuth token expires; may be `null` for tokens with
	// indefinite lifetime
	Scope []string `json:"scope"` // The scope of access OAuth authorization allows
}

// Create a new OAuth authorization.
func (s *Service) OAuthAuthorizationCreate(o struct {
	Client      *string `json:"client,omitempty"`      // unique identifier of this OAuth client
	Description *string `json:"description,omitempty"` // human-friendly description of this OAuth authorization
	ExpiresIn   *int    `json:"expires_in,omitempty"`  // seconds until OAuth token expires; may be `null` for tokens with
	// indefinite lifetime
	Scope []string `json:"scope"` // The scope of access OAuth authorization allows
}) (*OAuthAuthorization, error) {
	var oauthAuthorization OAuthAuthorization
	return &oauthAuthorization, s.Post(&oauthAuthorization, fmt.Sprintf("/oauth/authorizations"), o)
}

// Delete OAuth authorization.
func (s *Service) OAuthAuthorizationDelete(oauthAuthorizationIdentity string) error {
	return s.Delete(fmt.Sprintf("/oauth/authorizations/%v", oauthAuthorizationIdentity))
}

// Info for an OAuth authorization.
func (s *Service) OAuthAuthorizationInfo(oauthAuthorizationIdentity string) (*OAuthAuthorization, error) {
	var oauthAuthorization OAuthAuthorization
	return &oauthAuthorization, s.Get(&oauthAuthorization, fmt.Sprintf("/oauth/authorizations/%v", oauthAuthorizationIdentity), nil)
}

// List OAuth authorizations.
func (s *Service) OAuthAuthorizationList(lr *ListRange) ([]*OAuthAuthorization, error) {
	var oauthAuthorizationList []*OAuthAuthorization
	return oauthAuthorizationList, s.Get(&oauthAuthorizationList, fmt.Sprintf("/oauth/authorizations"), lr)
}

// OAuth clients are applications that Heroku users can authorize to
// automate, customize or extend their usage of the platform. For more
// information please refer to the [Heroku OAuth
// documentation](https://devcenter.heroku.com/articles/oauth).
type OAuthClient struct {
	CreatedAt         time.Time `json:"created_at"`         // when OAuth client was created
	ID                string    `json:"id"`                 // unique identifier of this OAuth client
	IgnoresDelinquent *bool     `json:"ignores_delinquent"` // whether the client is still operable given a delinquent account
	Name              string    `json:"name"`               // OAuth client name
	RedirectURI       string    `json:"redirect_uri"`       // endpoint for redirection after authorization with OAuth client
	Secret            string    `json:"secret"`             // secret used to obtain OAuth authorizations under this client
	UpdatedAt         time.Time `json:"updated_at"`         // when OAuth client was updated
}
type OAuthClientCreateOpts struct {
	Name        string `json:"name"`         // OAuth client name
	RedirectURI string `json:"redirect_uri"` // endpoint for redirection after authorization with OAuth client
}

// Create a new OAuth client.
func (s *Service) OAuthClientCreate(o struct {
	Name        string `json:"name"`         // OAuth client name
	RedirectURI string `json:"redirect_uri"` // endpoint for redirection after authorization with OAuth client
}) (*OAuthClient, error) {
	var oauthClient OAuthClient
	return &oauthClient, s.Post(&oauthClient, fmt.Sprintf("/oauth/clients"), o)
}

// Delete OAuth client.
func (s *Service) OAuthClientDelete(oauthClientIdentity string) error {
	return s.Delete(fmt.Sprintf("/oauth/clients/%v", oauthClientIdentity))
}

// Info for an OAuth client
func (s *Service) OAuthClientInfo(oauthClientIdentity string) (*OAuthClient, error) {
	var oauthClient OAuthClient
	return &oauthClient, s.Get(&oauthClient, fmt.Sprintf("/oauth/clients/%v", oauthClientIdentity), nil)
}

// List OAuth clients
func (s *Service) OAuthClientList(lr *ListRange) ([]*OAuthClient, error) {
	var oauthClientList []*OAuthClient
	return oauthClientList, s.Get(&oauthClientList, fmt.Sprintf("/oauth/clients"), lr)
}

type OAuthClientUpdateOpts struct {
	Name        *string `json:"name,omitempty"`         // OAuth client name
	RedirectURI *string `json:"redirect_uri,omitempty"` // endpoint for redirection after authorization with OAuth client
}

// Update OAuth client
func (s *Service) OAuthClientUpdate(oauthClientIdentity string, o struct {
	Name        *string `json:"name,omitempty"`         // OAuth client name
	RedirectURI *string `json:"redirect_uri,omitempty"` // endpoint for redirection after authorization with OAuth client
}) (*OAuthClient, error) {
	var oauthClient OAuthClient
	return &oauthClient, s.Patch(&oauthClient, fmt.Sprintf("/oauth/clients/%v", oauthClientIdentity), o)
}

// OAuth grants are used to obtain authorizations on behalf of a user.
// For more information please refer to the [Heroku OAuth
// documentation](https://devcenter.heroku.com/articles/oauth)
type OAuthGrant struct{}

// OAuth tokens provide access for authorized clients to act on behalf
// of a Heroku user to automate, customize or extend their usage of the
// platform. For more information please refer to the [Heroku OAuth
// documentation](https://devcenter.heroku.com/articles/oauth)
type OAuthToken struct {
	AccessToken struct {
		ExpiresIn *int `json:"expires_in"` // seconds until OAuth token expires; may be `null` for tokens with
		// indefinite lifetime
		ID    string `json:"id"`    // unique identifier of OAuth token
		Token string `json:"token"` // contents of the token to be used for authorization
	} `json:"access_token"` // current access token
	Authorization struct {
		ID string `json:"id"` // unique identifier of OAuth authorization
	} `json:"authorization"` // authorization for this set of tokens
	Client *struct {
		Secret string `json:"secret"` // secret used to obtain OAuth authorizations under this client
	} `json:"client"` // OAuth client secret used to obtain token
	CreatedAt time.Time `json:"created_at"` // when OAuth token was created
	Grant     struct {
		Code string `json:"code"` // grant code received from OAuth web application authorization
		Type string `json:"type"` // type of grant requested, one of `authorization_code` or
		// `refresh_token`
	} `json:"grant"` // grant used on the underlying authorization
	ID           string `json:"id"` // unique identifier of OAuth token
	RefreshToken struct {
		ExpiresIn *int `json:"expires_in"` // seconds until OAuth token expires; may be `null` for tokens with
		// indefinite lifetime
		ID    string `json:"id"`    // unique identifier of OAuth token
		Token string `json:"token"` // contents of the token to be used for authorization
	} `json:"refresh_token"` // refresh token for this authorization
	Session struct {
		ID string `json:"id"` // unique identifier of OAuth token
	} `json:"session"` // OAuth session using this token
	UpdatedAt time.Time `json:"updated_at"` // when OAuth token was updated
	User      struct {
		ID string `json:"id"` // unique identifier of an account
	} `json:"user"` // Reference to the user associated with this token
}
type OAuthTokenCreateOpts struct {
	Client struct {
		Secret *string `json:"secret,omitempty"` // secret used to obtain OAuth authorizations under this client
	} `json:"client"`
	Grant struct {
		Code *string `json:"code,omitempty"` // grant code received from OAuth web application authorization
		Type *string `json:"type,omitempty"` // type of grant requested, one of `authorization_code` or
		// `refresh_token`
	} `json:"grant"`
	RefreshToken struct {
		Token *string `json:"token,omitempty"` // contents of the token to be used for authorization
	} `json:"refresh_token"`
}

// Create a new OAuth token.
func (s *Service) OAuthTokenCreate(o struct {
	Client struct {
		Secret *string `json:"secret,omitempty"` // secret used to obtain OAuth authorizations under this client
	} `json:"client"`
	Grant struct {
		Code *string `json:"code,omitempty"` // grant code received from OAuth web application authorization
		Type *string `json:"type,omitempty"` // type of grant requested, one of `authorization_code` or
		// `refresh_token`
	} `json:"grant"`
	RefreshToken struct {
		Token *string `json:"token,omitempty"` // contents of the token to be used for authorization
	} `json:"refresh_token"`
}) (*OAuthToken, error) {
	var oauthToken OAuthToken
	return &oauthToken, s.Post(&oauthToken, fmt.Sprintf("/oauth/tokens"), o)
}

// Organizations allow you to manage access to a shared group of
// applications across your development team.
type Organization struct {
	CreditCardCollections bool   `json:"credit_card_collections"` // whether charges incurred by the org are paid by credit card.
	Default               bool   `json:"default"`                 // whether to use this organization when none is specified
	Name                  string `json:"name"`                    // unique name of organization
	ProvisionedLicenses   bool   `json:"provisioned_licenses"`    // whether the org is provisioned licenses by salesforce.
	Role                  string `json:"role"`                    // role in the organization
}

// List organizations in which you are a member.
func (s *Service) OrganizationList(lr *ListRange) ([]*Organization, error) {
	var organizationList []*Organization
	return organizationList, s.Get(&organizationList, fmt.Sprintf("/organizations"), lr)
}

type OrganizationUpdateOpts struct {
	Default *bool `json:"default,omitempty"` // whether to use this organization when none is specified
}

// Set or unset the organization as your default organization.
func (s *Service) OrganizationUpdate(organizationIdentity string, o struct {
	Default *bool `json:"default,omitempty"` // whether to use this organization when none is specified
}) (*Organization, error) {
	var organization Organization
	return &organization, s.Patch(&organization, fmt.Sprintf("/organizations/%v", organizationIdentity), o)
}

// An organization app encapsulates the organization specific
// functionality of Heroku apps.
type OrganizationApp struct {
	ArchivedAt                   *time.Time `json:"archived_at"`                    // when app was archived
	BuildpackProvidedDescription *string    `json:"buildpack_provided_description"` // description from buildpack of app
	CreatedAt                    time.Time  `json:"created_at"`                     // when app was created
	GitURL                       string     `json:"git_url"`                        // git repo URL of app
	ID                           string     `json:"id"`                             // unique identifier of app
	Joined                       bool       `json:"joined"`                         // is the current member a collaborator on this app.
	Locked                       bool       `json:"locked"`                         // are other organization members forbidden from joining this app.
	Maintenance                  bool       `json:"maintenance"`                    // maintenance status of app
	Name                         string     `json:"name"`                           // unique name of app
	Organization                 *struct {
		Name string `json:"name"` // unique name of organization
	} `json:"organization"` // organization that owns this app
	Owner *struct {
		Email string `json:"email"` // unique email address of account
		ID    string `json:"id"`    // unique identifier of an account
	} `json:"owner"` // identity of app owner
	Region struct {
		ID   string `json:"id"`   // unique identifier of region
		Name string `json:"name"` // unique name of region
	} `json:"region"` // identity of app region
	ReleasedAt *time.Time `json:"released_at"` // when app was released
	RepoSize   *int       `json:"repo_size"`   // git repo size in bytes of app
	SlugSize   *int       `json:"slug_size"`   // slug size in bytes of app
	Stack      struct {
		ID   string `json:"id"`   // unique identifier of stack
		Name string `json:"name"` // unique name of stack
	} `json:"stack"` // identity of app stack
	UpdatedAt time.Time `json:"updated_at"` // when app was updated
	WebURL    string    `json:"web_url"`    // web URL of app
}
type OrganizationAppCreateOpts struct {
	Locked       *bool   `json:"locked,omitempty"`       // are other organization members forbidden from joining this app.
	Name         *string `json:"name,omitempty"`         // unique name of app
	Organization *string `json:"organization,omitempty"` // unique name of organization
	Personal     *bool   `json:"personal,omitempty"`     // force creation of the app in the user account even if a default org
	// is set.
	Region *string `json:"region,omitempty"` // unique name of region
	Stack  *string `json:"stack,omitempty"`  // unique name of stack
}

// Create a new app in the specified organization, in the default
// organization if unspecified,  or in personal account, if default
// organization is not set.
func (s *Service) OrganizationAppCreate(o struct {
	Locked       *bool   `json:"locked,omitempty"`       // are other organization members forbidden from joining this app.
	Name         *string `json:"name,omitempty"`         // unique name of app
	Organization *string `json:"organization,omitempty"` // unique name of organization
	Personal     *bool   `json:"personal,omitempty"`     // force creation of the app in the user account even if a default org
	// is set.
	Region *string `json:"region,omitempty"` // unique name of region
	Stack  *string `json:"stack,omitempty"`  // unique name of stack
}) (*OrganizationApp, error) {
	var organizationApp OrganizationApp
	return &organizationApp, s.Post(&organizationApp, fmt.Sprintf("/organizations/apps"), o)
}

// List apps in the default organization, or in personal account, if
// default organization is not set.
func (s *Service) OrganizationAppList(lr *ListRange) ([]*OrganizationApp, error) {
	var organizationAppList []*OrganizationApp
	return organizationAppList, s.Get(&organizationAppList, fmt.Sprintf("/organizations/apps"), lr)
}

// List organization apps.
func (s *Service) OrganizationAppListForOrganization(organizationIdentity string, lr *ListRange) ([]*OrganizationApp, error) {
	var organizationAppList []*OrganizationApp
	return organizationAppList, s.Get(&organizationAppList, fmt.Sprintf("/organizations/%v/apps", organizationIdentity), lr)
}

// Info for an organization app.
func (s *Service) OrganizationAppInfo(organizationAppIdentity string) (*OrganizationApp, error) {
	var organizationApp OrganizationApp
	return &organizationApp, s.Get(&organizationApp, fmt.Sprintf("/organizations/apps/%v", organizationAppIdentity), nil)
}

type OrganizationAppUpdateLockedOpts struct {
	Locked bool `json:"locked"` // are other organization members forbidden from joining this app.
}

// Lock or unlock an organization app.
func (s *Service) OrganizationAppUpdateLocked(organizationAppIdentity string, o struct {
	Locked bool `json:"locked"` // are other organization members forbidden from joining this app.
}) (*OrganizationApp, error) {
	var organizationApp OrganizationApp
	return &organizationApp, s.Patch(&organizationApp, fmt.Sprintf("/organizations/apps/%v", organizationAppIdentity), o)
}

type OrganizationAppTransferToAccountOpts struct {
	Owner string `json:"owner"` // unique email address of account
}

// Transfer an existing organization app to another Heroku account.
func (s *Service) OrganizationAppTransferToAccount(organizationAppIdentity string, o struct {
	Owner string `json:"owner"` // unique email address of account
}) (*OrganizationApp, error) {
	var organizationApp OrganizationApp
	return &organizationApp, s.Patch(&organizationApp, fmt.Sprintf("/organizations/apps/%v", organizationAppIdentity), o)
}

type OrganizationAppTransferToOrganizationOpts struct {
	Owner string `json:"owner"` // unique name of organization
}

// Transfer an existing organization app to another organization.
func (s *Service) OrganizationAppTransferToOrganization(organizationAppIdentity string, o struct {
	Owner string `json:"owner"` // unique name of organization
}) (*OrganizationApp, error) {
	var organizationApp OrganizationApp
	return &organizationApp, s.Patch(&organizationApp, fmt.Sprintf("/organizations/apps/%v", organizationAppIdentity), o)
}

// An organization collaborator represents an account that has been
// given access to an organization app on Heroku.
type OrganizationAppCollaborator struct {
	CreatedAt time.Time `json:"created_at"` // when collaborator was created
	ID        string    `json:"id"`         // unique identifier of collaborator
	Role      string    `json:"role"`       // role in the organization
	UpdatedAt time.Time `json:"updated_at"` // when collaborator was updated
	User      struct {
		Email string `json:"email"` // unique email address of account
		ID    string `json:"id"`    // unique identifier of an account
	} `json:"user"` // identity of collaborated account
}
type OrganizationAppCollaboratorCreateOpts struct {
	Silent *bool  `json:"silent,omitempty"` // whether to suppress email invitation when creating collaborator
	User   string `json:"user"`             // unique email address of account
}

// Create a new collaborator on an organization app. Use this endpoint
// instead of the `/apps/{app_id_or_name}/collaborator` endpoint when
// you want the collaborator to be granted [privileges]
// (https://devcenter.heroku.com/articles/org-users-access#roles)
// according to their role in the organization.
func (s *Service) OrganizationAppCollaboratorCreate(appIdentity string, o struct {
	Silent *bool  `json:"silent,omitempty"` // whether to suppress email invitation when creating collaborator
	User   string `json:"user"`             // unique email address of account
}) (*OrganizationAppCollaborator, error) {
	var organizationAppCollaborator OrganizationAppCollaborator
	return &organizationAppCollaborator, s.Post(&organizationAppCollaborator, fmt.Sprintf("/organizations/apps/%v/collaborators", appIdentity), o)
}

// Delete an existing collaborator from an organization app.
func (s *Service) OrganizationAppCollaboratorDelete(organizationAppIdentity string, organizationAppCollaboratorIdentity string) error {
	return s.Delete(fmt.Sprintf("/organizations/apps/%v/collaborators/%v", organizationAppIdentity, organizationAppCollaboratorIdentity))
}

// Info for a collaborator on an organization app.
func (s *Service) OrganizationAppCollaboratorInfo(organizationAppIdentity string, organizationAppCollaboratorIdentity string) (*OrganizationAppCollaborator, error) {
	var organizationAppCollaborator OrganizationAppCollaborator
	return &organizationAppCollaborator, s.Get(&organizationAppCollaborator, fmt.Sprintf("/organizations/apps/%v/collaborators/%v", organizationAppIdentity, organizationAppCollaboratorIdentity), nil)
}

// List collaborators on an organization app.
func (s *Service) OrganizationAppCollaboratorList(organizationAppIdentity string, lr *ListRange) ([]*OrganizationAppCollaborator, error) {
	var organizationAppCollaboratorList []*OrganizationAppCollaborator
	return organizationAppCollaboratorList, s.Get(&organizationAppCollaboratorList, fmt.Sprintf("/organizations/apps/%v/collaborators", organizationAppIdentity), lr)
}

// An organization member is an individual with access to an
// organization.
type OrganizationMember struct {
	CreatedAt time.Time `json:"created_at"` // when organization-member was created
	Email     string    `json:"email"`      // email address of the organization member
	Role      string    `json:"role"`       // role in the organization
	UpdatedAt time.Time `json:"updated_at"` // when organization-member was updated
}
type OrganizationMemberCreateOrUpdateOpts struct {
	Email string `json:"email"` // email address of the organization member
	Role  string `json:"role"`  // role in the organization
}

// Create a new organization member, or update their role.
func (s *Service) OrganizationMemberCreateOrUpdate(organizationIdentity string, o struct {
	Email string `json:"email"` // email address of the organization member
	Role  string `json:"role"`  // role in the organization
}) (*OrganizationMember, error) {
	var organizationMember OrganizationMember
	return &organizationMember, s.Put(&organizationMember, fmt.Sprintf("/organizations/%v/members", organizationIdentity), o)
}

// Remove a member from the organization.
func (s *Service) OrganizationMemberDelete(organizationIdentity string, organizationMemberIdentity string) error {
	return s.Delete(fmt.Sprintf("/organizations/%v/members/%v", organizationIdentity, organizationMemberIdentity))
}

// List members of the organization.
func (s *Service) OrganizationMemberList(organizationIdentity string, lr *ListRange) ([]*OrganizationMember, error) {
	var organizationMemberList []*OrganizationMember
	return organizationMemberList, s.Get(&organizationMemberList, fmt.Sprintf("/organizations/%v/members", organizationIdentity), lr)
}

// Plans represent different configurations of add-ons that may be added
// to apps. Endpoints under add-on services can be accessed without
// authentication.
type Plan struct {
	CreatedAt   time.Time `json:"created_at"`  // when plan was created
	Default     bool      `json:"default"`     // whether this plan is the default for its addon service
	Description string    `json:"description"` // description of plan
	ID          string    `json:"id"`          // unique identifier of this plan
	Name        string    `json:"name"`        // unique name of this plan
	Price       struct {
		Cents int    `json:"cents"` // price in cents per unit of plan
		Unit  string `json:"unit"`  // unit of price for plan
	} `json:"price"` // price
	State     string    `json:"state"`      // release status for plan
	UpdatedAt time.Time `json:"updated_at"` // when plan was updated
}

// Info for existing plan.
func (s *Service) PlanInfo(addonServiceIdentity string, planIdentity string) (*Plan, error) {
	var plan Plan
	return &plan, s.Get(&plan, fmt.Sprintf("/addon-services/%v/plans/%v", addonServiceIdentity, planIdentity), nil)
}

// List existing plans.
func (s *Service) PlanList(addonServiceIdentity string, lr *ListRange) ([]*Plan, error) {
	var planList []*Plan
	return planList, s.Get(&planList, fmt.Sprintf("/addon-services/%v/plans", addonServiceIdentity), lr)
}

// Rate Limit represents the number of request tokens each account
// holds. Requests to this endpoint do not count towards the rate limit.
type RateLimit struct {
	Remaining int `json:"remaining"` // allowed requests remaining in current interval
}

// Info for rate limits.
func (s *Service) RateLimitInfo() (*RateLimit, error) {
	var rateLimit RateLimit
	return &rateLimit, s.Get(&rateLimit, fmt.Sprintf("/account/rate-limits"), nil)
}

// A region represents a geographic location in which your application
// may run.
type Region struct {
	CreatedAt   time.Time `json:"created_at"`  // when region was created
	Description string    `json:"description"` // description of region
	ID          string    `json:"id"`          // unique identifier of region
	Name        string    `json:"name"`        // unique name of region
	UpdatedAt   time.Time `json:"updated_at"`  // when region was updated
}

// Info for existing region.
func (s *Service) RegionInfo(regionIdentity string) (*Region, error) {
	var region Region
	return &region, s.Get(&region, fmt.Sprintf("/regions/%v", regionIdentity), nil)
}

// List existing regions.
func (s *Service) RegionList(lr *ListRange) ([]*Region, error) {
	var regionList []*Region
	return regionList, s.Get(&regionList, fmt.Sprintf("/regions"), lr)
}

// A release represents a combination of code, config vars and add-ons
// for an app on Heroku.
type Release struct {
	CreatedAt   time.Time `json:"created_at"`  // when release was created
	Description string    `json:"description"` // description of changes in this release
	ID          string    `json:"id"`          // unique identifier of release
	Slug        *struct {
		ID string `json:"id"` // unique identifier of slug
	} `json:"slug"` // slug running in this release
	UpdatedAt time.Time `json:"updated_at"` // when release was updated
	User      struct {
		Email string `json:"email"` // unique email address of account
		ID    string `json:"id"`    // unique identifier of an account
	} `json:"user"` // user that created the release
	Version int `json:"version"` // unique version assigned to the release
}

// Info for existing release.
func (s *Service) ReleaseInfo(appIdentity string, releaseIdentity string) (*Release, error) {
	var release Release
	return &release, s.Get(&release, fmt.Sprintf("/apps/%v/releases/%v", appIdentity, releaseIdentity), nil)
}

// List existing releases.
func (s *Service) ReleaseList(appIdentity string, lr *ListRange) ([]*Release, error) {
	var releaseList []*Release
	return releaseList, s.Get(&releaseList, fmt.Sprintf("/apps/%v/releases", appIdentity), lr)
}

type ReleaseCreateOpts struct {
	Description *string `json:"description,omitempty"` // description of changes in this release
	Slug        string  `json:"slug"`                  // unique identifier of slug
}

// Create new release. The API cannot be used to create releases on
// Bamboo apps.
func (s *Service) ReleaseCreate(appIdentity string, o struct {
	Description *string `json:"description,omitempty"` // description of changes in this release
	Slug        string  `json:"slug"`                  // unique identifier of slug
}) (*Release, error) {
	var release Release
	return &release, s.Post(&release, fmt.Sprintf("/apps/%v/releases", appIdentity), o)
}

type ReleaseRollbackOpts struct {
	Release string `json:"release"` // unique identifier of release
}

// Rollback to an existing release.
func (s *Service) ReleaseRollback(appIdentity string, o struct {
	Release string `json:"release"` // unique identifier of release
}) (*Release, error) {
	var release Release
	return &release, s.Post(&release, fmt.Sprintf("/apps/%v/releases", appIdentity), o)
}

// A slug is a snapshot of your application code that is ready to run on
// the platform.
type Slug struct {
	Blob struct {
		Method string `json:"method"` // method to be used to interact with the slug blob
		URL    string `json:"url"`    // URL to interact with the slug blob
	} `json:"blob"` // pointer to the url where clients can fetch or store the actual
	// release binary
	BuildpackProvidedDescription *string `json:"buildpack_provided_description"` // description from buildpack of slug
	Commit                       *string `json:"commit"`                         // identification of the code with your version control system (eg: SHA
	// of the git HEAD)
	CreatedAt    time.Time         `json:"created_at"`    // when slug was created
	ID           string            `json:"id"`            // unique identifier of slug
	ProcessTypes map[string]string `json:"process_types"` // hash mapping process type names to their respective command
	Size         *int              `json:"size"`          // size of slug, in bytes
	UpdatedAt    time.Time         `json:"updated_at"`    // when slug was updated
}

// Info for existing slug.
func (s *Service) SlugInfo(appIdentity string, slugIdentity string) (*Slug, error) {
	var slug Slug
	return &slug, s.Get(&slug, fmt.Sprintf("/apps/%v/slugs/%v", appIdentity, slugIdentity), nil)
}

type SlugCreateOpts struct {
	BuildpackProvidedDescription *string `json:"buildpack_provided_description,omitempty"` // description from buildpack of slug
	Commit                       *string `json:"commit,omitempty"`                         // identification of the code with your version control system (eg: SHA
	// of the git HEAD)
	ProcessTypes map[string]string `json:"process_types"` // hash mapping process type names to their respective command
}

// Create a new slug. For more information please refer to [Deploying
// Slugs using the Platform
// API](https://devcenter.heroku.com/articles/platform-api-deploying-slug
// s?preview=1).
func (s *Service) SlugCreate(appIdentity string, o struct {
	BuildpackProvidedDescription *string `json:"buildpack_provided_description,omitempty"` // description from buildpack of slug
	Commit                       *string `json:"commit,omitempty"`                         // identification of the code with your version control system (eg: SHA
	// of the git HEAD)
	ProcessTypes map[string]string `json:"process_types"` // hash mapping process type names to their respective command
}) (*Slug, error) {
	var slug Slug
	return &slug, s.Post(&slug, fmt.Sprintf("/apps/%v/slugs", appIdentity), o)
}

// [SSL Endpoint](https://devcenter.heroku.com/articles/ssl-endpoint) is
// a public address serving custom SSL cert for HTTPS traffic to a
// Heroku app. Note that an app must have the `ssl:endpoint` addon
// installed before it can provision an SSL Endpoint using these APIs.
type SSLEndpoint struct {
	CertificateChain string    `json:"certificate_chain"` // raw contents of the public certificate chain (eg: .crt or .pem file)
	CName            string    `json:"cname"`             // canonical name record, the address to point a domain at
	CreatedAt        time.Time `json:"created_at"`        // when endpoint was created
	ID               string    `json:"id"`                // unique identifier of this SSL endpoint
	Name             string    `json:"name"`              // unique name for SSL endpoint
	UpdatedAt        time.Time `json:"updated_at"`        // when endpoint was updated
}
type SSLEndpointCreateOpts struct {
	CertificateChain string `json:"certificate_chain"`    // raw contents of the public certificate chain (eg: .crt or .pem file)
	Preprocess       *bool  `json:"preprocess,omitempty"` // allow Heroku to modify an uploaded public certificate chain if deemed
	// advantageous by adding missing intermediaries, stripping unnecessary
	// ones, etc.
	PrivateKey string `json:"private_key"` // contents of the private key (eg .key file)
}

// Create a new SSL endpoint.
func (s *Service) SSLEndpointCreate(appIdentity string, o struct {
	CertificateChain string `json:"certificate_chain"`    // raw contents of the public certificate chain (eg: .crt or .pem file)
	Preprocess       *bool  `json:"preprocess,omitempty"` // allow Heroku to modify an uploaded public certificate chain if deemed
	// advantageous by adding missing intermediaries, stripping unnecessary
	// ones, etc.
	PrivateKey string `json:"private_key"` // contents of the private key (eg .key file)
}) (*SSLEndpoint, error) {
	var sslEndpoint SSLEndpoint
	return &sslEndpoint, s.Post(&sslEndpoint, fmt.Sprintf("/apps/%v/ssl-endpoints", appIdentity), o)
}

// Delete existing SSL endpoint.
func (s *Service) SSLEndpointDelete(appIdentity string, sslEndpointIdentity string) error {
	return s.Delete(fmt.Sprintf("/apps/%v/ssl-endpoints/%v", appIdentity, sslEndpointIdentity))
}

// Info for existing SSL endpoint.
func (s *Service) SSLEndpointInfo(appIdentity string, sslEndpointIdentity string) (*SSLEndpoint, error) {
	var sslEndpoint SSLEndpoint
	return &sslEndpoint, s.Get(&sslEndpoint, fmt.Sprintf("/apps/%v/ssl-endpoints/%v", appIdentity, sslEndpointIdentity), nil)
}

// List existing SSL endpoints.
func (s *Service) SSLEndpointList(appIdentity string, lr *ListRange) ([]*SSLEndpoint, error) {
	var sslEndpointList []*SSLEndpoint
	return sslEndpointList, s.Get(&sslEndpointList, fmt.Sprintf("/apps/%v/ssl-endpoints", appIdentity), lr)
}

type SSLEndpointUpdateOpts struct {
	CertificateChain *string `json:"certificate_chain,omitempty"` // raw contents of the public certificate chain (eg: .crt or .pem file)
	Preprocess       *bool   `json:"preprocess,omitempty"`        // allow Heroku to modify an uploaded public certificate chain if deemed
	// advantageous by adding missing intermediaries, stripping unnecessary
	// ones, etc.
	PrivateKey *string `json:"private_key,omitempty"` // contents of the private key (eg .key file)
	Rollback   *bool   `json:"rollback,omitempty"`    // indicates that a rollback should be performed
}

// Update an existing SSL endpoint.
func (s *Service) SSLEndpointUpdate(appIdentity string, sslEndpointIdentity string, o struct {
	CertificateChain *string `json:"certificate_chain,omitempty"` // raw contents of the public certificate chain (eg: .crt or .pem file)
	Preprocess       *bool   `json:"preprocess,omitempty"`        // allow Heroku to modify an uploaded public certificate chain if deemed
	// advantageous by adding missing intermediaries, stripping unnecessary
	// ones, etc.
	PrivateKey *string `json:"private_key,omitempty"` // contents of the private key (eg .key file)
	Rollback   *bool   `json:"rollback,omitempty"`    // indicates that a rollback should be performed
}) (*SSLEndpoint, error) {
	var sslEndpoint SSLEndpoint
	return &sslEndpoint, s.Patch(&sslEndpoint, fmt.Sprintf("/apps/%v/ssl-endpoints/%v", appIdentity, sslEndpointIdentity), o)
}

// Stacks are the different application execution environments available
// in the Heroku platform.
type Stack struct {
	CreatedAt time.Time `json:"created_at"` // when stack was introduced
	ID        string    `json:"id"`         // unique identifier of stack
	Name      string    `json:"name"`       // unique name of stack
	State     string    `json:"state"`      // availability of this stack: beta, deprecated or public
	UpdatedAt time.Time `json:"updated_at"` // when stack was last modified
}

// Stack info.
func (s *Service) StackInfo(stackIdentity string) (*Stack, error) {
	var stack Stack
	return &stack, s.Get(&stack, fmt.Sprintf("/stacks/%v", stackIdentity), nil)
}

// List available stacks.
func (s *Service) StackList(lr *ListRange) ([]*Stack, error) {
	var stackList []*Stack
	return stackList, s.Get(&stackList, fmt.Sprintf("/stacks"), lr)
}

