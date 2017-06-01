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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
)

const (
	Version          = "v3"
	DefaultUserAgent = "heroku/" + Version + " (" + runtime.GOOS + "; " + runtime.GOARCH + ")"
	DefaultURL       = "https://api.heroku.com"
)

// Service represents your API.
type Service struct {
	client *http.Client
	URL    string
}

// NewService creates a Service using the given, if none is provided
// it uses http.DefaultClient.
func NewService(c *http.Client) *Service {
	if c == nil {
		c = http.DefaultClient
	}
	return &Service{
		client: c,
		URL:    DefaultURL,
	}
}

// NewRequest generates an HTTP request, but does not perform the request.
func (s *Service) NewRequest(ctx context.Context, method, path string, body interface{}, q interface{}) (*http.Request, error) {
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
	req, err := http.NewRequest(method, s.URL+path, rbody)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if q != nil {
		v, err := query.Values(q)
		if err != nil {
			return nil, err
		}
		query := v.Encode()
		if req.URL.RawQuery != "" && query != "" {
			req.URL.RawQuery += "&"
		}
		req.URL.RawQuery += query
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", DefaultUserAgent)
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	return req, nil
}

// Do sends a request and decodes the response into v.
func (s *Service) Do(ctx context.Context, v interface{}, method, path string, body interface{}, q interface{}, lr *ListRange) error {
	req, err := s.NewRequest(ctx, method, path, body, q)
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
func (s *Service) Get(ctx context.Context, v interface{}, path string, query interface{}, lr *ListRange) error {
	return s.Do(ctx, v, "GET", path, nil, query, lr)
}

// Patch sends a Path request and decodes the response into v.
func (s *Service) Patch(ctx context.Context, v interface{}, path string, body interface{}) error {
	return s.Do(ctx, v, "PATCH", path, body, nil, nil)
}

// Post sends a POST request and decodes the response into v.
func (s *Service) Post(ctx context.Context, v interface{}, path string, body interface{}) error {
	return s.Do(ctx, v, "POST", path, body, nil, nil)
}

// Put sends a PUT request and decodes the response into v.
func (s *Service) Put(ctx context.Context, v interface{}, path string, body interface{}) error {
	return s.Do(ctx, v, "PUT", path, body, nil, nil)
}

// Delete sends a DELETE request.
func (s *Service) Delete(ctx context.Context, v interface{}, path string) error {
	return s.Do(ctx, v, "DELETE", path, nil, nil, nil)
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
	params := make([]string, 0, 2)
	if lr.Max != 0 {
		params = append(params, fmt.Sprintf("max=%d", lr.Max))
	}
	if lr.Descending {
		params = append(params, "order=desc")
	}
	if len(params) > 0 {
		hdrval += fmt.Sprintf("; %s", strings.Join(params, ","))
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
	AllowTracking       bool      `json:"allow_tracking" url:"allow_tracking,key"` // whether to allow third party web activity tracking
	Beta                bool      `json:"beta" url:"beta,key"`                     // whether allowed to utilize beta Heroku features
	CreatedAt           time.Time `json:"created_at" url:"created_at,key"`         // when account was created
	DefaultOrganization *struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of organization
		Name string `json:"name" url:"name,key"` // unique name of organization
	} `json:"default_organization" url:"default_organization,key"` // organization selected by default
	DelinquentAt     *time.Time `json:"delinquent_at" url:"delinquent_at,key"` // when account became delinquent
	Email            string     `json:"email" url:"email,key"`                 // unique email address of account
	Federated        bool       `json:"federated" url:"federated,key"`         // whether the user is federated and belongs to an Identity Provider
	ID               string     `json:"id" url:"id,key"`                       // unique identifier of an account
	IdentityProvider *struct {
		ID           string `json:"id" url:"id,key"` // unique identifier of this identity provider
		Organization struct {
			Name string `json:"name" url:"name,key"` // unique name of organization
		} `json:"organization" url:"organization,key"`
	} `json:"identity_provider" url:"identity_provider,key"` // Identity Provider details for federated users.
	LastLogin               *time.Time `json:"last_login" url:"last_login,key"`                               // when account last authorized with Heroku
	Name                    *string    `json:"name" url:"name,key"`                                           // full name of the account owner
	SmsNumber               *string    `json:"sms_number" url:"sms_number,key"`                               // SMS number of account
	SuspendedAt             *time.Time `json:"suspended_at" url:"suspended_at,key"`                           // when account was suspended
	TwoFactorAuthentication bool       `json:"two_factor_authentication" url:"two_factor_authentication,key"` // whether two-factor auth is enabled on the account
	UpdatedAt               time.Time  `json:"updated_at" url:"updated_at,key"`                               // when account was updated
	Verified                bool       `json:"verified" url:"verified,key"`                                   // whether account has been verified with billing information
}

// Info for account.
func (s *Service) AccountInfo(ctx context.Context) (*Account, error) {
	var account Account
	return &account, s.Get(ctx, &account, fmt.Sprintf("/account"), nil, nil)
}

type AccountUpdateOpts struct {
	AllowTracking *bool   `json:"allow_tracking,omitempty" url:"allow_tracking,omitempty,key"` // whether to allow third party web activity tracking
	Beta          *bool   `json:"beta,omitempty" url:"beta,omitempty,key"`                     // whether allowed to utilize beta Heroku features
	Name          *string `json:"name,omitempty" url:"name,omitempty,key"`                     // full name of the account owner
}

// Update account.
func (s *Service) AccountUpdate(ctx context.Context, o AccountUpdateOpts) (*Account, error) {
	var account Account
	return &account, s.Patch(ctx, &account, fmt.Sprintf("/account"), o)
}

// Delete account. Note that this action cannot be undone.
func (s *Service) AccountDelete(ctx context.Context) (*Account, error) {
	var account Account
	return &account, s.Delete(ctx, &account, fmt.Sprintf("/account"))
}

// Info for account.
func (s *Service) AccountInfoByUser(ctx context.Context, accountIdentity string) (*Account, error) {
	var account Account
	return &account, s.Get(ctx, &account, fmt.Sprintf("/users/%v", accountIdentity), nil, nil)
}

type AccountUpdateByUserOpts struct {
	AllowTracking *bool   `json:"allow_tracking,omitempty" url:"allow_tracking,omitempty,key"` // whether to allow third party web activity tracking
	Beta          *bool   `json:"beta,omitempty" url:"beta,omitempty,key"`                     // whether allowed to utilize beta Heroku features
	Name          *string `json:"name,omitempty" url:"name,omitempty,key"`                     // full name of the account owner
}

// Update account.
func (s *Service) AccountUpdateByUser(ctx context.Context, accountIdentity string, o AccountUpdateByUserOpts) (*Account, error) {
	var account Account
	return &account, s.Patch(ctx, &account, fmt.Sprintf("/users/%v", accountIdentity), o)
}

// Delete account. Note that this action cannot be undone.
func (s *Service) AccountDeleteByUser(ctx context.Context, accountIdentity string) (*Account, error) {
	var account Account
	return &account, s.Delete(ctx, &account, fmt.Sprintf("/users/%v", accountIdentity))
}

// An account feature represents a Heroku labs capability that can be
// enabled or disabled for an account on Heroku.
type AccountFeature struct {
	CreatedAt     time.Time `json:"created_at" url:"created_at,key"`         // when account feature was created
	Description   string    `json:"description" url:"description,key"`       // description of account feature
	DisplayName   string    `json:"display_name" url:"display_name,key"`     // user readable feature name
	DocURL        string    `json:"doc_url" url:"doc_url,key"`               // documentation URL of account feature
	Enabled       bool      `json:"enabled" url:"enabled,key"`               // whether or not account feature has been enabled
	FeedbackEmail string    `json:"feedback_email" url:"feedback_email,key"` // e-mail to send feedback about the feature
	ID            string    `json:"id" url:"id,key"`                         // unique identifier of account feature
	Name          string    `json:"name" url:"name,key"`                     // unique name of account feature
	State         string    `json:"state" url:"state,key"`                   // state of account feature
	UpdatedAt     time.Time `json:"updated_at" url:"updated_at,key"`         // when account feature was updated
}

// Info for an existing account feature.
func (s *Service) AccountFeatureInfo(ctx context.Context, accountFeatureIdentity string) (*AccountFeature, error) {
	var accountFeature AccountFeature
	return &accountFeature, s.Get(ctx, &accountFeature, fmt.Sprintf("/account/features/%v", accountFeatureIdentity), nil, nil)
}

type AccountFeatureListResult []AccountFeature

// List existing account features.
func (s *Service) AccountFeatureList(ctx context.Context, lr *ListRange) (AccountFeatureListResult, error) {
	var accountFeature AccountFeatureListResult
	return accountFeature, s.Get(ctx, &accountFeature, fmt.Sprintf("/account/features"), nil, lr)
}

type AccountFeatureUpdateOpts struct {
	Enabled bool `json:"enabled" url:"enabled,key"` // whether or not account feature has been enabled
}

// Update an existing account feature.
func (s *Service) AccountFeatureUpdate(ctx context.Context, accountFeatureIdentity string, o AccountFeatureUpdateOpts) (*AccountFeature, error) {
	var accountFeature AccountFeature
	return &accountFeature, s.Patch(ctx, &accountFeature, fmt.Sprintf("/account/features/%v", accountFeatureIdentity), o)
}

// Add-ons represent add-ons that have been provisioned and attached to
// one or more apps.
type AddOn struct {
	Actions      []struct{} `json:"actions" url:"actions,key"` // provider actions for this specific add-on
	AddonService struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of this add-on-service
		Name string `json:"name" url:"name,key"` // unique name of this add-on-service
	} `json:"addon_service" url:"addon_service,key"` // identity of add-on service
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // billing application associated with this add-on
	ConfigVars []string  `json:"config_vars" url:"config_vars,key"` // config vars exposed to the owning app by this add-on
	CreatedAt  time.Time `json:"created_at" url:"created_at,key"`   // when add-on was created
	ID         string    `json:"id" url:"id,key"`                   // unique identifier of add-on
	Name       string    `json:"name" url:"name,key"`               // globally unique name of the add-on
	Plan       struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of this plan
		Name string `json:"name" url:"name,key"` // unique name of this plan
	} `json:"plan" url:"plan,key"` // identity of add-on plan
	ProviderID string    `json:"provider_id" url:"provider_id,key"` // id of this add-on with its provider
	State      string    `json:"state" url:"state,key"`             // state in the add-on's lifecycle
	UpdatedAt  time.Time `json:"updated_at" url:"updated_at,key"`   // when add-on was updated
	WebURL     *string   `json:"web_url" url:"web_url,key"`         // URL for logging into web interface of add-on (e.g. a dashboard)
}
type AddOnListResult []AddOn

// List all existing add-ons.
func (s *Service) AddOnList(ctx context.Context, lr *ListRange) (AddOnListResult, error) {
	var addOn AddOnListResult
	return addOn, s.Get(ctx, &addOn, fmt.Sprintf("/addons"), nil, lr)
}

// Info for an existing add-on.
func (s *Service) AddOnInfo(ctx context.Context, addOnIdentity string) (*AddOn, error) {
	var addOn AddOn
	return &addOn, s.Get(ctx, &addOn, fmt.Sprintf("/addons/%v", addOnIdentity), nil, nil)
}

type AddOnCreateOpts struct {
	Attachment *struct{}          `json:"attachment,omitempty" url:"attachment,omitempty,key"` // name for add-on's initial attachment
	Config     *map[string]string `json:"config,omitempty" url:"config,omitempty,key"`         // custom add-on provisioning options
	Plan       string             `json:"plan" url:"plan,key"`                                 // unique identifier of this plan
}

// Create a new add-on.
func (s *Service) AddOnCreate(ctx context.Context, appIdentity string, o AddOnCreateOpts) (*AddOn, error) {
	var addOn AddOn
	return &addOn, s.Post(ctx, &addOn, fmt.Sprintf("/apps/%v/addons", appIdentity), o)
}

// Delete an existing add-on.
func (s *Service) AddOnDelete(ctx context.Context, appIdentity string, addOnIdentity string) (*AddOn, error) {
	var addOn AddOn
	return &addOn, s.Delete(ctx, &addOn, fmt.Sprintf("/apps/%v/addons/%v", appIdentity, addOnIdentity))
}

// Info for an existing add-on.
func (s *Service) AddOnInfoByApp(ctx context.Context, appIdentity string, addOnIdentity string) (*AddOn, error) {
	var addOn AddOn
	return &addOn, s.Get(ctx, &addOn, fmt.Sprintf("/apps/%v/addons/%v", appIdentity, addOnIdentity), nil, nil)
}

type AddOnListByAppResult []AddOn

// List existing add-ons for an app.
func (s *Service) AddOnListByApp(ctx context.Context, appIdentity string, lr *ListRange) (AddOnListByAppResult, error) {
	var addOn AddOnListByAppResult
	return addOn, s.Get(ctx, &addOn, fmt.Sprintf("/apps/%v/addons", appIdentity), nil, lr)
}

type AddOnUpdateOpts struct {
	Plan string `json:"plan" url:"plan,key"` // unique identifier of this plan
}

// Change add-on plan. Some add-ons may not support changing plans. In
// that case, an error will be returned.
func (s *Service) AddOnUpdate(ctx context.Context, appIdentity string, addOnIdentity string, o AddOnUpdateOpts) (*AddOn, error) {
	var addOn AddOn
	return &addOn, s.Patch(ctx, &addOn, fmt.Sprintf("/apps/%v/addons/%v", appIdentity, addOnIdentity), o)
}

type AddOnListByUserResult []AddOn

// List all existing add-ons a user has access to
func (s *Service) AddOnListByUser(ctx context.Context, accountIdentity string, lr *ListRange) (AddOnListByUserResult, error) {
	var addOn AddOnListByUserResult
	return addOn, s.Get(ctx, &addOn, fmt.Sprintf("/users/%v/addons", accountIdentity), nil, lr)
}

type AddOnListByTeamResult []AddOn

// List add-ons used across all Team apps
func (s *Service) AddOnListByTeam(ctx context.Context, teamIdentity string, lr *ListRange) (AddOnListByTeamResult, error) {
	var addOn AddOnListByTeamResult
	return addOn, s.Get(ctx, &addOn, fmt.Sprintf("/teams/%v/addons", teamIdentity), nil, lr)
}

// Add-on Actions are lifecycle operations for add-on provisioning and
// deprovisioning. They allow whitelisted add-on providers to
// (de)provision add-ons in the background and then report back when
// (de)provisioning is complete.
type AddOnAction struct{}
type AddOnActionProvisionResult struct {
	Actions      []struct{} `json:"actions" url:"actions,key"` // provider actions for this specific add-on
	AddonService struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of this add-on-service
		Name string `json:"name" url:"name,key"` // unique name of this add-on-service
	} `json:"addon_service" url:"addon_service,key"` // identity of add-on service
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // billing application associated with this add-on
	ConfigVars []string  `json:"config_vars" url:"config_vars,key"` // config vars exposed to the owning app by this add-on
	CreatedAt  time.Time `json:"created_at" url:"created_at,key"`   // when add-on was created
	ID         string    `json:"id" url:"id,key"`                   // unique identifier of add-on
	Name       string    `json:"name" url:"name,key"`               // globally unique name of the add-on
	Plan       struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of this plan
		Name string `json:"name" url:"name,key"` // unique name of this plan
	} `json:"plan" url:"plan,key"` // identity of add-on plan
	ProviderID string    `json:"provider_id" url:"provider_id,key"` // id of this add-on with its provider
	State      string    `json:"state" url:"state,key"`             // state in the add-on's lifecycle
	UpdatedAt  time.Time `json:"updated_at" url:"updated_at,key"`   // when add-on was updated
	WebURL     *string   `json:"web_url" url:"web_url,key"`         // URL for logging into web interface of add-on (e.g. a dashboard)
}

// Mark an add-on as provisioned for use.
func (s *Service) AddOnActionProvision(ctx context.Context, addOnIdentity string) (*AddOnActionProvisionResult, error) {
	var addOnAction AddOnActionProvisionResult
	return &addOnAction, s.Post(ctx, &addOnAction, fmt.Sprintf("/addons/%v/actions/provision", addOnIdentity), nil)
}

type AddOnActionDeprovisionResult struct {
	Actions      []struct{} `json:"actions" url:"actions,key"` // provider actions for this specific add-on
	AddonService struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of this add-on-service
		Name string `json:"name" url:"name,key"` // unique name of this add-on-service
	} `json:"addon_service" url:"addon_service,key"` // identity of add-on service
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // billing application associated with this add-on
	ConfigVars []string  `json:"config_vars" url:"config_vars,key"` // config vars exposed to the owning app by this add-on
	CreatedAt  time.Time `json:"created_at" url:"created_at,key"`   // when add-on was created
	ID         string    `json:"id" url:"id,key"`                   // unique identifier of add-on
	Name       string    `json:"name" url:"name,key"`               // globally unique name of the add-on
	Plan       struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of this plan
		Name string `json:"name" url:"name,key"` // unique name of this plan
	} `json:"plan" url:"plan,key"` // identity of add-on plan
	ProviderID string    `json:"provider_id" url:"provider_id,key"` // id of this add-on with its provider
	State      string    `json:"state" url:"state,key"`             // state in the add-on's lifecycle
	UpdatedAt  time.Time `json:"updated_at" url:"updated_at,key"`   // when add-on was updated
	WebURL     *string   `json:"web_url" url:"web_url,key"`         // URL for logging into web interface of add-on (e.g. a dashboard)
}

// Mark an add-on as deprovisioned.
func (s *Service) AddOnActionDeprovision(ctx context.Context, addOnIdentity string) (*AddOnActionDeprovisionResult, error) {
	var addOnAction AddOnActionDeprovisionResult
	return &addOnAction, s.Post(ctx, &addOnAction, fmt.Sprintf("/addons/%v/actions/deprovision", addOnIdentity), nil)
}

// An add-on attachment represents a connection between an app and an
// add-on that it has been given access to.
type AddOnAttachment struct {
	Addon struct {
		App struct {
			ID   string `json:"id" url:"id,key"`     // unique identifier of app
			Name string `json:"name" url:"name,key"` // unique name of app
		} `json:"app" url:"app,key"` // billing application associated with this add-on
		ID   string `json:"id" url:"id,key"`     // unique identifier of add-on
		Name string `json:"name" url:"name,key"` // globally unique name of the add-on
		Plan struct {
			ID   string `json:"id" url:"id,key"`     // unique identifier of this plan
			Name string `json:"name" url:"name,key"` // unique name of this plan
		} `json:"plan" url:"plan,key"` // identity of add-on plan
	} `json:"addon" url:"addon,key"` // identity of add-on
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // application that is attached to add-on
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when add-on attachment was created
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of this add-on attachment
	Name      string    `json:"name" url:"name,key"`             // unique name for this add-on attachment to this app
	Namespace *string   `json:"namespace" url:"namespace,key"`   // attachment namespace
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when add-on attachment was updated
	WebURL    *string   `json:"web_url" url:"web_url,key"`       // URL for logging into web interface of add-on in attached app context
}
type AddOnAttachmentCreateOpts struct {
	Addon string `json:"addon" url:"addon,key"`                     // unique identifier of add-on
	App   string `json:"app" url:"app,key"`                         // unique identifier of app
	Force *bool  `json:"force,omitempty" url:"force,omitempty,key"` // whether or not to allow existing attachment with same name to be
	// replaced
	Name      *string `json:"name,omitempty" url:"name,omitempty,key"`           // unique name for this add-on attachment to this app
	Namespace *string `json:"namespace,omitempty" url:"namespace,omitempty,key"` // attachment namespace
}

// Create a new add-on attachment.
func (s *Service) AddOnAttachmentCreate(ctx context.Context, o AddOnAttachmentCreateOpts) (*AddOnAttachment, error) {
	var addOnAttachment AddOnAttachment
	return &addOnAttachment, s.Post(ctx, &addOnAttachment, fmt.Sprintf("/addon-attachments"), o)
}

// Delete an existing add-on attachment.
func (s *Service) AddOnAttachmentDelete(ctx context.Context, addOnAttachmentIdentity string) (*AddOnAttachment, error) {
	var addOnAttachment AddOnAttachment
	return &addOnAttachment, s.Delete(ctx, &addOnAttachment, fmt.Sprintf("/addon-attachments/%v", addOnAttachmentIdentity))
}

// Info for existing add-on attachment.
func (s *Service) AddOnAttachmentInfo(ctx context.Context, addOnAttachmentIdentity string) (*AddOnAttachment, error) {
	var addOnAttachment AddOnAttachment
	return &addOnAttachment, s.Get(ctx, &addOnAttachment, fmt.Sprintf("/addon-attachments/%v", addOnAttachmentIdentity), nil, nil)
}

type AddOnAttachmentListResult []AddOnAttachment

// List existing add-on attachments.
func (s *Service) AddOnAttachmentList(ctx context.Context, lr *ListRange) (AddOnAttachmentListResult, error) {
	var addOnAttachment AddOnAttachmentListResult
	return addOnAttachment, s.Get(ctx, &addOnAttachment, fmt.Sprintf("/addon-attachments"), nil, lr)
}

type AddOnAttachmentListByAddOnResult []AddOnAttachment

// List existing add-on attachments for an add-on.
func (s *Service) AddOnAttachmentListByAddOn(ctx context.Context, addOnIdentity string, lr *ListRange) (AddOnAttachmentListByAddOnResult, error) {
	var addOnAttachment AddOnAttachmentListByAddOnResult
	return addOnAttachment, s.Get(ctx, &addOnAttachment, fmt.Sprintf("/addons/%v/addon-attachments", addOnIdentity), nil, lr)
}

type AddOnAttachmentListByAppResult []AddOnAttachment

// List existing add-on attachments for an app.
func (s *Service) AddOnAttachmentListByApp(ctx context.Context, appIdentity string, lr *ListRange) (AddOnAttachmentListByAppResult, error) {
	var addOnAttachment AddOnAttachmentListByAppResult
	return addOnAttachment, s.Get(ctx, &addOnAttachment, fmt.Sprintf("/apps/%v/addon-attachments", appIdentity), nil, lr)
}

// Info for existing add-on attachment for an app.
func (s *Service) AddOnAttachmentInfoByApp(ctx context.Context, appIdentity string, addOnAttachmentScopedIdentity string) (*AddOnAttachment, error) {
	var addOnAttachment AddOnAttachment
	return &addOnAttachment, s.Get(ctx, &addOnAttachment, fmt.Sprintf("/apps/%v/addon-attachments/%v", appIdentity, addOnAttachmentScopedIdentity), nil, nil)
}

// Configuration of an Add-on
type AddOnConfig struct {
	Name  string  `json:"name" url:"name,key"`   // unique name of the config
	Value *string `json:"value" url:"value,key"` // value of the config
}
type AddOnConfigListResult []AddOnConfig

// Get an add-on's config. Accessible by customers with access and by
// the add-on partner providing this add-on.
func (s *Service) AddOnConfigList(ctx context.Context, addOnIdentity string, lr *ListRange) (AddOnConfigListResult, error) {
	var addOnConfig AddOnConfigListResult
	return addOnConfig, s.Get(ctx, &addOnConfig, fmt.Sprintf("/addons/%v/config", addOnIdentity), nil, lr)
}

type AddOnConfigUpdateOpts struct {
	Config *[]*struct {
		Name  *string `json:"name,omitempty" url:"name,omitempty,key"`   // unique name of the config
		Value *string `json:"value,omitempty" url:"value,omitempty,key"` // value of the config
	} `json:"config,omitempty" url:"config,omitempty,key"`
}
type AddOnConfigUpdateResult []AddOnConfig

// Update an add-on's config. Can only be accessed by the add-on partner
// providing this add-on.
func (s *Service) AddOnConfigUpdate(ctx context.Context, addOnIdentity string, o AddOnConfigUpdateOpts) (AddOnConfigUpdateResult, error) {
	var addOnConfig AddOnConfigUpdateResult
	return addOnConfig, s.Patch(ctx, &addOnConfig, fmt.Sprintf("/addons/%v/config", addOnIdentity), o)
}

// Add-on Plan Actions are Provider functionality for specific add-on
// installations
type AddOnPlanAction struct {
	Action        string `json:"action" url:"action,key"`                 // identifier of the action to take that is sent via SSO
	ID            string `json:"id" url:"id,key"`                         // a unique identifier
	Label         string `json:"label" url:"label,key"`                   // the display text shown in Dashboard
	RequiresOwner bool   `json:"requires_owner" url:"requires_owner,key"` // if the action requires the user to own the app
	URL           string `json:"url" url:"url,key"`                       // absolute URL to use instead of an action
}

// Add-on region capabilities represent the relationship between an
// Add-on Service and a specific Region. Only Beta and GA add-ons are
// returned by these endpoints.
type AddOnRegionCapability struct {
	AddonService struct {
		CliPluginName                 *string   `json:"cli_plugin_name" url:"cli_plugin_name,key"`                                 // npm package name of the add-on service's Heroku CLI plugin
		CreatedAt                     time.Time `json:"created_at" url:"created_at,key"`                                           // when add-on-service was created
		HumanName                     string    `json:"human_name" url:"human_name,key"`                                           // human-readable name of the add-on service provider
		ID                            string    `json:"id" url:"id,key"`                                                           // unique identifier of this add-on-service
		Name                          string    `json:"name" url:"name,key"`                                                       // unique name of this add-on-service
		State                         string    `json:"state" url:"state,key"`                                                     // release status for add-on service
		SupportsMultipleInstallations bool      `json:"supports_multiple_installations" url:"supports_multiple_installations,key"` // whether or not apps can have access to more than one instance of this
		// add-on at the same time
		SupportsSharing bool `json:"supports_sharing" url:"supports_sharing,key"` // whether or not apps can have access to add-ons billed to a different
		// app
		UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when add-on-service was updated
	} `json:"addon_service" url:"addon_service,key"` // Add-on services represent add-ons that may be provisioned for apps.
	// Endpoints under add-on services can be accessed without
	// authentication.
	ID     string `json:"id" url:"id,key"` // unique identifier of this add-on-region-capability
	Region struct {
		Country        string    `json:"country" url:"country,key"`                 // country where the region exists
		CreatedAt      time.Time `json:"created_at" url:"created_at,key"`           // when region was created
		Description    string    `json:"description" url:"description,key"`         // description of region
		ID             string    `json:"id" url:"id,key"`                           // unique identifier of region
		Locale         string    `json:"locale" url:"locale,key"`                   // area in the country where the region exists
		Name           string    `json:"name" url:"name,key"`                       // unique name of region
		PrivateCapable bool      `json:"private_capable" url:"private_capable,key"` // whether or not region is available for creating a Private Space
		Provider       struct {
			Name   string `json:"name" url:"name,key"`     // name of provider
			Region string `json:"region" url:"region,key"` // region name used by provider
		} `json:"provider" url:"provider,key"` // provider of underlying substrate
		UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when region was updated
	} `json:"region" url:"region,key"` // A region represents a geographic location in which your application
	// may run.
	SupportsPrivateNetworking bool `json:"supports_private_networking" url:"supports_private_networking,key"` // whether the add-on can be installed to a Space
}
type AddOnRegionCapabilityListResult []AddOnRegionCapability

// List all existing add-on region capabilities.
func (s *Service) AddOnRegionCapabilityList(ctx context.Context, lr *ListRange) (AddOnRegionCapabilityListResult, error) {
	var addOnRegionCapability AddOnRegionCapabilityListResult
	return addOnRegionCapability, s.Get(ctx, &addOnRegionCapability, fmt.Sprintf("/addon-region-capabilities"), nil, lr)
}

type AddOnRegionCapabilityListByAddOnServiceResult []AddOnRegionCapability

// List existing add-on region capabilities for an add-on-service
func (s *Service) AddOnRegionCapabilityListByAddOnService(ctx context.Context, addOnServiceIdentity string, lr *ListRange) (AddOnRegionCapabilityListByAddOnServiceResult, error) {
	var addOnRegionCapability AddOnRegionCapabilityListByAddOnServiceResult
	return addOnRegionCapability, s.Get(ctx, &addOnRegionCapability, fmt.Sprintf("/addon-services/%v/region-capabilities", addOnServiceIdentity), nil, lr)
}

type AddOnRegionCapabilityListByRegionResult []AddOnRegionCapability

// List existing add-on region capabilities for a region.
func (s *Service) AddOnRegionCapabilityListByRegion(ctx context.Context, regionIdentity string, lr *ListRange) (AddOnRegionCapabilityListByRegionResult, error) {
	var addOnRegionCapability AddOnRegionCapabilityListByRegionResult
	return addOnRegionCapability, s.Get(ctx, &addOnRegionCapability, fmt.Sprintf("/regions/%v/addon-region-capabilities", regionIdentity), nil, lr)
}

// Add-on services represent add-ons that may be provisioned for apps.
// Endpoints under add-on services can be accessed without
// authentication.
type AddOnService struct {
	CliPluginName                 *string   `json:"cli_plugin_name" url:"cli_plugin_name,key"`                                 // npm package name of the add-on service's Heroku CLI plugin
	CreatedAt                     time.Time `json:"created_at" url:"created_at,key"`                                           // when add-on-service was created
	HumanName                     string    `json:"human_name" url:"human_name,key"`                                           // human-readable name of the add-on service provider
	ID                            string    `json:"id" url:"id,key"`                                                           // unique identifier of this add-on-service
	Name                          string    `json:"name" url:"name,key"`                                                       // unique name of this add-on-service
	State                         string    `json:"state" url:"state,key"`                                                     // release status for add-on service
	SupportsMultipleInstallations bool      `json:"supports_multiple_installations" url:"supports_multiple_installations,key"` // whether or not apps can have access to more than one instance of this
	// add-on at the same time
	SupportsSharing bool `json:"supports_sharing" url:"supports_sharing,key"` // whether or not apps can have access to add-ons billed to a different
	// app
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when add-on-service was updated
}

// Info for existing add-on-service.
func (s *Service) AddOnServiceInfo(ctx context.Context, addOnServiceIdentity string) (*AddOnService, error) {
	var addOnService AddOnService
	return &addOnService, s.Get(ctx, &addOnService, fmt.Sprintf("/addon-services/%v", addOnServiceIdentity), nil, nil)
}

type AddOnServiceListResult []AddOnService

// List existing add-on-services.
func (s *Service) AddOnServiceList(ctx context.Context, lr *ListRange) (AddOnServiceListResult, error) {
	var addOnService AddOnServiceListResult
	return addOnService, s.Get(ctx, &addOnService, fmt.Sprintf("/addon-services"), nil, lr)
}

// An app represents the program that you would like to deploy and run
// on Heroku.
type App struct {
	ArchivedAt *time.Time `json:"archived_at" url:"archived_at,key"` // when app was archived
	BuildStack struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of stack
		Name string `json:"name" url:"name,key"` // unique name of stack
	} `json:"build_stack" url:"build_stack,key"` // identity of the stack that will be used for new builds
	BuildpackProvidedDescription *string   `json:"buildpack_provided_description" url:"buildpack_provided_description,key"` // description from buildpack of app
	CreatedAt                    time.Time `json:"created_at" url:"created_at,key"`                                         // when app was created
	GitURL                       string    `json:"git_url" url:"git_url,key"`                                               // git repo URL of app
	ID                           string    `json:"id" url:"id,key"`                                                         // unique identifier of app
	Maintenance                  bool      `json:"maintenance" url:"maintenance,key"`                                       // maintenance status of app
	Name                         string    `json:"name" url:"name,key"`                                                     // unique name of app
	Organization                 *struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of organization
		Name string `json:"name" url:"name,key"` // unique name of organization
	} `json:"organization" url:"organization,key"` // identity of organization
	Owner struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"owner" url:"owner,key"` // identity of app owner
	Region struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of region
		Name string `json:"name" url:"name,key"` // unique name of region
	} `json:"region" url:"region,key"` // identity of app region
	ReleasedAt *time.Time `json:"released_at" url:"released_at,key"` // when app was released
	RepoSize   *int       `json:"repo_size" url:"repo_size,key"`     // git repo size in bytes of app
	SlugSize   *int       `json:"slug_size" url:"slug_size,key"`     // slug size in bytes of app
	Space      *struct {
		ID     string `json:"id" url:"id,key"`         // unique identifier of space
		Name   string `json:"name" url:"name,key"`     // unique name of space
		Shield bool   `json:"shield" url:"shield,key"` // true if this space has shield enabled
	} `json:"space" url:"space,key"` // identity of space
	Stack struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of stack
		Name string `json:"name" url:"name,key"` // unique name of stack
	} `json:"stack" url:"stack,key"` // identity of app stack
	Team *struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of team
		Name string `json:"name" url:"name,key"` // unique name of team
	} `json:"team" url:"team,key"` // identity of team
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when app was updated
	WebURL    string    `json:"web_url" url:"web_url,key"`       // web URL of app
}
type AppCreateOpts struct {
	Name   *string `json:"name,omitempty" url:"name,omitempty,key"`     // unique name of app
	Region *string `json:"region,omitempty" url:"region,omitempty,key"` // unique identifier of region
	Stack  *string `json:"stack,omitempty" url:"stack,omitempty,key"`   // unique name of stack
}

// Create a new app.
func (s *Service) AppCreate(ctx context.Context, o AppCreateOpts) (*App, error) {
	var app App
	return &app, s.Post(ctx, &app, fmt.Sprintf("/apps"), o)
}

// Delete an existing app.
func (s *Service) AppDelete(ctx context.Context, appIdentity string) (*App, error) {
	var app App
	return &app, s.Delete(ctx, &app, fmt.Sprintf("/apps/%v", appIdentity))
}

// Info for existing app.
func (s *Service) AppInfo(ctx context.Context, appIdentity string) (*App, error) {
	var app App
	return &app, s.Get(ctx, &app, fmt.Sprintf("/apps/%v", appIdentity), nil, nil)
}

type AppListResult []App

// List existing apps.
func (s *Service) AppList(ctx context.Context, lr *ListRange) (AppListResult, error) {
	var app AppListResult
	return app, s.Get(ctx, &app, fmt.Sprintf("/apps"), nil, lr)
}

type AppListOwnedAndCollaboratedResult []App

// List owned and collaborated apps (excludes team apps).
func (s *Service) AppListOwnedAndCollaborated(ctx context.Context, accountIdentity string, lr *ListRange) (AppListOwnedAndCollaboratedResult, error) {
	var app AppListOwnedAndCollaboratedResult
	return app, s.Get(ctx, &app, fmt.Sprintf("/users/%v/apps", accountIdentity), nil, lr)
}

type AppUpdateOpts struct {
	BuildStack  *string `json:"build_stack,omitempty" url:"build_stack,omitempty,key"` // unique name of stack
	Maintenance *bool   `json:"maintenance,omitempty" url:"maintenance,omitempty,key"` // maintenance status of app
	Name        *string `json:"name,omitempty" url:"name,omitempty,key"`               // unique name of app
}

// Update an existing app.
func (s *Service) AppUpdate(ctx context.Context, appIdentity string, o AppUpdateOpts) (*App, error) {
	var app App
	return &app, s.Patch(ctx, &app, fmt.Sprintf("/apps/%v", appIdentity), o)
}

// An app feature represents a Heroku labs capability that can be
// enabled or disabled for an app on Heroku.
type AppFeature struct {
	CreatedAt     time.Time `json:"created_at" url:"created_at,key"`         // when app feature was created
	Description   string    `json:"description" url:"description,key"`       // description of app feature
	DisplayName   string    `json:"display_name" url:"display_name,key"`     // user readable feature name
	DocURL        string    `json:"doc_url" url:"doc_url,key"`               // documentation URL of app feature
	Enabled       bool      `json:"enabled" url:"enabled,key"`               // whether or not app feature has been enabled
	FeedbackEmail string    `json:"feedback_email" url:"feedback_email,key"` // e-mail to send feedback about the feature
	ID            string    `json:"id" url:"id,key"`                         // unique identifier of app feature
	Name          string    `json:"name" url:"name,key"`                     // unique name of app feature
	State         string    `json:"state" url:"state,key"`                   // state of app feature
	UpdatedAt     time.Time `json:"updated_at" url:"updated_at,key"`         // when app feature was updated
}

// Info for an existing app feature.
func (s *Service) AppFeatureInfo(ctx context.Context, appIdentity string, appFeatureIdentity string) (*AppFeature, error) {
	var appFeature AppFeature
	return &appFeature, s.Get(ctx, &appFeature, fmt.Sprintf("/apps/%v/features/%v", appIdentity, appFeatureIdentity), nil, nil)
}

type AppFeatureListResult []AppFeature

// List existing app features.
func (s *Service) AppFeatureList(ctx context.Context, appIdentity string, lr *ListRange) (AppFeatureListResult, error) {
	var appFeature AppFeatureListResult
	return appFeature, s.Get(ctx, &appFeature, fmt.Sprintf("/apps/%v/features", appIdentity), nil, lr)
}

type AppFeatureUpdateOpts struct {
	Enabled bool `json:"enabled" url:"enabled,key"` // whether or not app feature has been enabled
}

// Update an existing app feature.
func (s *Service) AppFeatureUpdate(ctx context.Context, appIdentity string, appFeatureIdentity string, o AppFeatureUpdateOpts) (*AppFeature, error) {
	var appFeature AppFeature
	return &appFeature, s.Patch(ctx, &appFeature, fmt.Sprintf("/apps/%v/features/%v", appIdentity, appFeatureIdentity), o)
}

// App formation set describes the combination of process types with
// their quantities and sizes as well as application process tier
type AppFormationSet struct {
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // app being described by the formation-set
	Description string    `json:"description" url:"description,key"`   // a string representation of the formation set
	ProcessTier string    `json:"process_tier" url:"process_tier,key"` // application process tier
	UpdatedAt   time.Time `json:"updated_at" url:"updated_at,key"`     // last time fomation-set was updated
}

// An app setup represents an app on Heroku that is setup using an
// environment, addons, and scripts described in an app.json manifest
// file.
type AppSetup struct {
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // identity of app
	Build *struct {
		ID              string `json:"id" url:"id,key"`                               // unique identifier of build
		OutputStreamURL string `json:"output_stream_url" url:"output_stream_url,key"` // Build process output will be available from this URL as a stream. The
		// stream is available as either `text/plain` or `text/event-stream`.
		// Clients should be prepared to handle disconnects and can resume the
		// stream by sending a `Range` header (for `text/plain`) or a
		// `Last-Event-Id` header (for `text/event-stream`).
		Status string `json:"status" url:"status,key"` // status of build
	} `json:"build" url:"build,key"` // identity and status of build
	CreatedAt      time.Time `json:"created_at" url:"created_at,key"`           // when app setup was created
	FailureMessage *string   `json:"failure_message" url:"failure_message,key"` // reason that app setup has failed
	ID             string    `json:"id" url:"id,key"`                           // unique identifier of app setup
	ManifestErrors []string  `json:"manifest_errors" url:"manifest_errors,key"` // errors associated with invalid app.json manifest file
	Postdeploy     *struct {
		ExitCode int    `json:"exit_code" url:"exit_code,key"` // The exit code of the postdeploy script
		Output   string `json:"output" url:"output,key"`       // output of the postdeploy script
	} `json:"postdeploy" url:"postdeploy,key"` // result of postdeploy script
	ResolvedSuccessURL *string   `json:"resolved_success_url" url:"resolved_success_url,key"` // fully qualified success url
	Status             string    `json:"status" url:"status,key"`                             // the overall status of app setup
	UpdatedAt          time.Time `json:"updated_at" url:"updated_at,key"`                     // when app setup was updated
}
type AppSetupCreateOpts struct {
	App *struct {
		Locked       *bool   `json:"locked,omitempty" url:"locked,omitempty,key"`             // are other organization members forbidden from joining this app.
		Name         *string `json:"name,omitempty" url:"name,omitempty,key"`                 // unique name of app
		Organization *string `json:"organization,omitempty" url:"organization,omitempty,key"` // unique name of organization
		Personal     *bool   `json:"personal,omitempty" url:"personal,omitempty,key"`         // force creation of the app in the user account even if a default org
		// is set.
		Region *string `json:"region,omitempty" url:"region,omitempty,key"` // unique name of region
		Space  *string `json:"space,omitempty" url:"space,omitempty,key"`   // unique name of space
		Stack  *string `json:"stack,omitempty" url:"stack,omitempty,key"`   // unique name of stack
	} `json:"app,omitempty" url:"app,omitempty,key"` // optional parameters for created app
	Overrides *struct {
		Buildpacks *[]*struct {
			URL *string `json:"url,omitempty" url:"url,omitempty,key"` // location of the buildpack
		} `json:"buildpacks,omitempty" url:"buildpacks,omitempty,key"` // overrides the buildpacks specified in the app.json manifest file
		Env *map[string]string `json:"env,omitempty" url:"env,omitempty,key"` // overrides of the env specified in the app.json manifest file
	} `json:"overrides,omitempty" url:"overrides,omitempty,key"` // overrides of keys in the app.json manifest file
	SourceBlob struct {
		Checksum *string `json:"checksum,omitempty" url:"checksum,omitempty,key"` // an optional checksum of the gzipped tarball for verifying its
		// integrity
		URL *string `json:"url,omitempty" url:"url,omitempty,key"` // URL of gzipped tarball of source code containing app.json manifest
		// file
		Version *string `json:"version,omitempty" url:"version,omitempty,key"` // Version of the gzipped tarball.
	} `json:"source_blob" url:"source_blob,key"` // gzipped tarball of source code containing app.json manifest file
}

// Create a new app setup from a gzipped tar archive containing an
// app.json manifest file.
func (s *Service) AppSetupCreate(ctx context.Context, o AppSetupCreateOpts) (*AppSetup, error) {
	var appSetup AppSetup
	return &appSetup, s.Post(ctx, &appSetup, fmt.Sprintf("/app-setups"), o)
}

// Get the status of an app setup.
func (s *Service) AppSetupInfo(ctx context.Context, appSetupIdentity string) (*AppSetup, error) {
	var appSetup AppSetup
	return &appSetup, s.Get(ctx, &appSetup, fmt.Sprintf("/app-setups/%v", appSetupIdentity), nil, nil)
}

// An app transfer represents a two party interaction for transferring
// ownership of an app.
type AppTransfer struct {
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // app involved in the transfer
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when app transfer was created
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of app transfer
	Owner     struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"owner" url:"owner,key"` // identity of the owner of the transfer
	Recipient struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"recipient" url:"recipient,key"` // identity of the recipient of the transfer
	State     string    `json:"state" url:"state,key"`           // the current state of an app transfer
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when app transfer was updated
}
type AppTransferCreateOpts struct {
	App       string `json:"app" url:"app,key"`                           // unique identifier of app
	Recipient string `json:"recipient" url:"recipient,key"`               // unique email address of account
	Silent    *bool  `json:"silent,omitempty" url:"silent,omitempty,key"` // whether to suppress email notification when transferring apps
}

// Create a new app transfer.
func (s *Service) AppTransferCreate(ctx context.Context, o AppTransferCreateOpts) (*AppTransfer, error) {
	var appTransfer AppTransfer
	return &appTransfer, s.Post(ctx, &appTransfer, fmt.Sprintf("/account/app-transfers"), o)
}

// Delete an existing app transfer
func (s *Service) AppTransferDelete(ctx context.Context, appTransferIdentity string) (*AppTransfer, error) {
	var appTransfer AppTransfer
	return &appTransfer, s.Delete(ctx, &appTransfer, fmt.Sprintf("/account/app-transfers/%v", appTransferIdentity))
}

// Info for existing app transfer.
func (s *Service) AppTransferInfo(ctx context.Context, appTransferIdentity string) (*AppTransfer, error) {
	var appTransfer AppTransfer
	return &appTransfer, s.Get(ctx, &appTransfer, fmt.Sprintf("/account/app-transfers/%v", appTransferIdentity), nil, nil)
}

type AppTransferListResult []AppTransfer

// List existing apps transfers.
func (s *Service) AppTransferList(ctx context.Context, lr *ListRange) (AppTransferListResult, error) {
	var appTransfer AppTransferListResult
	return appTransfer, s.Get(ctx, &appTransfer, fmt.Sprintf("/account/app-transfers"), nil, lr)
}

type AppTransferUpdateOpts struct {
	State string `json:"state" url:"state,key"` // the current state of an app transfer
}

// Update an existing app transfer.
func (s *Service) AppTransferUpdate(ctx context.Context, appTransferIdentity string, o AppTransferUpdateOpts) (*AppTransfer, error) {
	var appTransfer AppTransfer
	return &appTransfer, s.Patch(ctx, &appTransfer, fmt.Sprintf("/account/app-transfers/%v", appTransferIdentity), o)
}

// A build represents the process of transforming a code tarball into a
// slug
type Build struct {
	App struct {
		ID string `json:"id" url:"id,key"` // unique identifier of app
	} `json:"app" url:"app,key"` // app that the build belongs to
	Buildpacks *[]struct {
		URL string `json:"url" url:"url,key"` // location of the buildpack for the app. Either a url (unofficial
		// buildpacks) or an internal urn (heroku official buildpacks).
	} `json:"buildpacks" url:"buildpacks,key"` // buildpacks executed for this build, in order
	CreatedAt       time.Time `json:"created_at" url:"created_at,key"`               // when build was created
	ID              string    `json:"id" url:"id,key"`                               // unique identifier of build
	OutputStreamURL string    `json:"output_stream_url" url:"output_stream_url,key"` // Build process output will be available from this URL as a stream. The
	// stream is available as either `text/plain` or `text/event-stream`.
	// Clients should be prepared to handle disconnects and can resume the
	// stream by sending a `Range` header (for `text/plain`) or a
	// `Last-Event-Id` header (for `text/event-stream`).
	Release *struct {
		ID string `json:"id" url:"id,key"` // unique identifier of release
	} `json:"release" url:"release,key"` // release resulting from the build
	Slug *struct {
		ID string `json:"id" url:"id,key"` // unique identifier of slug
	} `json:"slug" url:"slug,key"` // slug created by this build
	SourceBlob struct {
		Checksum *string `json:"checksum" url:"checksum,key"` // an optional checksum of the gzipped tarball for verifying its
		// integrity
		URL string `json:"url" url:"url,key"` // URL where gzipped tar archive of source code for build was
		// downloaded.
		Version *string `json:"version" url:"version,key"` // Version of the gzipped tarball.
	} `json:"source_blob" url:"source_blob,key"` // location of gzipped tarball of source code used to create build
	Status    string    `json:"status" url:"status,key"`         // status of build
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when build was updated
	User      struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"user" url:"user,key"` // user that started the build
}
type BuildCreateOpts struct {
	Buildpacks *[]*struct {
		URL *string `json:"url,omitempty" url:"url,omitempty,key"` // location of the buildpack for the app. Either a url (unofficial
		// buildpacks) or an internal urn (heroku official buildpacks).
	} `json:"buildpacks,omitempty" url:"buildpacks,omitempty,key"` // buildpacks executed for this build, in order
	SourceBlob struct {
		Checksum *string `json:"checksum,omitempty" url:"checksum,omitempty,key"` // an optional checksum of the gzipped tarball for verifying its
		// integrity
		URL *string `json:"url,omitempty" url:"url,omitempty,key"` // URL where gzipped tar archive of source code for build was
		// downloaded.
		Version *string `json:"version,omitempty" url:"version,omitempty,key"` // Version of the gzipped tarball.
	} `json:"source_blob" url:"source_blob,key"` // location of gzipped tarball of source code used to create build
}

// Create a new build.
func (s *Service) BuildCreate(ctx context.Context, appIdentity string, o BuildCreateOpts) (*Build, error) {
	var build Build
	return &build, s.Post(ctx, &build, fmt.Sprintf("/apps/%v/builds", appIdentity), o)
}

// Info for existing build.
func (s *Service) BuildInfo(ctx context.Context, appIdentity string, buildIdentity string) (*Build, error) {
	var build Build
	return &build, s.Get(ctx, &build, fmt.Sprintf("/apps/%v/builds/%v", appIdentity, buildIdentity), nil, nil)
}

type BuildListResult []Build

// List existing build.
func (s *Service) BuildList(ctx context.Context, appIdentity string, lr *ListRange) (BuildListResult, error) {
	var build BuildListResult
	return build, s.Get(ctx, &build, fmt.Sprintf("/apps/%v/builds", appIdentity), nil, lr)
}

// A build result contains the output from a build.
type BuildResult struct {
	Build struct {
		ID              string `json:"id" url:"id,key"`                               // unique identifier of build
		OutputStreamURL string `json:"output_stream_url" url:"output_stream_url,key"` // Build process output will be available from this URL as a stream. The
		// stream is available as either `text/plain` or `text/event-stream`.
		// Clients should be prepared to handle disconnects and can resume the
		// stream by sending a `Range` header (for `text/plain`) or a
		// `Last-Event-Id` header (for `text/event-stream`).
		Status string `json:"status" url:"status,key"` // status of build
	} `json:"build" url:"build,key"` // identity of build
	ExitCode float64 `json:"exit_code" url:"exit_code,key"` // status from the build
	Lines    []struct {
		Line   string `json:"line" url:"line,key"`     // A line of output from the build.
		Stream string `json:"stream" url:"stream,key"` // The output stream where the line was sent.
	} `json:"lines" url:"lines,key"` // A list of all the lines of a build's output. This has been replaced
	// by the `output_stream_url` attribute on the build resource.
}

// Info for existing result.
func (s *Service) BuildResultInfo(ctx context.Context, appIdentity string, buildIdentity string) (*BuildResult, error) {
	var buildResult BuildResult
	return &buildResult, s.Get(ctx, &buildResult, fmt.Sprintf("/apps/%v/builds/%v/result", appIdentity, buildIdentity), nil, nil)
}

// A buildpack installation represents a buildpack that will be run
// against an app.
type BuildpackInstallation struct {
	Buildpack struct {
		Name string `json:"name" url:"name,key"` // either the shorthand name (heroku official buildpacks) or url
		// (unofficial buildpacks) of the buildpack for the app
		URL string `json:"url" url:"url,key"` // location of the buildpack for the app. Either a url (unofficial
		// buildpacks) or an internal urn (heroku official buildpacks).
	} `json:"buildpack" url:"buildpack,key"` // buildpack
	Ordinal int `json:"ordinal" url:"ordinal,key"` // determines the order in which the buildpacks will execute
}
type BuildpackInstallationUpdateOpts struct {
	Updates []struct {
		Buildpack string `json:"buildpack" url:"buildpack,key"` // location of the buildpack for the app. Either a url (unofficial
		// buildpacks) or an internal urn (heroku official buildpacks).
	} `json:"updates" url:"updates,key"` // The buildpack attribute can accept a name, a url, or a urn.
}
type BuildpackInstallationUpdateResult []BuildpackInstallation

// Update an app's buildpack installations.
func (s *Service) BuildpackInstallationUpdate(ctx context.Context, appIdentity string, o BuildpackInstallationUpdateOpts) (BuildpackInstallationUpdateResult, error) {
	var buildpackInstallation BuildpackInstallationUpdateResult
	return buildpackInstallation, s.Put(ctx, &buildpackInstallation, fmt.Sprintf("/apps/%v/buildpack-installations", appIdentity), o)
}

type BuildpackInstallationListResult []BuildpackInstallation

// List an app's existing buildpack installations.
func (s *Service) BuildpackInstallationList(ctx context.Context, appIdentity string, lr *ListRange) (BuildpackInstallationListResult, error) {
	var buildpackInstallation BuildpackInstallationListResult
	return buildpackInstallation, s.Get(ctx, &buildpackInstallation, fmt.Sprintf("/apps/%v/buildpack-installations", appIdentity), nil, lr)
}

// A collaborator represents an account that has been given access to an
// app on Heroku.
type Collaborator struct {
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // app collaborator belongs to
	CreatedAt   time.Time `json:"created_at" url:"created_at,key"` // when collaborator was created
	ID          string    `json:"id" url:"id,key"`                 // unique identifier of collaborator
	Permissions []struct {
		Description string `json:"description" url:"description,key"` // A description of what the app permission allows.
		Name        string `json:"name" url:"name,key"`               // The name of the app permission.
	} `json:"permissions" url:"permissions,key"`
	Role      *string   `json:"role" url:"role,key"`             // role in the team
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when collaborator was updated
	User      struct {
		Email     string `json:"email" url:"email,key"`         // unique email address of account
		Federated bool   `json:"federated" url:"federated,key"` // whether the user is federated and belongs to an Identity Provider
		ID        string `json:"id" url:"id,key"`               // unique identifier of an account
	} `json:"user" url:"user,key"` // identity of collaborated account
}
type CollaboratorCreateOpts struct {
	Silent *bool  `json:"silent,omitempty" url:"silent,omitempty,key"` // whether to suppress email invitation when creating collaborator
	User   string `json:"user" url:"user,key"`                         // unique email address of account
}

// Create a new collaborator.
func (s *Service) CollaboratorCreate(ctx context.Context, appIdentity string, o CollaboratorCreateOpts) (*Collaborator, error) {
	var collaborator Collaborator
	return &collaborator, s.Post(ctx, &collaborator, fmt.Sprintf("/apps/%v/collaborators", appIdentity), o)
}

// Delete an existing collaborator.
func (s *Service) CollaboratorDelete(ctx context.Context, appIdentity string, collaboratorIdentity string) (*Collaborator, error) {
	var collaborator Collaborator
	return &collaborator, s.Delete(ctx, &collaborator, fmt.Sprintf("/apps/%v/collaborators/%v", appIdentity, collaboratorIdentity))
}

// Info for existing collaborator.
func (s *Service) CollaboratorInfo(ctx context.Context, appIdentity string, collaboratorIdentity string) (*Collaborator, error) {
	var collaborator Collaborator
	return &collaborator, s.Get(ctx, &collaborator, fmt.Sprintf("/apps/%v/collaborators/%v", appIdentity, collaboratorIdentity), nil, nil)
}

type CollaboratorListResult []Collaborator

// List existing collaborators.
func (s *Service) CollaboratorList(ctx context.Context, appIdentity string, lr *ListRange) (CollaboratorListResult, error) {
	var collaborator CollaboratorListResult
	return collaborator, s.Get(ctx, &collaborator, fmt.Sprintf("/apps/%v/collaborators", appIdentity), nil, lr)
}

// Config Vars allow you to manage the configuration information
// provided to an app on Heroku.
type ConfigVar map[string]string
type ConfigVarInfoForAppResult map[string]*string

// Get config-vars for app.
func (s *Service) ConfigVarInfoForApp(ctx context.Context, appIdentity string) (ConfigVarInfoForAppResult, error) {
	var configVar ConfigVarInfoForAppResult
	return configVar, s.Get(ctx, &configVar, fmt.Sprintf("/apps/%v/config-vars", appIdentity), nil, nil)
}

type ConfigVarInfoForAppReleaseResult map[string]*string

// Get config-vars for a release.
func (s *Service) ConfigVarInfoForAppRelease(ctx context.Context, appIdentity string, releaseIdentity string) (ConfigVarInfoForAppReleaseResult, error) {
	var configVar ConfigVarInfoForAppReleaseResult
	return configVar, s.Get(ctx, &configVar, fmt.Sprintf("/apps/%v/releases/%v/config-vars", appIdentity, releaseIdentity), nil, nil)
}

type ConfigVarUpdateResult map[string]*string

// Update config-vars for app. You can update existing config-vars by
// setting them again, and remove by setting it to `null`.
func (s *Service) ConfigVarUpdate(ctx context.Context, appIdentity string, o map[string]*string) (ConfigVarUpdateResult, error) {
	var configVar ConfigVarUpdateResult
	return configVar, s.Patch(ctx, &configVar, fmt.Sprintf("/apps/%v/config-vars", appIdentity), o)
}

// A credit represents value that will be used up before further charges
// are assigned to an account.
type Credit struct {
	Amount    float64   `json:"amount" url:"amount,key"`         // total value of credit in cents
	Balance   float64   `json:"balance" url:"balance,key"`       // remaining value of credit in cents
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when credit was created
	ExpiresAt time.Time `json:"expires_at" url:"expires_at,key"` // when credit will expire
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of credit
	Title     string    `json:"title" url:"title,key"`           // a name for credit
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when credit was updated
}
type CreditCreateOpts struct {
	Code1 *string `json:"code1,omitempty" url:"code1,omitempty,key"` // first code from a discount card
	Code2 *string `json:"code2,omitempty" url:"code2,omitempty,key"` // second code from a discount card
}

// Create a new credit.
func (s *Service) CreditCreate(ctx context.Context, o CreditCreateOpts) (*Credit, error) {
	var credit Credit
	return &credit, s.Post(ctx, &credit, fmt.Sprintf("/account/credits"), o)
}

// Info for existing credit.
func (s *Service) CreditInfo(ctx context.Context, creditIdentity string) (*Credit, error) {
	var credit Credit
	return &credit, s.Get(ctx, &credit, fmt.Sprintf("/account/credits/%v", creditIdentity), nil, nil)
}

type CreditListResult []Credit

// List existing credits.
func (s *Service) CreditList(ctx context.Context, lr *ListRange) (CreditListResult, error) {
	var credit CreditListResult
	return credit, s.Get(ctx, &credit, fmt.Sprintf("/account/credits"), nil, lr)
}

// Domains define what web routes should be routed to an app on Heroku.
type Domain struct {
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // app that owns the domain
	CName     *string   `json:"cname" url:"cname,key"`           // canonical name record, the address to point a domain at
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when domain was created
	Hostname  string    `json:"hostname" url:"hostname,key"`     // full hostname
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of this domain
	Kind      string    `json:"kind" url:"kind,key"`             // type of domain name
	Status    string    `json:"status" url:"status,key"`         // status of this record's cname
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when domain was updated
}
type DomainCreateOpts struct {
	Hostname string `json:"hostname" url:"hostname,key"` // full hostname
}

// Create a new domain.
func (s *Service) DomainCreate(ctx context.Context, appIdentity string, o DomainCreateOpts) (*Domain, error) {
	var domain Domain
	return &domain, s.Post(ctx, &domain, fmt.Sprintf("/apps/%v/domains", appIdentity), o)
}

// Delete an existing domain
func (s *Service) DomainDelete(ctx context.Context, appIdentity string, domainIdentity string) (*Domain, error) {
	var domain Domain
	return &domain, s.Delete(ctx, &domain, fmt.Sprintf("/apps/%v/domains/%v", appIdentity, domainIdentity))
}

// Info for existing domain.
func (s *Service) DomainInfo(ctx context.Context, appIdentity string, domainIdentity string) (*Domain, error) {
	var domain Domain
	return &domain, s.Get(ctx, &domain, fmt.Sprintf("/apps/%v/domains/%v", appIdentity, domainIdentity), nil, nil)
}

type DomainListResult []Domain

// List existing domains.
func (s *Service) DomainList(ctx context.Context, appIdentity string, lr *ListRange) (DomainListResult, error) {
	var domain DomainListResult
	return domain, s.Get(ctx, &domain, fmt.Sprintf("/apps/%v/domains", appIdentity), nil, lr)
}

// Dynos encapsulate running processes of an app on Heroku. Detailed
// information about dyno sizes can be found at:
// [https://devcenter.heroku.com/articles/dyno-types](https://devcenter.h
// eroku.com/articles/dyno-types).
type Dyno struct {
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // app formation belongs to
	AttachURL *string `json:"attach_url" url:"attach_url,key"` // a URL to stream output from for attached processes or null for
	// non-attached processes
	Command   string    `json:"command" url:"command,key"`       // command used to start this process
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when dyno was created
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of this dyno
	Name      string    `json:"name" url:"name,key"`             // the name of this process on this dyno
	Release   struct {
		ID      string `json:"id" url:"id,key"`           // unique identifier of release
		Version int    `json:"version" url:"version,key"` // unique version assigned to the release
	} `json:"release" url:"release,key"` // app release of the dyno
	Size  string `json:"size" url:"size,key"`   // dyno size (default: "standard-1X")
	State string `json:"state" url:"state,key"` // current status of process (either: crashed, down, idle, starting, or
	// up)
	Type      string    `json:"type" url:"type,key"`             // type of process
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when process last changed state
}
type DynoCreateOpts struct {
	Attach     *bool              `json:"attach,omitempty" url:"attach,omitempty,key"`             // whether to stream output or not
	Command    string             `json:"command" url:"command,key"`                               // command used to start this process
	Env        *map[string]string `json:"env,omitempty" url:"env,omitempty,key"`                   // custom environment to add to the dyno config vars
	ForceNoTty *bool              `json:"force_no_tty,omitempty" url:"force_no_tty,omitempty,key"` // force an attached one-off dyno to not run in a tty
	Size       *string            `json:"size,omitempty" url:"size,omitempty,key"`                 // dyno size (default: "standard-1X")
	TimeToLive *int               `json:"time_to_live,omitempty" url:"time_to_live,omitempty,key"` // seconds until dyno expires, after which it will soon be killed
	Type       *string            `json:"type,omitempty" url:"type,omitempty,key"`                 // type of process
}

// Create a new dyno.
func (s *Service) DynoCreate(ctx context.Context, appIdentity string, o DynoCreateOpts) (*Dyno, error) {
	var dyno Dyno
	return &dyno, s.Post(ctx, &dyno, fmt.Sprintf("/apps/%v/dynos", appIdentity), o)
}

type DynoRestartResult struct{}

// Restart dyno.
func (s *Service) DynoRestart(ctx context.Context, appIdentity string, dynoIdentity string) (DynoRestartResult, error) {
	var dyno DynoRestartResult
	return dyno, s.Delete(ctx, &dyno, fmt.Sprintf("/apps/%v/dynos/%v", appIdentity, dynoIdentity))
}

type DynoRestartAllResult struct{}

// Restart all dynos.
func (s *Service) DynoRestartAll(ctx context.Context, appIdentity string) (DynoRestartAllResult, error) {
	var dyno DynoRestartAllResult
	return dyno, s.Delete(ctx, &dyno, fmt.Sprintf("/apps/%v/dynos", appIdentity))
}

type DynoStopResult struct{}

// Stop dyno.
func (s *Service) DynoStop(ctx context.Context, appIdentity string, dynoIdentity string) (DynoStopResult, error) {
	var dyno DynoStopResult
	return dyno, s.Post(ctx, &dyno, fmt.Sprintf("/apps/%v/dynos/%v/actions/stop", appIdentity, dynoIdentity), nil)
}

// Info for existing dyno.
func (s *Service) DynoInfo(ctx context.Context, appIdentity string, dynoIdentity string) (*Dyno, error) {
	var dyno Dyno
	return &dyno, s.Get(ctx, &dyno, fmt.Sprintf("/apps/%v/dynos/%v", appIdentity, dynoIdentity), nil, nil)
}

type DynoListResult []Dyno

// List existing dynos.
func (s *Service) DynoList(ctx context.Context, appIdentity string, lr *ListRange) (DynoListResult, error) {
	var dyno DynoListResult
	return dyno, s.Get(ctx, &dyno, fmt.Sprintf("/apps/%v/dynos", appIdentity), nil, lr)
}

// Dyno sizes are the values and details of sizes that can be assigned
// to dynos. This information can also be found at :
// [https://devcenter.heroku.com/articles/dyno-types](https://devcenter.h
// eroku.com/articles/dyno-types).
type DynoSize struct {
	Compute          int       `json:"compute" url:"compute,key"`                       // minimum vCPUs, non-dedicated may get more depending on load
	Cost             *struct{} `json:"cost" url:"cost,key"`                             // price information for this dyno size
	Dedicated        bool      `json:"dedicated" url:"dedicated,key"`                   // whether this dyno will be dedicated to one user
	DynoUnits        int       `json:"dyno_units" url:"dyno_units,key"`                 // unit of consumption for Heroku Enterprise customers
	ID               string    `json:"id" url:"id,key"`                                 // unique identifier of this dyno size
	Memory           float64   `json:"memory" url:"memory,key"`                         // amount of RAM in GB
	Name             string    `json:"name" url:"name,key"`                             // the name of this dyno-size
	PrivateSpaceOnly bool      `json:"private_space_only" url:"private_space_only,key"` // whether this dyno can only be provisioned in a private space
}

// Info for existing dyno size.
func (s *Service) DynoSizeInfo(ctx context.Context, dynoSizeIdentity string) (*DynoSize, error) {
	var dynoSize DynoSize
	return &dynoSize, s.Get(ctx, &dynoSize, fmt.Sprintf("/dyno-sizes/%v", dynoSizeIdentity), nil, nil)
}

type DynoSizeListResult []DynoSize

// List existing dyno sizes.
func (s *Service) DynoSizeList(ctx context.Context, lr *ListRange) (DynoSizeListResult, error) {
	var dynoSize DynoSizeListResult
	return dynoSize, s.Get(ctx, &dynoSize, fmt.Sprintf("/dyno-sizes"), nil, lr)
}

// An event represents an action performed on another API resource.
type Event struct {
	Action string `json:"action" url:"action,key"` // the operation performed on the resource
	Actor  struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"actor" url:"actor,key"` // user that performed the operation
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when the event was created
	Data      struct {
		AllowTracking       bool      `json:"allow_tracking" url:"allow_tracking,key"` // whether to allow third party web activity tracking
		Beta                bool      `json:"beta" url:"beta,key"`                     // whether allowed to utilize beta Heroku features
		CreatedAt           time.Time `json:"created_at" url:"created_at,key"`         // when account was created
		DefaultOrganization *struct {
			ID   string `json:"id" url:"id,key"`     // unique identifier of organization
			Name string `json:"name" url:"name,key"` // unique name of organization
		} `json:"default_organization" url:"default_organization,key"` // organization selected by default
		DelinquentAt     *time.Time `json:"delinquent_at" url:"delinquent_at,key"` // when account became delinquent
		Email            string     `json:"email" url:"email,key"`                 // unique email address of account
		Federated        bool       `json:"federated" url:"federated,key"`         // whether the user is federated and belongs to an Identity Provider
		ID               string     `json:"id" url:"id,key"`                       // unique identifier of an account
		IdentityProvider *struct {
			ID           string `json:"id" url:"id,key"` // unique identifier of this identity provider
			Organization struct {
				Name string `json:"name" url:"name,key"` // unique name of organization
			} `json:"organization" url:"organization,key"`
		} `json:"identity_provider" url:"identity_provider,key"` // Identity Provider details for federated users.
		LastLogin               *time.Time `json:"last_login" url:"last_login,key"`                               // when account last authorized with Heroku
		Name                    *string    `json:"name" url:"name,key"`                                           // full name of the account owner
		SmsNumber               *string    `json:"sms_number" url:"sms_number,key"`                               // SMS number of account
		SuspendedAt             *time.Time `json:"suspended_at" url:"suspended_at,key"`                           // when account was suspended
		TwoFactorAuthentication bool       `json:"two_factor_authentication" url:"two_factor_authentication,key"` // whether two-factor auth is enabled on the account
		UpdatedAt               time.Time  `json:"updated_at" url:"updated_at,key"`                               // when account was updated
		Verified                bool       `json:"verified" url:"verified,key"`                                   // whether account has been verified with billing information
	} `json:"data" url:"data,key"` // An account represents an individual signed up to use the Heroku
	// platform.
	ID           string     `json:"id" url:"id,key"`                       // unique identifier of an event
	PreviousData struct{}   `json:"previous_data" url:"previous_data,key"` // data fields that were changed during update with previous values
	PublishedAt  *time.Time `json:"published_at" url:"published_at,key"`   // when the event was published
	Resource     string     `json:"resource" url:"resource,key"`           // the type of resource affected
	Sequence     *string    `json:"sequence" url:"sequence,key"`           // a numeric string representing the event's sequence
	UpdatedAt    time.Time  `json:"updated_at" url:"updated_at,key"`       // when the event was updated (same as created)
	Version      string     `json:"version" url:"version,key"`             // the event's API version string
}

// A failed event represents a failure of an action performed on another
// API resource.
type FailedEvent struct {
	Action   string  `json:"action" url:"action,key"`     // The attempted operation performed on the resource.
	Code     *int    `json:"code" url:"code,key"`         // An HTTP status code.
	ErrorID  *string `json:"error_id" url:"error_id,key"` // ID of error raised.
	Message  string  `json:"message" url:"message,key"`   // A detailed error message.
	Method   string  `json:"method" url:"method,key"`     // The HTTP method type of the failed action.
	Path     string  `json:"path" url:"path,key"`         // The path of the attempted operation.
	Resource *struct {
		ID   string `json:"id" url:"id,key"`     // Unique identifier of a resource.
		Name string `json:"name" url:"name,key"` // the type of resource affected
	} `json:"resource" url:"resource,key"` // The related resource of the failed action.
}

// Filters are special endpoints to allow for API consumers to specify a
// subset of resources to consume in order to reduce the number of
// requests that are performed.  Each filter endpoint endpoint is
// responsible for determining its supported request format.  The
// endpoints are over POST in order to handle large request bodies
// without hitting request uri query length limitations, but the
// requests themselves are idempotent and will not have side effects.
type FilterApps struct{}
type FilterAppsAppsOpts struct {
	In *struct {
		ID *[]*string `json:"id,omitempty" url:"id,omitempty,key"`
	} `json:"in,omitempty" url:"in,omitempty,key"`
}
type FilterAppsAppsResult []struct {
	ArchivedAt                   *time.Time `json:"archived_at" url:"archived_at,key"`                                       // when app was archived
	BuildpackProvidedDescription *string    `json:"buildpack_provided_description" url:"buildpack_provided_description,key"` // description from buildpack of app
	CreatedAt                    time.Time  `json:"created_at" url:"created_at,key"`                                         // when app was created
	GitURL                       string     `json:"git_url" url:"git_url,key"`                                               // git repo URL of app
	ID                           string     `json:"id" url:"id,key"`                                                         // unique identifier of app
	Joined                       bool       `json:"joined" url:"joined,key"`                                                 // is the current member a collaborator on this app.
	Locked                       bool       `json:"locked" url:"locked,key"`                                                 // are other team members forbidden from joining this app.
	Maintenance                  bool       `json:"maintenance" url:"maintenance,key"`                                       // maintenance status of app
	Name                         string     `json:"name" url:"name,key"`                                                     // unique name of app
	Owner                        *struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"owner" url:"owner,key"` // identity of app owner
	Region struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of region
		Name string `json:"name" url:"name,key"` // unique name of region
	} `json:"region" url:"region,key"` // identity of app region
	ReleasedAt *time.Time `json:"released_at" url:"released_at,key"` // when app was released
	RepoSize   *int       `json:"repo_size" url:"repo_size,key"`     // git repo size in bytes of app
	SlugSize   *int       `json:"slug_size" url:"slug_size,key"`     // slug size in bytes of app
	Space      *struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of space
		Name string `json:"name" url:"name,key"` // unique name of space
	} `json:"space" url:"space,key"` // identity of space
	Stack struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of stack
		Name string `json:"name" url:"name,key"` // unique name of stack
	} `json:"stack" url:"stack,key"` // identity of app stack
	Team *struct {
		Name string `json:"name" url:"name,key"` // unique name of team
	} `json:"team" url:"team,key"` // team that owns this app
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when app was updated
	WebURL    string    `json:"web_url" url:"web_url,key"`       // web URL of app
}

// Request an apps list filtered by app id.
func (s *Service) FilterAppsApps(ctx context.Context, o FilterAppsAppsOpts) (FilterAppsAppsResult, error) {
	var filterApps FilterAppsAppsResult
	return filterApps, s.Post(ctx, &filterApps, fmt.Sprintf("/filters/apps"), o)
}

// The formation of processes that should be maintained for an app.
// Update the formation to scale processes or change dyno sizes.
// Available process type names and commands are defined by the
// `process_types` attribute for the [slug](#slug) currently released on
// an app.
type Formation struct {
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // app formation belongs to
	Command   string    `json:"command" url:"command,key"`       // command to use to launch this process
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when process type was created
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of this process type
	Quantity  int       `json:"quantity" url:"quantity,key"`     // number of processes to maintain
	Size      string    `json:"size" url:"size,key"`             // dyno size (default: "standard-1X")
	Type      string    `json:"type" url:"type,key"`             // type of process to maintain
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when dyno type was updated
}

// Info for a process type
func (s *Service) FormationInfo(ctx context.Context, appIdentity string, formationIdentity string) (*Formation, error) {
	var formation Formation
	return &formation, s.Get(ctx, &formation, fmt.Sprintf("/apps/%v/formation/%v", appIdentity, formationIdentity), nil, nil)
}

type FormationListResult []Formation

// List process type formation
func (s *Service) FormationList(ctx context.Context, appIdentity string, lr *ListRange) (FormationListResult, error) {
	var formation FormationListResult
	return formation, s.Get(ctx, &formation, fmt.Sprintf("/apps/%v/formation", appIdentity), nil, lr)
}

type FormationBatchUpdateOpts struct {
	Updates []struct {
		Quantity *int    `json:"quantity,omitempty" url:"quantity,omitempty,key"` // number of processes to maintain
		Size     *string `json:"size,omitempty" url:"size,omitempty,key"`         // dyno size (default: "standard-1X")
		Type     string  `json:"type" url:"type,key"`                             // type of process to maintain
	} `json:"updates" url:"updates,key"` // Array with formation updates. Each element must have "type", the id
	// or name of the process type to be updated, and can optionally update
	// its "quantity" or "size".
}
type FormationBatchUpdateResult []Formation

// Batch update process types
func (s *Service) FormationBatchUpdate(ctx context.Context, appIdentity string, o FormationBatchUpdateOpts) (FormationBatchUpdateResult, error) {
	var formation FormationBatchUpdateResult
	return formation, s.Patch(ctx, &formation, fmt.Sprintf("/apps/%v/formation", appIdentity), o)
}

type FormationUpdateOpts struct {
	Quantity *int    `json:"quantity,omitempty" url:"quantity,omitempty,key"` // number of processes to maintain
	Size     *string `json:"size,omitempty" url:"size,omitempty,key"`         // dyno size (default: "standard-1X")
}

// Update process type
func (s *Service) FormationUpdate(ctx context.Context, appIdentity string, formationIdentity string, o FormationUpdateOpts) (*Formation, error) {
	var formation Formation
	return &formation, s.Patch(ctx, &formation, fmt.Sprintf("/apps/%v/formation/%v", appIdentity, formationIdentity), o)
}

// Identity Providers represent the SAML configuration of an
// Organization.
type IdentityProvider struct {
	Certificate  string    `json:"certificate" url:"certificate,key"` // raw contents of the public certificate (eg: .crt or .pem file)
	CreatedAt    time.Time `json:"created_at" url:"created_at,key"`   // when provider record was created
	EntityID     string    `json:"entity_id" url:"entity_id,key"`     // URL identifier provided by the identity provider
	ID           string    `json:"id" url:"id,key"`                   // unique identifier of this identity provider
	Organization *struct {
		Name string `json:"name" url:"name,key"` // unique name of organization
	} `json:"organization" url:"organization,key"` // organization associated with this identity provider
	SloTargetURL string    `json:"slo_target_url" url:"slo_target_url,key"` // single log out URL for this identity provider
	SsoTargetURL string    `json:"sso_target_url" url:"sso_target_url,key"` // single sign on URL for this identity provider
	UpdatedAt    time.Time `json:"updated_at" url:"updated_at,key"`         // when the identity provider record was updated
}
type IdentityProviderListByOrganizationResult []IdentityProvider

// Get a list of an organization's Identity Providers
func (s *Service) IdentityProviderListByOrganization(ctx context.Context, organizationName string, lr *ListRange) (IdentityProviderListByOrganizationResult, error) {
	var identityProvider IdentityProviderListByOrganizationResult
	return identityProvider, s.Get(ctx, &identityProvider, fmt.Sprintf("/organizations/%v/identity-providers", organizationName), nil, lr)
}

type IdentityProviderCreateByOrganizationOpts struct {
	Certificate  string  `json:"certificate" url:"certificate,key"`                           // raw contents of the public certificate (eg: .crt or .pem file)
	EntityID     string  `json:"entity_id" url:"entity_id,key"`                               // URL identifier provided by the identity provider
	SloTargetURL *string `json:"slo_target_url,omitempty" url:"slo_target_url,omitempty,key"` // single log out URL for this identity provider
	SsoTargetURL string  `json:"sso_target_url" url:"sso_target_url,key"`                     // single sign on URL for this identity provider
}

// Create an Identity Provider for an organization
func (s *Service) IdentityProviderCreateByOrganization(ctx context.Context, organizationName string, o IdentityProviderCreateByOrganizationOpts) (*IdentityProvider, error) {
	var identityProvider IdentityProvider
	return &identityProvider, s.Post(ctx, &identityProvider, fmt.Sprintf("/organizations/%v/identity-providers", organizationName), o)
}

type IdentityProviderUpdateByOrganizationOpts struct {
	Certificate  *string `json:"certificate,omitempty" url:"certificate,omitempty,key"`       // raw contents of the public certificate (eg: .crt or .pem file)
	EntityID     *string `json:"entity_id,omitempty" url:"entity_id,omitempty,key"`           // URL identifier provided by the identity provider
	SloTargetURL *string `json:"slo_target_url,omitempty" url:"slo_target_url,omitempty,key"` // single log out URL for this identity provider
	SsoTargetURL *string `json:"sso_target_url,omitempty" url:"sso_target_url,omitempty,key"` // single sign on URL for this identity provider
}

// Update an organization's Identity Provider
func (s *Service) IdentityProviderUpdateByOrganization(ctx context.Context, organizationName string, identityProviderID string, o IdentityProviderUpdateByOrganizationOpts) (*IdentityProvider, error) {
	var identityProvider IdentityProvider
	return &identityProvider, s.Patch(ctx, &identityProvider, fmt.Sprintf("/organizations/%v/identity-providers/%v", organizationName, identityProviderID), o)
}

// Delete an organization's Identity Provider
func (s *Service) IdentityProviderDeleteByOrganization(ctx context.Context, organizationName string, identityProviderID string) (*IdentityProvider, error) {
	var identityProvider IdentityProvider
	return &identityProvider, s.Delete(ctx, &identityProvider, fmt.Sprintf("/organizations/%v/identity-providers/%v", organizationName, identityProviderID))
}

type IdentityProviderListByTeamResult []IdentityProvider

// Get a list of a team's Identity Providers
func (s *Service) IdentityProviderListByTeam(ctx context.Context, teamIdentity string, lr *ListRange) (IdentityProviderListByTeamResult, error) {
	var identityProvider IdentityProviderListByTeamResult
	return identityProvider, s.Get(ctx, &identityProvider, fmt.Sprintf("/teams/%v/identity-providers", teamIdentity), nil, lr)
}

type IdentityProviderCreateByTeamOpts struct {
	Certificate  string  `json:"certificate" url:"certificate,key"`                           // raw contents of the public certificate (eg: .crt or .pem file)
	EntityID     string  `json:"entity_id" url:"entity_id,key"`                               // URL identifier provided by the identity provider
	SloTargetURL *string `json:"slo_target_url,omitempty" url:"slo_target_url,omitempty,key"` // single log out URL for this identity provider
	SsoTargetURL string  `json:"sso_target_url" url:"sso_target_url,key"`                     // single sign on URL for this identity provider
}

// Create an Identity Provider for a team
func (s *Service) IdentityProviderCreateByTeam(ctx context.Context, teamIdentity string, o IdentityProviderCreateByTeamOpts) (*IdentityProvider, error) {
	var identityProvider IdentityProvider
	return &identityProvider, s.Post(ctx, &identityProvider, fmt.Sprintf("/teams/%v/identity-providers", teamIdentity), o)
}

type IdentityProviderUpdateByTeamOpts struct {
	Certificate  *string `json:"certificate,omitempty" url:"certificate,omitempty,key"`       // raw contents of the public certificate (eg: .crt or .pem file)
	EntityID     *string `json:"entity_id,omitempty" url:"entity_id,omitempty,key"`           // URL identifier provided by the identity provider
	SloTargetURL *string `json:"slo_target_url,omitempty" url:"slo_target_url,omitempty,key"` // single log out URL for this identity provider
	SsoTargetURL *string `json:"sso_target_url,omitempty" url:"sso_target_url,omitempty,key"` // single sign on URL for this identity provider
}

// Update a team's Identity Provider
func (s *Service) IdentityProviderUpdateByTeam(ctx context.Context, teamIdentity string, identityProviderID string, o IdentityProviderUpdateByTeamOpts) (*IdentityProvider, error) {
	var identityProvider IdentityProvider
	return &identityProvider, s.Patch(ctx, &identityProvider, fmt.Sprintf("/teams/%v/identity-providers/%v", teamIdentity, identityProviderID), o)
}

// Delete a team's Identity Provider
func (s *Service) IdentityProviderDeleteByTeam(ctx context.Context, teamName string, identityProviderID string) (*IdentityProvider, error) {
	var identityProvider IdentityProvider
	return &identityProvider, s.Delete(ctx, &identityProvider, fmt.Sprintf("/teams/%v/identity-providers/%v", teamName, identityProviderID))
}

// An inbound-ruleset is a collection of rules that specify what hosts
// can or cannot connect to an application.
type InboundRuleset struct {
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when inbound-ruleset was created
	CreatedBy string    `json:"created_by" url:"created_by,key"` // unique email address of account
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of an inbound-ruleset
	Rules     []struct {
		Action string `json:"action" url:"action,key"` // states whether the connection is allowed or denied
		Source string `json:"source" url:"source,key"` // is the request’s source in CIDR notation
	} `json:"rules" url:"rules,key"`
}

// Current inbound ruleset for a space
func (s *Service) InboundRulesetCurrent(ctx context.Context, spaceIdentity string) (*InboundRuleset, error) {
	var inboundRuleset InboundRuleset
	return &inboundRuleset, s.Get(ctx, &inboundRuleset, fmt.Sprintf("/spaces/%v/inbound-ruleset", spaceIdentity), nil, nil)
}

// Info on an existing Inbound Ruleset
func (s *Service) InboundRulesetInfo(ctx context.Context, spaceIdentity string, inboundRulesetIdentity string) (*InboundRuleset, error) {
	var inboundRuleset InboundRuleset
	return &inboundRuleset, s.Get(ctx, &inboundRuleset, fmt.Sprintf("/spaces/%v/inbound-rulesets/%v", spaceIdentity, inboundRulesetIdentity), nil, nil)
}

type InboundRulesetListResult []InboundRuleset

// List all inbound rulesets for a space
func (s *Service) InboundRulesetList(ctx context.Context, spaceIdentity string, lr *ListRange) (InboundRulesetListResult, error) {
	var inboundRuleset InboundRulesetListResult
	return inboundRuleset, s.Get(ctx, &inboundRuleset, fmt.Sprintf("/spaces/%v/inbound-rulesets", spaceIdentity), nil, lr)
}

type InboundRulesetCreateOpts struct {
	Rules *[]*struct {
		Action string `json:"action" url:"action,key"` // states whether the connection is allowed or denied
		Source string `json:"source" url:"source,key"` // is the request’s source in CIDR notation
	} `json:"rules,omitempty" url:"rules,omitempty,key"`
}

// Create a new inbound ruleset
func (s *Service) InboundRulesetCreate(ctx context.Context, spaceIdentity string, o InboundRulesetCreateOpts) (*InboundRuleset, error) {
	var inboundRuleset InboundRuleset
	return &inboundRuleset, s.Put(ctx, &inboundRuleset, fmt.Sprintf("/spaces/%v/inbound-ruleset", spaceIdentity), o)
}

// An invitation represents an invite sent to a user to use the Heroku
// platform.
type Invitation struct {
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when invitation was created
	User      struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"user" url:"user,key"`
	VerificationRequired bool `json:"verification_required" url:"verification_required,key"` // if the invitation requires verification
}

// Info for invitation.
func (s *Service) InvitationInfo(ctx context.Context, invitationIdentity string) (*Invitation, error) {
	var invitation Invitation
	return &invitation, s.Get(ctx, &invitation, fmt.Sprintf("/invitations/%v", invitationIdentity), nil, nil)
}

type InvitationCreateOpts struct {
	Email string  `json:"email" url:"email,key"` // unique email address of account
	Name  *string `json:"name" url:"name,key"`   // full name of the account owner
}

// Invite a user.
func (s *Service) InvitationCreate(ctx context.Context, o InvitationCreateOpts) (*Invitation, error) {
	var invitation Invitation
	return &invitation, s.Post(ctx, &invitation, fmt.Sprintf("/invitations"), o)
}

type InvitationSendVerificationCodeOpts struct {
	Method      *string `json:"method,omitempty" url:"method,omitempty,key"` // Transport used to send verification code
	PhoneNumber string  `json:"phone_number" url:"phone_number,key"`         // Phone number to send verification code
}

// Send a verification code for an invitation via SMS/phone call.
func (s *Service) InvitationSendVerificationCode(ctx context.Context, invitationIdentity string, o InvitationSendVerificationCodeOpts) (*Invitation, error) {
	var invitation Invitation
	return &invitation, s.Post(ctx, &invitation, fmt.Sprintf("/invitations/%v/actions/send-verification", invitationIdentity), o)
}

type InvitationVerifyOpts struct {
	VerificationCode string `json:"verification_code" url:"verification_code,key"` // Value used to verify invitation
}

// Verify an invitation using a verification code.
func (s *Service) InvitationVerify(ctx context.Context, invitationIdentity string, o InvitationVerifyOpts) (*Invitation, error) {
	var invitation Invitation
	return &invitation, s.Post(ctx, &invitation, fmt.Sprintf("/invitations/%v/actions/verify", invitationIdentity), o)
}

type InvitationFinalizeOpts struct {
	Password             string `json:"password" url:"password,key"`                                         // current password on the account
	PasswordConfirmation string `json:"password_confirmation" url:"password_confirmation,key"`               // current password on the account
	ReceiveNewsletter    *bool  `json:"receive_newsletter,omitempty" url:"receive_newsletter,omitempty,key"` // whether this user should receive a newsletter or not
}

// Finalize Invitation and Create Account.
func (s *Service) InvitationFinalize(ctx context.Context, invitationIdentity string, o InvitationFinalizeOpts) (*Invitation, error) {
	var invitation Invitation
	return &invitation, s.Patch(ctx, &invitation, fmt.Sprintf("/invitations/%v", invitationIdentity), o)
}

// An invoice is an itemized bill of goods for an account which includes
// pricing and charges.
type Invoice struct {
	ChargesTotal float64   `json:"charges_total" url:"charges_total,key"` // total charges on this invoice
	CreatedAt    time.Time `json:"created_at" url:"created_at,key"`       // when invoice was created
	CreditsTotal float64   `json:"credits_total" url:"credits_total,key"` // total credits on this invoice
	ID           string    `json:"id" url:"id,key"`                       // unique identifier of this invoice
	Number       int       `json:"number" url:"number,key"`               // human readable invoice number
	PeriodEnd    string    `json:"period_end" url:"period_end,key"`       // the ending date that the invoice covers
	PeriodStart  string    `json:"period_start" url:"period_start,key"`   // the starting date that this invoice covers
	State        int       `json:"state" url:"state,key"`                 // payment status for this invoice (pending, successful, failed)
	Total        float64   `json:"total" url:"total,key"`                 // combined total of charges and credits on this invoice
	UpdatedAt    time.Time `json:"updated_at" url:"updated_at,key"`       // when invoice was updated
}

// Info for existing invoice.
func (s *Service) InvoiceInfo(ctx context.Context, invoiceIdentity int) (*Invoice, error) {
	var invoice Invoice
	return &invoice, s.Get(ctx, &invoice, fmt.Sprintf("/account/invoices/%v", invoiceIdentity), nil, nil)
}

type InvoiceListResult []Invoice

// List existing invoices.
func (s *Service) InvoiceList(ctx context.Context, lr *ListRange) (InvoiceListResult, error) {
	var invoice InvoiceListResult
	return invoice, s.Get(ctx, &invoice, fmt.Sprintf("/account/invoices"), nil, lr)
}

// An invoice address represents the address that should be listed on an
// invoice.
type InvoiceAddress struct {
	Address1          string `json:"address_1" url:"address_1,key"`                     // invoice street address line 1
	Address2          string `json:"address_2" url:"address_2,key"`                     // invoice street address line 2
	City              string `json:"city" url:"city,key"`                               // invoice city
	Country           string `json:"country" url:"country,key"`                         // country
	HerokuID          string `json:"heroku_id" url:"heroku_id,key"`                     // heroku_id identifier reference
	Other             string `json:"other" url:"other,key"`                             // metadata / additional information to go on invoice
	PostalCode        string `json:"postal_code" url:"postal_code,key"`                 // invoice zip code
	State             string `json:"state" url:"state,key"`                             // invoice state
	UseInvoiceAddress bool   `json:"use_invoice_address" url:"use_invoice_address,key"` // flag to use the invoice address for an account or not
}

// Retrieve existing invoice address.
func (s *Service) InvoiceAddressInfo(ctx context.Context) (*InvoiceAddress, error) {
	var invoiceAddress InvoiceAddress
	return &invoiceAddress, s.Get(ctx, &invoiceAddress, fmt.Sprintf("/account/invoice-address"), nil, nil)
}

type InvoiceAddressUpdateOpts struct {
	Address1          *string `json:"address_1,omitempty" url:"address_1,omitempty,key"`                     // invoice street address line 1
	Address2          *string `json:"address_2,omitempty" url:"address_2,omitempty,key"`                     // invoice street address line 2
	City              *string `json:"city,omitempty" url:"city,omitempty,key"`                               // invoice city
	Country           *string `json:"country,omitempty" url:"country,omitempty,key"`                         // country
	Other             *string `json:"other,omitempty" url:"other,omitempty,key"`                             // metadata / additional information to go on invoice
	PostalCode        *string `json:"postal_code,omitempty" url:"postal_code,omitempty,key"`                 // invoice zip code
	State             *string `json:"state,omitempty" url:"state,omitempty,key"`                             // invoice state
	UseInvoiceAddress *bool   `json:"use_invoice_address,omitempty" url:"use_invoice_address,omitempty,key"` // flag to use the invoice address for an account or not
}

// Update invoice address for an account.
func (s *Service) InvoiceAddressUpdate(ctx context.Context, o InvoiceAddressUpdateOpts) (*InvoiceAddress, error) {
	var invoiceAddress InvoiceAddress
	return &invoiceAddress, s.Put(ctx, &invoiceAddress, fmt.Sprintf("/account/invoice-address"), o)
}

// Keys represent public SSH keys associated with an account and are
// used to authorize accounts as they are performing git operations.
type Key struct {
	Comment     string    `json:"comment" url:"comment,key"`         // comment on the key
	CreatedAt   time.Time `json:"created_at" url:"created_at,key"`   // when key was created
	Email       string    `json:"email" url:"email,key"`             // deprecated. Please refer to 'comment' instead
	Fingerprint string    `json:"fingerprint" url:"fingerprint,key"` // a unique identifying string based on contents
	ID          string    `json:"id" url:"id,key"`                   // unique identifier of this key
	PublicKey   string    `json:"public_key" url:"public_key,key"`   // full public_key as uploaded
	UpdatedAt   time.Time `json:"updated_at" url:"updated_at,key"`   // when key was updated
}

// Info for existing key.
func (s *Service) KeyInfo(ctx context.Context, keyIdentity string) (*Key, error) {
	var key Key
	return &key, s.Get(ctx, &key, fmt.Sprintf("/account/keys/%v", keyIdentity), nil, nil)
}

type KeyListResult []Key

// List existing keys.
func (s *Service) KeyList(ctx context.Context, lr *ListRange) (KeyListResult, error) {
	var key KeyListResult
	return key, s.Get(ctx, &key, fmt.Sprintf("/account/keys"), nil, lr)
}

// [Log drains](https://devcenter.heroku.com/articles/log-drains)
// provide a way to forward your Heroku logs to an external syslog
// server for long-term archiving. This external service must be
// configured to receive syslog packets from Heroku, whereupon its URL
// can be added to an app using this API. Some add-ons will add a log
// drain when they are provisioned to an app. These drains can only be
// removed by removing the add-on.
type LogDrain struct {
	Addon *struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of add-on
		Name string `json:"name" url:"name,key"` // globally unique name of the add-on
	} `json:"addon" url:"addon,key"` // add-on that created the drain
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when log drain was created
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of this log drain
	Token     string    `json:"token" url:"token,key"`           // token associated with the log drain
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when log drain was updated
	URL       string    `json:"url" url:"url,key"`               // url associated with the log drain
}
type LogDrainCreateOpts struct {
	URL string `json:"url" url:"url,key"` // url associated with the log drain
}

// Create a new log drain.
func (s *Service) LogDrainCreate(ctx context.Context, appIdentity string, o LogDrainCreateOpts) (*LogDrain, error) {
	var logDrain LogDrain
	return &logDrain, s.Post(ctx, &logDrain, fmt.Sprintf("/apps/%v/log-drains", appIdentity), o)
}

// Delete an existing log drain. Log drains added by add-ons can only be
// removed by removing the add-on.
func (s *Service) LogDrainDelete(ctx context.Context, appIdentity string, logDrainQueryIdentity string) (*LogDrain, error) {
	var logDrain LogDrain
	return &logDrain, s.Delete(ctx, &logDrain, fmt.Sprintf("/apps/%v/log-drains/%v", appIdentity, logDrainQueryIdentity))
}

// Info for existing log drain.
func (s *Service) LogDrainInfo(ctx context.Context, appIdentity string, logDrainQueryIdentity string) (*LogDrain, error) {
	var logDrain LogDrain
	return &logDrain, s.Get(ctx, &logDrain, fmt.Sprintf("/apps/%v/log-drains/%v", appIdentity, logDrainQueryIdentity), nil, nil)
}

type LogDrainListResult []LogDrain

// List existing log drains.
func (s *Service) LogDrainList(ctx context.Context, appIdentity string, lr *ListRange) (LogDrainListResult, error) {
	var logDrain LogDrainListResult
	return logDrain, s.Get(ctx, &logDrain, fmt.Sprintf("/apps/%v/log-drains", appIdentity), nil, lr)
}

// A log session is a reference to the http based log stream for an app.
type LogSession struct {
	CreatedAt  time.Time `json:"created_at" url:"created_at,key"`   // when log connection was created
	ID         string    `json:"id" url:"id,key"`                   // unique identifier of this log session
	LogplexURL string    `json:"logplex_url" url:"logplex_url,key"` // URL for log streaming session
	UpdatedAt  time.Time `json:"updated_at" url:"updated_at,key"`   // when log session was updated
}
type LogSessionCreateOpts struct {
	Dyno   *string `json:"dyno,omitempty" url:"dyno,omitempty,key"`     // dyno to limit results to
	Lines  *int    `json:"lines,omitempty" url:"lines,omitempty,key"`   // number of log lines to stream at once
	Source *string `json:"source,omitempty" url:"source,omitempty,key"` // log source to limit results to
	Tail   *bool   `json:"tail,omitempty" url:"tail,omitempty,key"`     // whether to stream ongoing logs
}

// Create a new log session.
func (s *Service) LogSessionCreate(ctx context.Context, appIdentity string, o LogSessionCreateOpts) (*LogSession, error) {
	var logSession LogSession
	return &logSession, s.Post(ctx, &logSession, fmt.Sprintf("/apps/%v/log-sessions", appIdentity), o)
}

// OAuth authorizations represent clients that a Heroku user has
// authorized to automate, customize or extend their usage of the
// platform. For more information please refer to the [Heroku OAuth
// documentation](https://devcenter.heroku.com/articles/oauth)
type OAuthAuthorization struct {
	AccessToken *struct {
		ExpiresIn *int `json:"expires_in" url:"expires_in,key"` // seconds until OAuth token expires; may be `null` for tokens with
		// indefinite lifetime
		ID    string `json:"id" url:"id,key"`       // unique identifier of OAuth token
		Token string `json:"token" url:"token,key"` // contents of the token to be used for authorization
	} `json:"access_token" url:"access_token,key"` // access token for this authorization
	Client *struct {
		ID          string `json:"id" url:"id,key"`                     // unique identifier of this OAuth client
		Name        string `json:"name" url:"name,key"`                 // OAuth client name
		RedirectURI string `json:"redirect_uri" url:"redirect_uri,key"` // endpoint for redirection after authorization with OAuth client
	} `json:"client" url:"client,key"` // identifier of the client that obtained this authorization, if any
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when OAuth authorization was created
	Grant     *struct {
		Code      string `json:"code" url:"code,key"`             // grant code received from OAuth web application authorization
		ExpiresIn int    `json:"expires_in" url:"expires_in,key"` // seconds until OAuth grant expires
		ID        string `json:"id" url:"id,key"`                 // unique identifier of OAuth grant
	} `json:"grant" url:"grant,key"` // this authorization's grant
	ID           string `json:"id" url:"id,key"` // unique identifier of OAuth authorization
	RefreshToken *struct {
		ExpiresIn *int `json:"expires_in" url:"expires_in,key"` // seconds until OAuth token expires; may be `null` for tokens with
		// indefinite lifetime
		ID    string `json:"id" url:"id,key"`       // unique identifier of OAuth token
		Token string `json:"token" url:"token,key"` // contents of the token to be used for authorization
	} `json:"refresh_token" url:"refresh_token,key"` // refresh token for this authorization
	Scope     []string  `json:"scope" url:"scope,key"`           // The scope of access OAuth authorization allows
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when OAuth authorization was updated
	User      struct {
		Email    string  `json:"email" url:"email,key"`         // unique email address of account
		FullName *string `json:"full_name" url:"full_name,key"` // full name of the account owner
		ID       string  `json:"id" url:"id,key"`               // unique identifier of an account
	} `json:"user" url:"user,key"` // authenticated user associated with this authorization
}
type OAuthAuthorizationCreateOpts struct {
	Client      *string `json:"client,omitempty" url:"client,omitempty,key"`           // unique identifier of this OAuth client
	Description *string `json:"description,omitempty" url:"description,omitempty,key"` // human-friendly description of this OAuth authorization
	ExpiresIn   *int    `json:"expires_in,omitempty" url:"expires_in,omitempty,key"`   // seconds until OAuth token expires; may be `null` for tokens with
	// indefinite lifetime
	Scope []string `json:"scope" url:"scope,key"` // The scope of access OAuth authorization allows
}

// Create a new OAuth authorization.
func (s *Service) OAuthAuthorizationCreate(ctx context.Context, o OAuthAuthorizationCreateOpts) (*OAuthAuthorization, error) {
	var oauthAuthorization OAuthAuthorization
	return &oauthAuthorization, s.Post(ctx, &oauthAuthorization, fmt.Sprintf("/oauth/authorizations"), o)
}

// Delete OAuth authorization.
func (s *Service) OAuthAuthorizationDelete(ctx context.Context, oauthAuthorizationIdentity string) (*OAuthAuthorization, error) {
	var oauthAuthorization OAuthAuthorization
	return &oauthAuthorization, s.Delete(ctx, &oauthAuthorization, fmt.Sprintf("/oauth/authorizations/%v", oauthAuthorizationIdentity))
}

// Info for an OAuth authorization.
func (s *Service) OAuthAuthorizationInfo(ctx context.Context, oauthAuthorizationIdentity string) (*OAuthAuthorization, error) {
	var oauthAuthorization OAuthAuthorization
	return &oauthAuthorization, s.Get(ctx, &oauthAuthorization, fmt.Sprintf("/oauth/authorizations/%v", oauthAuthorizationIdentity), nil, nil)
}

type OAuthAuthorizationListResult []OAuthAuthorization

// List OAuth authorizations.
func (s *Service) OAuthAuthorizationList(ctx context.Context, lr *ListRange) (OAuthAuthorizationListResult, error) {
	var oauthAuthorization OAuthAuthorizationListResult
	return oauthAuthorization, s.Get(ctx, &oauthAuthorization, fmt.Sprintf("/oauth/authorizations"), nil, lr)
}

// Regenerate OAuth tokens. This endpoint is only available to direct
// authorizations or privileged OAuth clients.
func (s *Service) OAuthAuthorizationRegenerate(ctx context.Context, oauthAuthorizationIdentity string) (*OAuthAuthorization, error) {
	var oauthAuthorization OAuthAuthorization
	return &oauthAuthorization, s.Post(ctx, &oauthAuthorization, fmt.Sprintf("/oauth/authorizations/%v/actions/regenerate-tokens", oauthAuthorizationIdentity), nil)
}

// OAuth clients are applications that Heroku users can authorize to
// automate, customize or extend their usage of the platform. For more
// information please refer to the [Heroku OAuth
// documentation](https://devcenter.heroku.com/articles/oauth).
type OAuthClient struct {
	CreatedAt         time.Time `json:"created_at" url:"created_at,key"`                 // when OAuth client was created
	ID                string    `json:"id" url:"id,key"`                                 // unique identifier of this OAuth client
	IgnoresDelinquent *bool     `json:"ignores_delinquent" url:"ignores_delinquent,key"` // whether the client is still operable given a delinquent account
	Name              string    `json:"name" url:"name,key"`                             // OAuth client name
	RedirectURI       string    `json:"redirect_uri" url:"redirect_uri,key"`             // endpoint for redirection after authorization with OAuth client
	Secret            string    `json:"secret" url:"secret,key"`                         // secret used to obtain OAuth authorizations under this client
	UpdatedAt         time.Time `json:"updated_at" url:"updated_at,key"`                 // when OAuth client was updated
}
type OAuthClientCreateOpts struct {
	Name        string `json:"name" url:"name,key"`                 // OAuth client name
	RedirectURI string `json:"redirect_uri" url:"redirect_uri,key"` // endpoint for redirection after authorization with OAuth client
}

// Create a new OAuth client.
func (s *Service) OAuthClientCreate(ctx context.Context, o OAuthClientCreateOpts) (*OAuthClient, error) {
	var oauthClient OAuthClient
	return &oauthClient, s.Post(ctx, &oauthClient, fmt.Sprintf("/oauth/clients"), o)
}

// Delete OAuth client.
func (s *Service) OAuthClientDelete(ctx context.Context, oauthClientIdentity string) (*OAuthClient, error) {
	var oauthClient OAuthClient
	return &oauthClient, s.Delete(ctx, &oauthClient, fmt.Sprintf("/oauth/clients/%v", oauthClientIdentity))
}

// Info for an OAuth client
func (s *Service) OAuthClientInfo(ctx context.Context, oauthClientIdentity string) (*OAuthClient, error) {
	var oauthClient OAuthClient
	return &oauthClient, s.Get(ctx, &oauthClient, fmt.Sprintf("/oauth/clients/%v", oauthClientIdentity), nil, nil)
}

type OAuthClientListResult []OAuthClient

// List OAuth clients
func (s *Service) OAuthClientList(ctx context.Context, lr *ListRange) (OAuthClientListResult, error) {
	var oauthClient OAuthClientListResult
	return oauthClient, s.Get(ctx, &oauthClient, fmt.Sprintf("/oauth/clients"), nil, lr)
}

type OAuthClientUpdateOpts struct {
	Name        *string `json:"name,omitempty" url:"name,omitempty,key"`                 // OAuth client name
	RedirectURI *string `json:"redirect_uri,omitempty" url:"redirect_uri,omitempty,key"` // endpoint for redirection after authorization with OAuth client
}

// Update OAuth client
func (s *Service) OAuthClientUpdate(ctx context.Context, oauthClientIdentity string, o OAuthClientUpdateOpts) (*OAuthClient, error) {
	var oauthClient OAuthClient
	return &oauthClient, s.Patch(ctx, &oauthClient, fmt.Sprintf("/oauth/clients/%v", oauthClientIdentity), o)
}

// Rotate credentials for an OAuth client
func (s *Service) OAuthClientRotateCredentials(ctx context.Context, oauthClientIdentity string) (*OAuthClient, error) {
	var oauthClient OAuthClient
	return &oauthClient, s.Post(ctx, &oauthClient, fmt.Sprintf("/oauth/clients/%v/actions/rotate-credentials", oauthClientIdentity), nil)
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
		ExpiresIn *int `json:"expires_in" url:"expires_in,key"` // seconds until OAuth token expires; may be `null` for tokens with
		// indefinite lifetime
		ID    string `json:"id" url:"id,key"`       // unique identifier of OAuth token
		Token string `json:"token" url:"token,key"` // contents of the token to be used for authorization
	} `json:"access_token" url:"access_token,key"` // current access token
	Authorization struct {
		ID string `json:"id" url:"id,key"` // unique identifier of OAuth authorization
	} `json:"authorization" url:"authorization,key"` // authorization for this set of tokens
	Client *struct {
		Secret string `json:"secret" url:"secret,key"` // secret used to obtain OAuth authorizations under this client
	} `json:"client" url:"client,key"` // OAuth client secret used to obtain token
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when OAuth token was created
	Grant     struct {
		Code string `json:"code" url:"code,key"` // grant code received from OAuth web application authorization
		Type string `json:"type" url:"type,key"` // type of grant requested, one of `authorization_code` or
		// `refresh_token`
	} `json:"grant" url:"grant,key"` // grant used on the underlying authorization
	ID           string `json:"id" url:"id,key"` // unique identifier of OAuth token
	RefreshToken struct {
		ExpiresIn *int `json:"expires_in" url:"expires_in,key"` // seconds until OAuth token expires; may be `null` for tokens with
		// indefinite lifetime
		ID    string `json:"id" url:"id,key"`       // unique identifier of OAuth token
		Token string `json:"token" url:"token,key"` // contents of the token to be used for authorization
	} `json:"refresh_token" url:"refresh_token,key"` // refresh token for this authorization
	Session struct {
		ID string `json:"id" url:"id,key"` // unique identifier of OAuth token
	} `json:"session" url:"session,key"` // OAuth session using this token
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when OAuth token was updated
	User      struct {
		ID string `json:"id" url:"id,key"` // unique identifier of an account
	} `json:"user" url:"user,key"` // Reference to the user associated with this token
}
type OAuthTokenCreateOpts struct {
	Client struct {
		Secret *string `json:"secret,omitempty" url:"secret,omitempty,key"` // secret used to obtain OAuth authorizations under this client
	} `json:"client" url:"client,key"`
	Grant struct {
		Code *string `json:"code,omitempty" url:"code,omitempty,key"` // grant code received from OAuth web application authorization
		Type *string `json:"type,omitempty" url:"type,omitempty,key"` // type of grant requested, one of `authorization_code` or
		// `refresh_token`
	} `json:"grant" url:"grant,key"`
	RefreshToken struct {
		Token *string `json:"token,omitempty" url:"token,omitempty,key"` // contents of the token to be used for authorization
	} `json:"refresh_token" url:"refresh_token,key"`
}

// Create a new OAuth token.
func (s *Service) OAuthTokenCreate(ctx context.Context, o OAuthTokenCreateOpts) (*OAuthToken, error) {
	var oauthToken OAuthToken
	return &oauthToken, s.Post(ctx, &oauthToken, fmt.Sprintf("/oauth/tokens"), o)
}

// Revoke OAuth access token.
func (s *Service) OAuthTokenDelete(ctx context.Context, oauthTokenIdentity string) (*OAuthToken, error) {
	var oauthToken OAuthToken
	return &oauthToken, s.Delete(ctx, &oauthToken, fmt.Sprintf("/oauth/tokens/%v", oauthTokenIdentity))
}

// Deprecated: Organizations allow you to manage access to a shared
// group of applications across your development team.
type Organization struct {
	CreatedAt             time.Time `json:"created_at" url:"created_at,key"`                           // when the organization was created
	CreditCardCollections bool      `json:"credit_card_collections" url:"credit_card_collections,key"` // whether charges incurred by the org are paid by credit card.
	Default               bool      `json:"default" url:"default,key"`                                 // whether to use this organization when none is specified
	ID                    string    `json:"id" url:"id,key"`                                           // unique identifier of organization
	MembershipLimit       *float64  `json:"membership_limit" url:"membership_limit,key"`               // upper limit of members allowed in an organization.
	Name                  string    `json:"name" url:"name,key"`                                       // unique name of organization
	ProvisionedLicenses   bool      `json:"provisioned_licenses" url:"provisioned_licenses,key"`       // whether the org is provisioned licenses by salesforce.
	Role                  *string   `json:"role" url:"role,key"`                                       // role in the organization
	Type                  string    `json:"type" url:"type,key"`                                       // type of organization.
	UpdatedAt             time.Time `json:"updated_at" url:"updated_at,key"`                           // when the organization was updated
}
type OrganizationListResult []Organization

// List organizations in which you are a member.
func (s *Service) OrganizationList(ctx context.Context, lr *ListRange) (OrganizationListResult, error) {
	var organization OrganizationListResult
	return organization, s.Get(ctx, &organization, fmt.Sprintf("/organizations"), nil, lr)
}

// Info for an organization.
func (s *Service) OrganizationInfo(ctx context.Context, organizationIdentity string) (*Organization, error) {
	var organization Organization
	return &organization, s.Get(ctx, &organization, fmt.Sprintf("/organizations/%v", organizationIdentity), nil, nil)
}

type OrganizationUpdateOpts struct {
	Default *bool   `json:"default,omitempty" url:"default,omitempty,key"` // whether to use this organization when none is specified
	Name    *string `json:"name,omitempty" url:"name,omitempty,key"`       // unique name of organization
}

// Update organization properties.
func (s *Service) OrganizationUpdate(ctx context.Context, organizationIdentity string, o OrganizationUpdateOpts) (*Organization, error) {
	var organization Organization
	return &organization, s.Patch(ctx, &organization, fmt.Sprintf("/organizations/%v", organizationIdentity), o)
}

type OrganizationCreateOpts struct {
	Address1        *string `json:"address_1,omitempty" url:"address_1,omitempty,key"`               // street address line 1
	Address2        *string `json:"address_2,omitempty" url:"address_2,omitempty,key"`               // street address line 2
	CardNumber      *string `json:"card_number,omitempty" url:"card_number,omitempty,key"`           // encrypted card number of payment method
	City            *string `json:"city,omitempty" url:"city,omitempty,key"`                         // city
	Country         *string `json:"country,omitempty" url:"country,omitempty,key"`                   // country
	Cvv             *string `json:"cvv,omitempty" url:"cvv,omitempty,key"`                           // card verification value
	ExpirationMonth *string `json:"expiration_month,omitempty" url:"expiration_month,omitempty,key"` // expiration month
	ExpirationYear  *string `json:"expiration_year,omitempty" url:"expiration_year,omitempty,key"`   // expiration year
	FirstName       *string `json:"first_name,omitempty" url:"first_name,omitempty,key"`             // the first name for payment method
	LastName        *string `json:"last_name,omitempty" url:"last_name,omitempty,key"`               // the last name for payment method
	Name            string  `json:"name" url:"name,key"`                                             // unique name of organization
	Other           *string `json:"other,omitempty" url:"other,omitempty,key"`                       // metadata
	PostalCode      *string `json:"postal_code,omitempty" url:"postal_code,omitempty,key"`           // postal code
	State           *string `json:"state,omitempty" url:"state,omitempty,key"`                       // state
}

// Create a new organization.
func (s *Service) OrganizationCreate(ctx context.Context, o OrganizationCreateOpts) (*Organization, error) {
	var organization Organization
	return &organization, s.Post(ctx, &organization, fmt.Sprintf("/organizations"), o)
}

// Delete an existing organization.
func (s *Service) OrganizationDelete(ctx context.Context, organizationIdentity string) (*Organization, error) {
	var organization Organization
	return &organization, s.Delete(ctx, &organization, fmt.Sprintf("/organizations/%v", organizationIdentity))
}

// Deprecated: A list of add-ons the Organization uses across all apps
type OrganizationAddOn struct{}
type OrganizationAddOnListForOrganizationResult []struct {
	Actions      []struct{} `json:"actions" url:"actions,key"` // provider actions for this specific add-on
	AddonService struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of this add-on-service
		Name string `json:"name" url:"name,key"` // unique name of this add-on-service
	} `json:"addon_service" url:"addon_service,key"` // identity of add-on service
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // billing application associated with this add-on
	ConfigVars []string  `json:"config_vars" url:"config_vars,key"` // config vars exposed to the owning app by this add-on
	CreatedAt  time.Time `json:"created_at" url:"created_at,key"`   // when add-on was created
	ID         string    `json:"id" url:"id,key"`                   // unique identifier of add-on
	Name       string    `json:"name" url:"name,key"`               // globally unique name of the add-on
	Plan       struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of this plan
		Name string `json:"name" url:"name,key"` // unique name of this plan
	} `json:"plan" url:"plan,key"` // identity of add-on plan
	ProviderID string    `json:"provider_id" url:"provider_id,key"` // id of this add-on with its provider
	State      string    `json:"state" url:"state,key"`             // state in the add-on's lifecycle
	UpdatedAt  time.Time `json:"updated_at" url:"updated_at,key"`   // when add-on was updated
	WebURL     *string   `json:"web_url" url:"web_url,key"`         // URL for logging into web interface of add-on (e.g. a dashboard)
}

// List add-ons used across all Organization apps
func (s *Service) OrganizationAddOnListForOrganization(ctx context.Context, organizationIdentity string, lr *ListRange) (OrganizationAddOnListForOrganizationResult, error) {
	var organizationAddOn OrganizationAddOnListForOrganizationResult
	return organizationAddOn, s.Get(ctx, &organizationAddOn, fmt.Sprintf("/organizations/%v/addons", organizationIdentity), nil, lr)
}

// Deprecated: An organization app encapsulates the organization
// specific functionality of Heroku apps.
type OrganizationApp struct {
	ArchivedAt                   *time.Time `json:"archived_at" url:"archived_at,key"`                                       // when app was archived
	BuildpackProvidedDescription *string    `json:"buildpack_provided_description" url:"buildpack_provided_description,key"` // description from buildpack of app
	CreatedAt                    time.Time  `json:"created_at" url:"created_at,key"`                                         // when app was created
	GitURL                       string     `json:"git_url" url:"git_url,key"`                                               // git repo URL of app
	ID                           string     `json:"id" url:"id,key"`                                                         // unique identifier of app
	Joined                       bool       `json:"joined" url:"joined,key"`                                                 // is the current member a collaborator on this app.
	Locked                       bool       `json:"locked" url:"locked,key"`                                                 // are other organization members forbidden from joining this app.
	Maintenance                  bool       `json:"maintenance" url:"maintenance,key"`                                       // maintenance status of app
	Name                         string     `json:"name" url:"name,key"`                                                     // unique name of app
	Organization                 *struct {
		Name string `json:"name" url:"name,key"` // unique name of organization
	} `json:"organization" url:"organization,key"` // organization that owns this app
	Owner *struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"owner" url:"owner,key"` // identity of app owner
	Region struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of region
		Name string `json:"name" url:"name,key"` // unique name of region
	} `json:"region" url:"region,key"` // identity of app region
	ReleasedAt *time.Time `json:"released_at" url:"released_at,key"` // when app was released
	RepoSize   *int       `json:"repo_size" url:"repo_size,key"`     // git repo size in bytes of app
	SlugSize   *int       `json:"slug_size" url:"slug_size,key"`     // slug size in bytes of app
	Space      *struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of space
		Name string `json:"name" url:"name,key"` // unique name of space
	} `json:"space" url:"space,key"` // identity of space
	Stack struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of stack
		Name string `json:"name" url:"name,key"` // unique name of stack
	} `json:"stack" url:"stack,key"` // identity of app stack
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when app was updated
	WebURL    string    `json:"web_url" url:"web_url,key"`       // web URL of app
}
type OrganizationAppCreateOpts struct {
	Locked       *bool   `json:"locked,omitempty" url:"locked,omitempty,key"`             // are other organization members forbidden from joining this app.
	Name         *string `json:"name,omitempty" url:"name,omitempty,key"`                 // unique name of app
	Organization *string `json:"organization,omitempty" url:"organization,omitempty,key"` // unique name of organization
	Personal     *bool   `json:"personal,omitempty" url:"personal,omitempty,key"`         // force creation of the app in the user account even if a default org
	// is set.
	Region *string `json:"region,omitempty" url:"region,omitempty,key"` // unique name of region
	Space  *string `json:"space,omitempty" url:"space,omitempty,key"`   // unique name of space
	Stack  *string `json:"stack,omitempty" url:"stack,omitempty,key"`   // unique name of stack
}

// Create a new app in the specified organization, in the default
// organization if unspecified,  or in personal account, if default
// organization is not set.
func (s *Service) OrganizationAppCreate(ctx context.Context, o OrganizationAppCreateOpts) (*OrganizationApp, error) {
	var organizationApp OrganizationApp
	return &organizationApp, s.Post(ctx, &organizationApp, fmt.Sprintf("/organizations/apps"), o)
}

type OrganizationAppListResult []OrganizationApp

// List apps in the default organization, or in personal account, if
// default organization is not set.
func (s *Service) OrganizationAppList(ctx context.Context, lr *ListRange) (OrganizationAppListResult, error) {
	var organizationApp OrganizationAppListResult
	return organizationApp, s.Get(ctx, &organizationApp, fmt.Sprintf("/organizations/apps"), nil, lr)
}

type OrganizationAppListForOrganizationResult []OrganizationApp

// List organization apps.
func (s *Service) OrganizationAppListForOrganization(ctx context.Context, organizationIdentity string, lr *ListRange) (OrganizationAppListForOrganizationResult, error) {
	var organizationApp OrganizationAppListForOrganizationResult
	return organizationApp, s.Get(ctx, &organizationApp, fmt.Sprintf("/organizations/%v/apps", organizationIdentity), nil, lr)
}

// Info for an organization app.
func (s *Service) OrganizationAppInfo(ctx context.Context, organizationAppIdentity string) (*OrganizationApp, error) {
	var organizationApp OrganizationApp
	return &organizationApp, s.Get(ctx, &organizationApp, fmt.Sprintf("/organizations/apps/%v", organizationAppIdentity), nil, nil)
}

type OrganizationAppUpdateLockedOpts struct {
	Locked bool `json:"locked" url:"locked,key"` // are other organization members forbidden from joining this app.
}

// Lock or unlock an organization app.
func (s *Service) OrganizationAppUpdateLocked(ctx context.Context, organizationAppIdentity string, o OrganizationAppUpdateLockedOpts) (*OrganizationApp, error) {
	var organizationApp OrganizationApp
	return &organizationApp, s.Patch(ctx, &organizationApp, fmt.Sprintf("/organizations/apps/%v", organizationAppIdentity), o)
}

type OrganizationAppTransferToAccountOpts struct {
	Owner string `json:"owner" url:"owner,key"` // unique email address of account
}

// Transfer an existing organization app to another Heroku account.
func (s *Service) OrganizationAppTransferToAccount(ctx context.Context, organizationAppIdentity string, o OrganizationAppTransferToAccountOpts) (*OrganizationApp, error) {
	var organizationApp OrganizationApp
	return &organizationApp, s.Patch(ctx, &organizationApp, fmt.Sprintf("/organizations/apps/%v", organizationAppIdentity), o)
}

type OrganizationAppTransferToOrganizationOpts struct {
	Owner string `json:"owner" url:"owner,key"` // unique name of organization
}

// Transfer an existing organization app to another organization.
func (s *Service) OrganizationAppTransferToOrganization(ctx context.Context, organizationAppIdentity string, o OrganizationAppTransferToOrganizationOpts) (*OrganizationApp, error) {
	var organizationApp OrganizationApp
	return &organizationApp, s.Patch(ctx, &organizationApp, fmt.Sprintf("/organizations/apps/%v", organizationAppIdentity), o)
}

// Deprecated: An organization collaborator represents an account that
// has been given access to an organization app on Heroku.
type OrganizationAppCollaborator struct {
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // app collaborator belongs to
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when collaborator was created
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of collaborator
	Role      *string   `json:"role" url:"role,key"`             // role in the organization
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when collaborator was updated
	User      struct {
		Email     string `json:"email" url:"email,key"`         // unique email address of account
		Federated bool   `json:"federated" url:"federated,key"` // whether the user is federated and belongs to an Identity Provider
		ID        string `json:"id" url:"id,key"`               // unique identifier of an account
	} `json:"user" url:"user,key"` // identity of collaborated account
}
type OrganizationAppCollaboratorCreateOpts struct {
	Permissions *[]*string `json:"permissions,omitempty" url:"permissions,omitempty,key"` // An array of permissions to give to the collaborator.
	Silent      *bool      `json:"silent,omitempty" url:"silent,omitempty,key"`           // whether to suppress email invitation when creating collaborator
	User        string     `json:"user" url:"user,key"`                                   // unique email address of account
}

// Create a new collaborator on an organization app. Use this endpoint
// instead of the `/apps/{app_id_or_name}/collaborator` endpoint when
// you want the collaborator to be granted [permissions]
// (https://devcenter.heroku.com/articles/org-users-access#roles-and-app-
// permissions) according to their role in the organization.
func (s *Service) OrganizationAppCollaboratorCreate(ctx context.Context, appIdentity string, o OrganizationAppCollaboratorCreateOpts) (*OrganizationAppCollaborator, error) {
	var organizationAppCollaborator OrganizationAppCollaborator
	return &organizationAppCollaborator, s.Post(ctx, &organizationAppCollaborator, fmt.Sprintf("/organizations/apps/%v/collaborators", appIdentity), o)
}

// Delete an existing collaborator from an organization app.
func (s *Service) OrganizationAppCollaboratorDelete(ctx context.Context, organizationAppIdentity string, organizationAppCollaboratorIdentity string) (*OrganizationAppCollaborator, error) {
	var organizationAppCollaborator OrganizationAppCollaborator
	return &organizationAppCollaborator, s.Delete(ctx, &organizationAppCollaborator, fmt.Sprintf("/organizations/apps/%v/collaborators/%v", organizationAppIdentity, organizationAppCollaboratorIdentity))
}

// Info for a collaborator on an organization app.
func (s *Service) OrganizationAppCollaboratorInfo(ctx context.Context, organizationAppIdentity string, organizationAppCollaboratorIdentity string) (*OrganizationAppCollaborator, error) {
	var organizationAppCollaborator OrganizationAppCollaborator
	return &organizationAppCollaborator, s.Get(ctx, &organizationAppCollaborator, fmt.Sprintf("/organizations/apps/%v/collaborators/%v", organizationAppIdentity, organizationAppCollaboratorIdentity), nil, nil)
}

type OrganizationAppCollaboratorUpdateOpts struct {
	Permissions []string `json:"permissions" url:"permissions,key"` // An array of permissions to give to the collaborator.
}

// Update an existing collaborator from an organization app.
func (s *Service) OrganizationAppCollaboratorUpdate(ctx context.Context, organizationAppIdentity string, organizationAppCollaboratorIdentity string, o OrganizationAppCollaboratorUpdateOpts) (*OrganizationAppCollaborator, error) {
	var organizationAppCollaborator OrganizationAppCollaborator
	return &organizationAppCollaborator, s.Patch(ctx, &organizationAppCollaborator, fmt.Sprintf("/organizations/apps/%v/collaborators/%v", organizationAppIdentity, organizationAppCollaboratorIdentity), o)
}

type OrganizationAppCollaboratorListResult []OrganizationAppCollaborator

// List collaborators on an organization app.
func (s *Service) OrganizationAppCollaboratorList(ctx context.Context, organizationAppIdentity string, lr *ListRange) (OrganizationAppCollaboratorListResult, error) {
	var organizationAppCollaborator OrganizationAppCollaboratorListResult
	return organizationAppCollaborator, s.Get(ctx, &organizationAppCollaborator, fmt.Sprintf("/organizations/apps/%v/collaborators", organizationAppIdentity), nil, lr)
}

// Deprecated: An organization app permission is a behavior that is
// assigned to a user in an organization app.
type OrganizationAppPermission struct {
	Description string `json:"description" url:"description,key"` // A description of what the app permission allows.
	Name        string `json:"name" url:"name,key"`               // The name of the app permission.
}
type OrganizationAppPermissionListResult []OrganizationAppPermission

// Lists permissions available to organizations.
func (s *Service) OrganizationAppPermissionList(ctx context.Context, lr *ListRange) (OrganizationAppPermissionListResult, error) {
	var organizationAppPermission OrganizationAppPermissionListResult
	return organizationAppPermission, s.Get(ctx, &organizationAppPermission, fmt.Sprintf("/organizations/permissions"), nil, lr)
}

// Deprecated: An organization feature represents a feature enabled on
// an organization account.
type OrganizationFeature struct {
	CreatedAt     time.Time `json:"created_at" url:"created_at,key"`         // when organization feature was created
	Description   string    `json:"description" url:"description,key"`       // description of organization feature
	DisplayName   string    `json:"display_name" url:"display_name,key"`     // user readable feature name
	DocURL        string    `json:"doc_url" url:"doc_url,key"`               // documentation URL of organization feature
	Enabled       bool      `json:"enabled" url:"enabled,key"`               // whether or not organization feature has been enabled
	FeedbackEmail string    `json:"feedback_email" url:"feedback_email,key"` // e-mail to send feedback about the feature
	ID            string    `json:"id" url:"id,key"`                         // unique identifier of organization feature
	Name          string    `json:"name" url:"name,key"`                     // unique name of organization feature
	State         string    `json:"state" url:"state,key"`                   // state of organization feature
	UpdatedAt     time.Time `json:"updated_at" url:"updated_at,key"`         // when organization feature was updated
}

// Info for an existing organization feature.
func (s *Service) OrganizationFeatureInfo(ctx context.Context, organizationIdentity string, organizationFeatureIdentity string) (*OrganizationFeature, error) {
	var organizationFeature OrganizationFeature
	return &organizationFeature, s.Get(ctx, &organizationFeature, fmt.Sprintf("/organizations/%v/features/%v", organizationIdentity, organizationFeatureIdentity), nil, nil)
}

type OrganizationFeatureListResult []OrganizationFeature

// List existing organization features.
func (s *Service) OrganizationFeatureList(ctx context.Context, organizationIdentity string, lr *ListRange) (OrganizationFeatureListResult, error) {
	var organizationFeature OrganizationFeatureListResult
	return organizationFeature, s.Get(ctx, &organizationFeature, fmt.Sprintf("/organizations/%v/features", organizationIdentity), nil, lr)
}

type OrganizationFeatureUpdateOpts struct {
	Enabled bool `json:"enabled" url:"enabled,key"` // whether or not organization feature has been enabled
}

// Update an existing organization feature.
func (s *Service) OrganizationFeatureUpdate(ctx context.Context, organizationIdentity string, organizationFeatureIdentity string, o OrganizationFeatureUpdateOpts) (*OrganizationFeature, error) {
	var organizationFeature OrganizationFeature
	return &organizationFeature, s.Patch(ctx, &organizationFeature, fmt.Sprintf("/organizations/%v/features/%v", organizationIdentity, organizationFeatureIdentity), o)
}

// Deprecated: An organization invitation represents an invite to an
// organization.
type OrganizationInvitation struct {
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when invitation was created
	ID        string    `json:"id" url:"id,key"`                 // Unique identifier of an invitation
	InvitedBy struct {
		Email string  `json:"email" url:"email,key"` // unique email address of account
		ID    string  `json:"id" url:"id,key"`       // unique identifier of an account
		Name  *string `json:"name" url:"name,key"`   // full name of the account owner
	} `json:"invited_by" url:"invited_by,key"`
	Organization struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of organization
		Name string `json:"name" url:"name,key"` // unique name of organization
	} `json:"organization" url:"organization,key"`
	Role      *string   `json:"role" url:"role,key"`             // role in the organization
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when invitation was updated
	User      struct {
		Email string  `json:"email" url:"email,key"` // unique email address of account
		ID    string  `json:"id" url:"id,key"`       // unique identifier of an account
		Name  *string `json:"name" url:"name,key"`   // full name of the account owner
	} `json:"user" url:"user,key"`
}
type OrganizationInvitationListResult []OrganizationInvitation

// Get a list of an organization's Identity Providers
func (s *Service) OrganizationInvitationList(ctx context.Context, organizationName string, lr *ListRange) (OrganizationInvitationListResult, error) {
	var organizationInvitation OrganizationInvitationListResult
	return organizationInvitation, s.Get(ctx, &organizationInvitation, fmt.Sprintf("/organizations/%v/invitations", organizationName), nil, lr)
}

type OrganizationInvitationCreateOpts struct {
	Email string  `json:"email" url:"email,key"` // unique email address of account
	Role  *string `json:"role" url:"role,key"`   // role in the organization
}

// Create Organization Invitation
func (s *Service) OrganizationInvitationCreate(ctx context.Context, organizationIdentity string, o OrganizationInvitationCreateOpts) (*OrganizationInvitation, error) {
	var organizationInvitation OrganizationInvitation
	return &organizationInvitation, s.Put(ctx, &organizationInvitation, fmt.Sprintf("/organizations/%v/invitations", organizationIdentity), o)
}

// Revoke an organization invitation.
func (s *Service) OrganizationInvitationRevoke(ctx context.Context, organizationIdentity string, organizationInvitationIdentity string) (*OrganizationInvitation, error) {
	var organizationInvitation OrganizationInvitation
	return &organizationInvitation, s.Delete(ctx, &organizationInvitation, fmt.Sprintf("/organizations/%v/invitations/%v", organizationIdentity, organizationInvitationIdentity))
}

// Get an invitation by its token
func (s *Service) OrganizationInvitationGet(ctx context.Context, organizationInvitationToken string, lr *ListRange) (*OrganizationInvitation, error) {
	var organizationInvitation OrganizationInvitation
	return &organizationInvitation, s.Get(ctx, &organizationInvitation, fmt.Sprintf("/organizations/invitations/%v", organizationInvitationToken), nil, lr)
}

type OrganizationInvitationAcceptResult struct {
	CreatedAt               time.Time `json:"created_at" url:"created_at,key"`                               // when the membership record was created
	Email                   string    `json:"email" url:"email,key"`                                         // email address of the organization member
	Federated               bool      `json:"federated" url:"federated,key"`                                 // whether the user is federated and belongs to an Identity Provider
	ID                      string    `json:"id" url:"id,key"`                                               // unique identifier of organization member
	Role                    *string   `json:"role" url:"role,key"`                                           // role in the organization
	TwoFactorAuthentication bool      `json:"two_factor_authentication" url:"two_factor_authentication,key"` // whether the Enterprise organization member has two factor
	// authentication enabled
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when the membership record was updated
	User      struct {
		Email string  `json:"email" url:"email,key"` // unique email address of account
		ID    string  `json:"id" url:"id,key"`       // unique identifier of an account
		Name  *string `json:"name" url:"name,key"`   // full name of the account owner
	} `json:"user" url:"user,key"` // user information for the membership
}

// Accept Organization Invitation
func (s *Service) OrganizationInvitationAccept(ctx context.Context, organizationInvitationToken string) (*OrganizationInvitationAcceptResult, error) {
	var organizationInvitation OrganizationInvitationAcceptResult
	return &organizationInvitation, s.Post(ctx, &organizationInvitation, fmt.Sprintf("/organizations/invitations/%v/accept", organizationInvitationToken), nil)
}

// Deprecated: An organization invoice is an itemized bill of goods for
// an organization which includes pricing and charges.
type OrganizationInvoice struct {
	AddonsTotal       int       `json:"addons_total" url:"addons_total,key"`               // total add-ons charges in on this invoice
	ChargesTotal      int       `json:"charges_total" url:"charges_total,key"`             // total charges on this invoice
	CreatedAt         time.Time `json:"created_at" url:"created_at,key"`                   // when invoice was created
	CreditsTotal      int       `json:"credits_total" url:"credits_total,key"`             // total credits on this invoice
	DatabaseTotal     int       `json:"database_total" url:"database_total,key"`           // total database charges on this invoice
	DynoUnits         float64   `json:"dyno_units" url:"dyno_units,key"`                   // The total amount of dyno units consumed across dyno types.
	ID                string    `json:"id" url:"id,key"`                                   // unique identifier of this invoice
	Number            int       `json:"number" url:"number,key"`                           // human readable invoice number
	PaymentStatus     string    `json:"payment_status" url:"payment_status,key"`           // Status of the invoice payment.
	PeriodEnd         string    `json:"period_end" url:"period_end,key"`                   // the ending date that the invoice covers
	PeriodStart       string    `json:"period_start" url:"period_start,key"`               // the starting date that this invoice covers
	PlatformTotal     int       `json:"platform_total" url:"platform_total,key"`           // total platform charges on this invoice
	State             int       `json:"state" url:"state,key"`                             // payment status for this invoice (pending, successful, failed)
	Total             int       `json:"total" url:"total,key"`                             // combined total of charges and credits on this invoice
	UpdatedAt         time.Time `json:"updated_at" url:"updated_at,key"`                   // when invoice was updated
	WeightedDynoHours float64   `json:"weighted_dyno_hours" url:"weighted_dyno_hours,key"` // The total amount of hours consumed across dyno types.
}

// Info for existing invoice.
func (s *Service) OrganizationInvoiceInfo(ctx context.Context, organizationIdentity string, organizationInvoiceIdentity int) (*OrganizationInvoice, error) {
	var organizationInvoice OrganizationInvoice
	return &organizationInvoice, s.Get(ctx, &organizationInvoice, fmt.Sprintf("/organizations/%v/invoices/%v", organizationIdentity, organizationInvoiceIdentity), nil, nil)
}

type OrganizationInvoiceListResult []OrganizationInvoice

// List existing invoices.
func (s *Service) OrganizationInvoiceList(ctx context.Context, organizationIdentity string, lr *ListRange) (OrganizationInvoiceListResult, error) {
	var organizationInvoice OrganizationInvoiceListResult
	return organizationInvoice, s.Get(ctx, &organizationInvoice, fmt.Sprintf("/organizations/%v/invoices", organizationIdentity), nil, lr)
}

// Deprecated: An organization member is an individual with access to an
// organization.
type OrganizationMember struct {
	CreatedAt               time.Time `json:"created_at" url:"created_at,key"`                               // when the membership record was created
	Email                   string    `json:"email" url:"email,key"`                                         // email address of the organization member
	Federated               bool      `json:"federated" url:"federated,key"`                                 // whether the user is federated and belongs to an Identity Provider
	ID                      string    `json:"id" url:"id,key"`                                               // unique identifier of organization member
	Role                    *string   `json:"role" url:"role,key"`                                           // role in the organization
	TwoFactorAuthentication bool      `json:"two_factor_authentication" url:"two_factor_authentication,key"` // whether the Enterprise organization member has two factor
	// authentication enabled
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when the membership record was updated
	User      struct {
		Email string  `json:"email" url:"email,key"` // unique email address of account
		ID    string  `json:"id" url:"id,key"`       // unique identifier of an account
		Name  *string `json:"name" url:"name,key"`   // full name of the account owner
	} `json:"user" url:"user,key"` // user information for the membership
}
type OrganizationMemberCreateOrUpdateOpts struct {
	Email     string  `json:"email" url:"email,key"`                             // email address of the organization member
	Federated *bool   `json:"federated,omitempty" url:"federated,omitempty,key"` // whether the user is federated and belongs to an Identity Provider
	Role      *string `json:"role" url:"role,key"`                               // role in the organization
}

// Create a new organization member, or update their role.
func (s *Service) OrganizationMemberCreateOrUpdate(ctx context.Context, organizationIdentity string, o OrganizationMemberCreateOrUpdateOpts) (*OrganizationMember, error) {
	var organizationMember OrganizationMember
	return &organizationMember, s.Put(ctx, &organizationMember, fmt.Sprintf("/organizations/%v/members", organizationIdentity), o)
}

type OrganizationMemberCreateOpts struct {
	Email     string  `json:"email" url:"email,key"`                             // email address of the organization member
	Federated *bool   `json:"federated,omitempty" url:"federated,omitempty,key"` // whether the user is federated and belongs to an Identity Provider
	Role      *string `json:"role" url:"role,key"`                               // role in the organization
}

// Create a new organization member.
func (s *Service) OrganizationMemberCreate(ctx context.Context, organizationIdentity string, o OrganizationMemberCreateOpts) (*OrganizationMember, error) {
	var organizationMember OrganizationMember
	return &organizationMember, s.Post(ctx, &organizationMember, fmt.Sprintf("/organizations/%v/members", organizationIdentity), o)
}

type OrganizationMemberUpdateOpts struct {
	Email     string  `json:"email" url:"email,key"`                             // email address of the organization member
	Federated *bool   `json:"federated,omitempty" url:"federated,omitempty,key"` // whether the user is federated and belongs to an Identity Provider
	Role      *string `json:"role" url:"role,key"`                               // role in the organization
}

// Update an organization member.
func (s *Service) OrganizationMemberUpdate(ctx context.Context, organizationIdentity string, o OrganizationMemberUpdateOpts) (*OrganizationMember, error) {
	var organizationMember OrganizationMember
	return &organizationMember, s.Patch(ctx, &organizationMember, fmt.Sprintf("/organizations/%v/members", organizationIdentity), o)
}

// Remove a member from the organization.
func (s *Service) OrganizationMemberDelete(ctx context.Context, organizationIdentity string, organizationMemberIdentity string) (*OrganizationMember, error) {
	var organizationMember OrganizationMember
	return &organizationMember, s.Delete(ctx, &organizationMember, fmt.Sprintf("/organizations/%v/members/%v", organizationIdentity, organizationMemberIdentity))
}

type OrganizationMemberListResult []OrganizationMember

// List members of the organization.
func (s *Service) OrganizationMemberList(ctx context.Context, organizationIdentity string, lr *ListRange) (OrganizationMemberListResult, error) {
	var organizationMember OrganizationMemberListResult
	return organizationMember, s.Get(ctx, &organizationMember, fmt.Sprintf("/organizations/%v/members", organizationIdentity), nil, lr)
}

type OrganizationMemberAppListResult []struct {
	ArchivedAt                   *time.Time `json:"archived_at" url:"archived_at,key"`                                       // when app was archived
	BuildpackProvidedDescription *string    `json:"buildpack_provided_description" url:"buildpack_provided_description,key"` // description from buildpack of app
	CreatedAt                    time.Time  `json:"created_at" url:"created_at,key"`                                         // when app was created
	GitURL                       string     `json:"git_url" url:"git_url,key"`                                               // git repo URL of app
	ID                           string     `json:"id" url:"id,key"`                                                         // unique identifier of app
	Joined                       bool       `json:"joined" url:"joined,key"`                                                 // is the current member a collaborator on this app.
	Locked                       bool       `json:"locked" url:"locked,key"`                                                 // are other organization members forbidden from joining this app.
	Maintenance                  bool       `json:"maintenance" url:"maintenance,key"`                                       // maintenance status of app
	Name                         string     `json:"name" url:"name,key"`                                                     // unique name of app
	Organization                 *struct {
		Name string `json:"name" url:"name,key"` // unique name of organization
	} `json:"organization" url:"organization,key"` // organization that owns this app
	Owner *struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"owner" url:"owner,key"` // identity of app owner
	Region struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of region
		Name string `json:"name" url:"name,key"` // unique name of region
	} `json:"region" url:"region,key"` // identity of app region
	ReleasedAt *time.Time `json:"released_at" url:"released_at,key"` // when app was released
	RepoSize   *int       `json:"repo_size" url:"repo_size,key"`     // git repo size in bytes of app
	SlugSize   *int       `json:"slug_size" url:"slug_size,key"`     // slug size in bytes of app
	Space      *struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of space
		Name string `json:"name" url:"name,key"` // unique name of space
	} `json:"space" url:"space,key"` // identity of space
	Stack struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of stack
		Name string `json:"name" url:"name,key"` // unique name of stack
	} `json:"stack" url:"stack,key"` // identity of app stack
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when app was updated
	WebURL    string    `json:"web_url" url:"web_url,key"`       // web URL of app
}

// List the apps of a member.
func (s *Service) OrganizationMemberAppList(ctx context.Context, organizationIdentity string, organizationMemberIdentity string, lr *ListRange) (OrganizationMemberAppListResult, error) {
	var organizationMember OrganizationMemberAppListResult
	return organizationMember, s.Get(ctx, &organizationMember, fmt.Sprintf("/organizations/%v/members/%v/apps", organizationIdentity, organizationMemberIdentity), nil, lr)
}

// Deprecated: Tracks an organization's preferences
type OrganizationPreferences struct {
	DefaultPermission *string `json:"default-permission" url:"default-permission,key"` // The default permission used when adding new members to the
	// organization
	WhitelistingEnabled *bool `json:"whitelisting-enabled" url:"whitelisting-enabled,key"` // Whether whitelisting rules should be applied to add-on installations
}

// Retrieve Organization Preferences
func (s *Service) OrganizationPreferencesList(ctx context.Context, organizationPreferencesIdentity string) (*OrganizationPreferences, error) {
	var organizationPreferences OrganizationPreferences
	return &organizationPreferences, s.Get(ctx, &organizationPreferences, fmt.Sprintf("/organizations/%v/preferences", organizationPreferencesIdentity), nil, nil)
}

type OrganizationPreferencesUpdateOpts struct {
	WhitelistingEnabled *bool `json:"whitelisting-enabled,omitempty" url:"whitelisting-enabled,omitempty,key"` // Whether whitelisting rules should be applied to add-on installations
}

// Update Organization Preferences
func (s *Service) OrganizationPreferencesUpdate(ctx context.Context, organizationPreferencesIdentity string, o OrganizationPreferencesUpdateOpts) (*OrganizationPreferences, error) {
	var organizationPreferences OrganizationPreferences
	return &organizationPreferences, s.Patch(ctx, &organizationPreferences, fmt.Sprintf("/organizations/%v/preferences", organizationPreferencesIdentity), o)
}

// An outbound-ruleset is a collection of rules that specify what hosts
// Dynos are allowed to communicate with.
type OutboundRuleset struct {
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when outbound-ruleset was created
	CreatedBy string    `json:"created_by" url:"created_by,key"` // unique email address of account
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of an outbound-ruleset
	Rules     []struct {
		FromPort int    `json:"from_port" url:"from_port,key"` // an endpoint of communication in an operating system.
		Protocol string `json:"protocol" url:"protocol,key"`   // formal standards and policies comprised of rules, procedures and
		// formats that define communication between two or more devices over a
		// network
		Target string `json:"target" url:"target,key"`   // is the target destination in CIDR notation
		ToPort int    `json:"to_port" url:"to_port,key"` // an endpoint of communication in an operating system.
	} `json:"rules" url:"rules,key"`
}

// Current outbound ruleset for a space
func (s *Service) OutboundRulesetCurrent(ctx context.Context, spaceIdentity string) (*OutboundRuleset, error) {
	var outboundRuleset OutboundRuleset
	return &outboundRuleset, s.Get(ctx, &outboundRuleset, fmt.Sprintf("/spaces/%v/outbound-ruleset", spaceIdentity), nil, nil)
}

// Info on an existing Outbound Ruleset
func (s *Service) OutboundRulesetInfo(ctx context.Context, spaceIdentity string, outboundRulesetIdentity string) (*OutboundRuleset, error) {
	var outboundRuleset OutboundRuleset
	return &outboundRuleset, s.Get(ctx, &outboundRuleset, fmt.Sprintf("/spaces/%v/outbound-rulesets/%v", spaceIdentity, outboundRulesetIdentity), nil, nil)
}

type OutboundRulesetListResult []OutboundRuleset

// List all Outbound Rulesets for a space
func (s *Service) OutboundRulesetList(ctx context.Context, spaceIdentity string, lr *ListRange) (OutboundRulesetListResult, error) {
	var outboundRuleset OutboundRulesetListResult
	return outboundRuleset, s.Get(ctx, &outboundRuleset, fmt.Sprintf("/spaces/%v/outbound-rulesets", spaceIdentity), nil, lr)
}

type OutboundRulesetCreateOpts struct {
	Rules *[]*struct {
		FromPort int    `json:"from_port" url:"from_port,key"` // an endpoint of communication in an operating system.
		Protocol string `json:"protocol" url:"protocol,key"`   // formal standards and policies comprised of rules, procedures and
		// formats that define communication between two or more devices over a
		// network
		Target string `json:"target" url:"target,key"`   // is the target destination in CIDR notation
		ToPort int    `json:"to_port" url:"to_port,key"` // an endpoint of communication in an operating system.
	} `json:"rules,omitempty" url:"rules,omitempty,key"`
}

// Create a new outbound ruleset
func (s *Service) OutboundRulesetCreate(ctx context.Context, spaceIdentity string, o OutboundRulesetCreateOpts) (*OutboundRuleset, error) {
	var outboundRuleset OutboundRuleset
	return &outboundRuleset, s.Put(ctx, &outboundRuleset, fmt.Sprintf("/spaces/%v/outbound-ruleset", spaceIdentity), o)
}

// A password reset represents a in-process password reset attempt.
type PasswordReset struct {
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when password reset was created
	User      struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"user" url:"user,key"`
}
type PasswordResetResetPasswordOpts struct {
	Email *string `json:"email,omitempty" url:"email,omitempty,key"` // unique email address of account
}

// Reset account's password. This will send a reset password link to the
// user's email address.
func (s *Service) PasswordResetResetPassword(ctx context.Context, o PasswordResetResetPasswordOpts) (*PasswordReset, error) {
	var passwordReset PasswordReset
	return &passwordReset, s.Post(ctx, &passwordReset, fmt.Sprintf("/password-resets"), o)
}

type PasswordResetCompleteResetPasswordOpts struct {
	Password             *string `json:"password,omitempty" url:"password,omitempty,key"`                           // current password on the account
	PasswordConfirmation *string `json:"password_confirmation,omitempty" url:"password_confirmation,omitempty,key"` // confirmation of the new password
}

// Complete password reset.
func (s *Service) PasswordResetCompleteResetPassword(ctx context.Context, passwordResetResetPasswordToken string, o PasswordResetCompleteResetPasswordOpts) (*PasswordReset, error) {
	var passwordReset PasswordReset
	return &passwordReset, s.Post(ctx, &passwordReset, fmt.Sprintf("/password-resets/%v/actions/finalize", passwordResetResetPasswordToken), o)
}

// A pipeline allows grouping of apps into different stages.
type Pipeline struct {
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when pipeline was created
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of pipeline
	Name      string    `json:"name" url:"name,key"`             // name of pipeline
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when pipeline was updated
}
type PipelineCreateOpts struct {
	Name string `json:"name" url:"name,key"` // name of pipeline
}

// Create a new pipeline.
func (s *Service) PipelineCreate(ctx context.Context, o PipelineCreateOpts) (*Pipeline, error) {
	var pipeline Pipeline
	return &pipeline, s.Post(ctx, &pipeline, fmt.Sprintf("/pipelines"), o)
}

// Info for existing pipeline.
func (s *Service) PipelineInfo(ctx context.Context, pipelineIdentity string) (*Pipeline, error) {
	var pipeline Pipeline
	return &pipeline, s.Get(ctx, &pipeline, fmt.Sprintf("/pipelines/%v", pipelineIdentity), nil, nil)
}

// Delete an existing pipeline.
func (s *Service) PipelineDelete(ctx context.Context, pipelineID string) (*Pipeline, error) {
	var pipeline Pipeline
	return &pipeline, s.Delete(ctx, &pipeline, fmt.Sprintf("/pipelines/%v", pipelineID))
}

type PipelineUpdateOpts struct {
	Name *string `json:"name,omitempty" url:"name,omitempty,key"` // name of pipeline
}

// Update an existing pipeline.
func (s *Service) PipelineUpdate(ctx context.Context, pipelineID string, o PipelineUpdateOpts) (*Pipeline, error) {
	var pipeline Pipeline
	return &pipeline, s.Patch(ctx, &pipeline, fmt.Sprintf("/pipelines/%v", pipelineID), o)
}

type PipelineListResult []Pipeline

// List existing pipelines.
func (s *Service) PipelineList(ctx context.Context, lr *ListRange) (PipelineListResult, error) {
	var pipeline PipelineListResult
	return pipeline, s.Get(ctx, &pipeline, fmt.Sprintf("/pipelines"), nil, lr)
}

// Information about an app's coupling to a pipeline
type PipelineCoupling struct {
	App struct {
		ID string `json:"id" url:"id,key"` // unique identifier of app
	} `json:"app" url:"app,key"` // app involved in the pipeline coupling
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when pipeline coupling was created
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of pipeline coupling
	Pipeline  struct {
		ID string `json:"id" url:"id,key"` // unique identifier of pipeline
	} `json:"pipeline" url:"pipeline,key"` // pipeline involved in the coupling
	Stage     string    `json:"stage" url:"stage,key"`           // target pipeline stage
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when pipeline coupling was updated
}
type PipelineCouplingListByPipelineResult []PipelineCoupling

// List couplings for a pipeline
func (s *Service) PipelineCouplingListByPipeline(ctx context.Context, pipelineID string, lr *ListRange) (PipelineCouplingListByPipelineResult, error) {
	var pipelineCoupling PipelineCouplingListByPipelineResult
	return pipelineCoupling, s.Get(ctx, &pipelineCoupling, fmt.Sprintf("/pipelines/%v/pipeline-couplings", pipelineID), nil, lr)
}

type PipelineCouplingListResult []PipelineCoupling

// List pipeline couplings.
func (s *Service) PipelineCouplingList(ctx context.Context, lr *ListRange) (PipelineCouplingListResult, error) {
	var pipelineCoupling PipelineCouplingListResult
	return pipelineCoupling, s.Get(ctx, &pipelineCoupling, fmt.Sprintf("/pipeline-couplings"), nil, lr)
}

type PipelineCouplingCreateOpts struct {
	App      string `json:"app" url:"app,key"`           // unique identifier of app
	Pipeline string `json:"pipeline" url:"pipeline,key"` // unique identifier of pipeline
	Stage    string `json:"stage" url:"stage,key"`       // target pipeline stage
}

// Create a new pipeline coupling.
func (s *Service) PipelineCouplingCreate(ctx context.Context, o PipelineCouplingCreateOpts) (*PipelineCoupling, error) {
	var pipelineCoupling PipelineCoupling
	return &pipelineCoupling, s.Post(ctx, &pipelineCoupling, fmt.Sprintf("/pipeline-couplings"), o)
}

// Info for an existing pipeline coupling.
func (s *Service) PipelineCouplingInfo(ctx context.Context, pipelineCouplingIdentity string) (*PipelineCoupling, error) {
	var pipelineCoupling PipelineCoupling
	return &pipelineCoupling, s.Get(ctx, &pipelineCoupling, fmt.Sprintf("/pipeline-couplings/%v", pipelineCouplingIdentity), nil, nil)
}

// Delete an existing pipeline coupling.
func (s *Service) PipelineCouplingDelete(ctx context.Context, pipelineCouplingIdentity string) (*PipelineCoupling, error) {
	var pipelineCoupling PipelineCoupling
	return &pipelineCoupling, s.Delete(ctx, &pipelineCoupling, fmt.Sprintf("/pipeline-couplings/%v", pipelineCouplingIdentity))
}

type PipelineCouplingUpdateOpts struct {
	Stage *string `json:"stage,omitempty" url:"stage,omitempty,key"` // target pipeline stage
}

// Update an existing pipeline coupling.
func (s *Service) PipelineCouplingUpdate(ctx context.Context, pipelineCouplingIdentity string, o PipelineCouplingUpdateOpts) (*PipelineCoupling, error) {
	var pipelineCoupling PipelineCoupling
	return &pipelineCoupling, s.Patch(ctx, &pipelineCoupling, fmt.Sprintf("/pipeline-couplings/%v", pipelineCouplingIdentity), o)
}

// Info for an existing app pipeline coupling.
func (s *Service) PipelineCouplingInfoByApp(ctx context.Context, appIdentity string) (*PipelineCoupling, error) {
	var pipelineCoupling PipelineCoupling
	return &pipelineCoupling, s.Get(ctx, &pipelineCoupling, fmt.Sprintf("/apps/%v/pipeline-couplings", appIdentity), nil, nil)
}

// Promotions allow you to move code from an app in a pipeline to all
// targets
type PipelinePromotion struct {
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when promotion was created
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of promotion
	Pipeline  struct {
		ID string `json:"id" url:"id,key"` // unique identifier of pipeline
	} `json:"pipeline" url:"pipeline,key"` // the pipeline which the promotion belongs to
	Source struct {
		App struct {
			ID string `json:"id" url:"id,key"` // unique identifier of app
		} `json:"app" url:"app,key"` // the app which was promoted from
		Release struct {
			ID string `json:"id" url:"id,key"` // unique identifier of release
		} `json:"release" url:"release,key"` // the release used to promoted from
	} `json:"source" url:"source,key"` // the app being promoted from
	Status    string     `json:"status" url:"status,key"`         // status of promotion
	UpdatedAt *time.Time `json:"updated_at" url:"updated_at,key"` // when promotion was updated
}
type PipelinePromotionCreateOpts struct {
	Pipeline struct {
		ID string `json:"id" url:"id,key"` // unique identifier of pipeline
	} `json:"pipeline" url:"pipeline,key"` // pipeline involved in the promotion
	Source struct {
		App *struct {
			ID *string `json:"id,omitempty" url:"id,omitempty,key"` // unique identifier of app
		} `json:"app,omitempty" url:"app,omitempty,key"` // the app which was promoted from
	} `json:"source" url:"source,key"` // the app being promoted from
	Targets []struct {
		App *struct {
			ID *string `json:"id,omitempty" url:"id,omitempty,key"` // unique identifier of app
		} `json:"app,omitempty" url:"app,omitempty,key"` // the app is being promoted to
	} `json:"targets" url:"targets,key"`
}

// Create a new promotion.
func (s *Service) PipelinePromotionCreate(ctx context.Context, o PipelinePromotionCreateOpts) (*PipelinePromotion, error) {
	var pipelinePromotion PipelinePromotion
	return &pipelinePromotion, s.Post(ctx, &pipelinePromotion, fmt.Sprintf("/pipeline-promotions"), o)
}

// Info for existing pipeline promotion.
func (s *Service) PipelinePromotionInfo(ctx context.Context, pipelinePromotionIdentity string) (*PipelinePromotion, error) {
	var pipelinePromotion PipelinePromotion
	return &pipelinePromotion, s.Get(ctx, &pipelinePromotion, fmt.Sprintf("/pipeline-promotions/%v", pipelinePromotionIdentity), nil, nil)
}

// Promotion targets represent an individual app being promoted to
type PipelinePromotionTarget struct {
	App struct {
		ID string `json:"id" url:"id,key"` // unique identifier of app
	} `json:"app" url:"app,key"` // the app which was promoted to
	ErrorMessage      *string `json:"error_message" url:"error_message,key"` // an error message for why the promotion failed
	ID                string  `json:"id" url:"id,key"`                       // unique identifier of promotion target
	PipelinePromotion struct {
		ID string `json:"id" url:"id,key"` // unique identifier of promotion
	} `json:"pipeline_promotion" url:"pipeline_promotion,key"` // the promotion which the target belongs to
	Release *struct {
		ID string `json:"id" url:"id,key"` // unique identifier of release
	} `json:"release" url:"release,key"` // the release which was created on the target app
	Status string `json:"status" url:"status,key"` // status of promotion
}
type PipelinePromotionTargetListResult []PipelinePromotionTarget

// List promotion targets belonging to an existing promotion.
func (s *Service) PipelinePromotionTargetList(ctx context.Context, pipelinePromotionID string, lr *ListRange) (PipelinePromotionTargetListResult, error) {
	var pipelinePromotionTarget PipelinePromotionTargetListResult
	return pipelinePromotionTarget, s.Get(ctx, &pipelinePromotionTarget, fmt.Sprintf("/pipeline-promotions/%v/promotion-targets", pipelinePromotionID), nil, lr)
}

// Plans represent different configurations of add-ons that may be added
// to apps. Endpoints under add-on services can be accessed without
// authentication.
type Plan struct {
	AddonService struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of this add-on-service
		Name string `json:"name" url:"name,key"` // unique name of this add-on-service
	} `json:"addon_service" url:"addon_service,key"` // identity of add-on service
	Compliance                       *[]string `json:"compliance" url:"compliance,key"`                                                   // the compliance regimes applied to an add-on plan
	CreatedAt                        time.Time `json:"created_at" url:"created_at,key"`                                                   // when plan was created
	Default                          bool      `json:"default" url:"default,key"`                                                         // whether this plan is the default for its add-on service
	Description                      string    `json:"description" url:"description,key"`                                                 // description of plan
	HumanName                        string    `json:"human_name" url:"human_name,key"`                                                   // human readable name of the add-on plan
	ID                               string    `json:"id" url:"id,key"`                                                                   // unique identifier of this plan
	InstallableInsidePrivateNetwork  bool      `json:"installable_inside_private_network" url:"installable_inside_private_network,key"`   // whether this plan is installable to a Private Spaces app
	InstallableOutsidePrivateNetwork bool      `json:"installable_outside_private_network" url:"installable_outside_private_network,key"` // whether this plan is installable to a Common Runtime app
	Name                             string    `json:"name" url:"name,key"`                                                               // unique name of this plan
	Price                            struct {
		Cents int    `json:"cents" url:"cents,key"` // price in cents per unit of plan
		Unit  string `json:"unit" url:"unit,key"`   // unit of price for plan
	} `json:"price" url:"price,key"` // price
	SpaceDefault bool      `json:"space_default" url:"space_default,key"` // whether this plan is the default for apps in Private Spaces
	State        string    `json:"state" url:"state,key"`                 // release status for plan
	UpdatedAt    time.Time `json:"updated_at" url:"updated_at,key"`       // when plan was updated
	Visible      bool      `json:"visible" url:"visible,key"`             // whether this plan is publicly visible
}

// Info for existing plan.
func (s *Service) PlanInfo(ctx context.Context, planIdentity string) (*Plan, error) {
	var plan Plan
	return &plan, s.Get(ctx, &plan, fmt.Sprintf("/plans/%v", planIdentity), nil, nil)
}

// Info for existing plan by Add-on.
func (s *Service) PlanInfoByAddOn(ctx context.Context, addOnServiceIdentity string, planIdentity string) (*Plan, error) {
	var plan Plan
	return &plan, s.Get(ctx, &plan, fmt.Sprintf("/addon-services/%v/plans/%v", addOnServiceIdentity, planIdentity), nil, nil)
}

type PlanListByAddOnResult []Plan

// List existing plans by Add-on.
func (s *Service) PlanListByAddOn(ctx context.Context, addOnServiceIdentity string, lr *ListRange) (PlanListByAddOnResult, error) {
	var plan PlanListByAddOnResult
	return plan, s.Get(ctx, &plan, fmt.Sprintf("/addon-services/%v/plans", addOnServiceIdentity), nil, lr)
}

// Rate Limit represents the number of request tokens each account
// holds. Requests to this endpoint do not count towards the rate limit.
type RateLimit struct {
	Remaining int `json:"remaining" url:"remaining,key"` // allowed requests remaining in current interval
}

// Info for rate limits.
func (s *Service) RateLimitInfo(ctx context.Context) (*RateLimit, error) {
	var rateLimit RateLimit
	return &rateLimit, s.Get(ctx, &rateLimit, fmt.Sprintf("/account/rate-limits"), nil, nil)
}

// A region represents a geographic location in which your application
// may run.
type Region struct {
	Country        string    `json:"country" url:"country,key"`                 // country where the region exists
	CreatedAt      time.Time `json:"created_at" url:"created_at,key"`           // when region was created
	Description    string    `json:"description" url:"description,key"`         // description of region
	ID             string    `json:"id" url:"id,key"`                           // unique identifier of region
	Locale         string    `json:"locale" url:"locale,key"`                   // area in the country where the region exists
	Name           string    `json:"name" url:"name,key"`                       // unique name of region
	PrivateCapable bool      `json:"private_capable" url:"private_capable,key"` // whether or not region is available for creating a Private Space
	Provider       struct {
		Name   string `json:"name" url:"name,key"`     // name of provider
		Region string `json:"region" url:"region,key"` // region name used by provider
	} `json:"provider" url:"provider,key"` // provider of underlying substrate
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when region was updated
}

// Info for existing region.
func (s *Service) RegionInfo(ctx context.Context, regionIdentity string) (*Region, error) {
	var region Region
	return &region, s.Get(ctx, &region, fmt.Sprintf("/regions/%v", regionIdentity), nil, nil)
}

type RegionListResult []Region

// List existing regions.
func (s *Service) RegionList(ctx context.Context, lr *ListRange) (RegionListResult, error) {
	var region RegionListResult
	return region, s.Get(ctx, &region, fmt.Sprintf("/regions"), nil, lr)
}

// A release represents a combination of code, config vars and add-ons
// for an app on Heroku.
type Release struct {
	AddonPlanNames []string `json:"addon_plan_names" url:"addon_plan_names,key"` // add-on plans installed on the app for this release
	App            struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // app involved in the release
	CreatedAt   time.Time `json:"created_at" url:"created_at,key"`   // when release was created
	Current     bool      `json:"current" url:"current,key"`         // indicates this release as being the current one for the app
	Description string    `json:"description" url:"description,key"` // description of changes in this release
	ID          string    `json:"id" url:"id,key"`                   // unique identifier of release
	Slug        *struct {
		ID string `json:"id" url:"id,key"` // unique identifier of slug
	} `json:"slug" url:"slug,key"` // slug running in this release
	Status    string    `json:"status" url:"status,key"`         // current status of the release
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when release was updated
	User      struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"user" url:"user,key"` // user that created the release
	Version int `json:"version" url:"version,key"` // unique version assigned to the release
}

// Info for existing release.
func (s *Service) ReleaseInfo(ctx context.Context, appIdentity string, releaseIdentity string) (*Release, error) {
	var release Release
	return &release, s.Get(ctx, &release, fmt.Sprintf("/apps/%v/releases/%v", appIdentity, releaseIdentity), nil, nil)
}

type ReleaseListResult []Release

// List existing releases.
func (s *Service) ReleaseList(ctx context.Context, appIdentity string, lr *ListRange) (ReleaseListResult, error) {
	var release ReleaseListResult
	return release, s.Get(ctx, &release, fmt.Sprintf("/apps/%v/releases", appIdentity), nil, lr)
}

type ReleaseCreateOpts struct {
	Description *string `json:"description,omitempty" url:"description,omitempty,key"` // description of changes in this release
	Slug        string  `json:"slug" url:"slug,key"`                                   // unique identifier of slug
}

// Create new release.
func (s *Service) ReleaseCreate(ctx context.Context, appIdentity string, o ReleaseCreateOpts) (*Release, error) {
	var release Release
	return &release, s.Post(ctx, &release, fmt.Sprintf("/apps/%v/releases", appIdentity), o)
}

type ReleaseRollbackOpts struct {
	Release string `json:"release" url:"release,key"` // unique identifier of release
}

// Rollback to an existing release.
func (s *Service) ReleaseRollback(ctx context.Context, appIdentity string, o ReleaseRollbackOpts) (*Release, error) {
	var release Release
	return &release, s.Post(ctx, &release, fmt.Sprintf("/apps/%v/releases", appIdentity), o)
}

// A slug is a snapshot of your application code that is ready to run on
// the platform.
type Slug struct {
	Blob struct {
		Method string `json:"method" url:"method,key"` // method to be used to interact with the slug blob
		URL    string `json:"url" url:"url,key"`       // URL to interact with the slug blob
	} `json:"blob" url:"blob,key"` // pointer to the url where clients can fetch or store the actual
	// release binary
	BuildpackProvidedDescription *string `json:"buildpack_provided_description" url:"buildpack_provided_description,key"` // description from buildpack of slug
	Checksum                     *string `json:"checksum" url:"checksum,key"`                                             // an optional checksum of the slug for verifying its integrity
	Commit                       *string `json:"commit" url:"commit,key"`                                                 // identification of the code with your version control system (eg: SHA
	// of the git HEAD)
	CommitDescription *string           `json:"commit_description" url:"commit_description,key"` // an optional description of the provided commit
	CreatedAt         time.Time         `json:"created_at" url:"created_at,key"`                 // when slug was created
	ID                string            `json:"id" url:"id,key"`                                 // unique identifier of slug
	ProcessTypes      map[string]string `json:"process_types" url:"process_types,key"`           // hash mapping process type names to their respective command
	Size              *int              `json:"size" url:"size,key"`                             // size of slug, in bytes
	Stack             struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of stack
		Name string `json:"name" url:"name,key"` // unique name of stack
	} `json:"stack" url:"stack,key"` // identity of slug stack
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when slug was updated
}

// Info for existing slug.
func (s *Service) SlugInfo(ctx context.Context, appIdentity string, slugIdentity string) (*Slug, error) {
	var slug Slug
	return &slug, s.Get(ctx, &slug, fmt.Sprintf("/apps/%v/slugs/%v", appIdentity, slugIdentity), nil, nil)
}

type SlugCreateOpts struct {
	BuildpackProvidedDescription *string `json:"buildpack_provided_description,omitempty" url:"buildpack_provided_description,omitempty,key"` // description from buildpack of slug
	Checksum                     *string `json:"checksum,omitempty" url:"checksum,omitempty,key"`                                             // an optional checksum of the slug for verifying its integrity
	Commit                       *string `json:"commit,omitempty" url:"commit,omitempty,key"`                                                 // identification of the code with your version control system (eg: SHA
	// of the git HEAD)
	CommitDescription *string           `json:"commit_description,omitempty" url:"commit_description,omitempty,key"` // an optional description of the provided commit
	ProcessTypes      map[string]string `json:"process_types" url:"process_types,key"`                               // hash mapping process type names to their respective command
	Stack             *string           `json:"stack,omitempty" url:"stack,omitempty,key"`                           // unique name of stack
}

// Create a new slug. For more information please refer to [Deploying
// Slugs using the Platform
// API](https://devcenter.heroku.com/articles/platform-api-deploying-slug
// s).
func (s *Service) SlugCreate(ctx context.Context, appIdentity string, o SlugCreateOpts) (*Slug, error) {
	var slug Slug
	return &slug, s.Post(ctx, &slug, fmt.Sprintf("/apps/%v/slugs", appIdentity), o)
}

// SMS numbers are used for recovery on accounts with two-factor
// authentication enabled.
type SmsNumber struct {
	SmsNumber *string `json:"sms_number" url:"sms_number,key"` // SMS number of account
}

// Recover an account using an SMS recovery code
func (s *Service) SmsNumberSMSNumber(ctx context.Context, accountIdentity string) (*SmsNumber, error) {
	var smsNumber SmsNumber
	return &smsNumber, s.Get(ctx, &smsNumber, fmt.Sprintf("/users/%v/sms-number", accountIdentity), nil, nil)
}

// Recover an account using an SMS recovery code
func (s *Service) SmsNumberRecover(ctx context.Context, accountIdentity string) (*SmsNumber, error) {
	var smsNumber SmsNumber
	return &smsNumber, s.Post(ctx, &smsNumber, fmt.Sprintf("/users/%v/sms-number/actions/recover", accountIdentity), nil)
}

// Confirm an SMS number change with a confirmation code
func (s *Service) SmsNumberConfirm(ctx context.Context, accountIdentity string) (*SmsNumber, error) {
	var smsNumber SmsNumber
	return &smsNumber, s.Post(ctx, &smsNumber, fmt.Sprintf("/users/%v/sms-number/actions/confirm", accountIdentity), nil)
}

// SNI Endpoint is a public address serving a custom SSL cert for HTTPS
// traffic, using the SNI TLS extension, to a Heroku app.
type SniEndpoint struct {
	CertificateChain string `json:"certificate_chain" url:"certificate_chain,key"` // raw contents of the public certificate chain (eg: .crt or .pem file)
	CName            string `json:"cname" url:"cname,key"`                         // deprecated; refer to GET /apps/:id/domains for valid CNAMEs for this
	// app
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when endpoint was created
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of this SNI endpoint
	Name      string    `json:"name" url:"name,key"`             // unique name for SNI endpoint
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when SNI endpoint was updated
}
type SniEndpointCreateOpts struct {
	CertificateChain string `json:"certificate_chain" url:"certificate_chain,key"` // raw contents of the public certificate chain (eg: .crt or .pem file)
	PrivateKey       string `json:"private_key" url:"private_key,key"`             // contents of the private key (eg .key file)
}

// Create a new SNI endpoint.
func (s *Service) SniEndpointCreate(ctx context.Context, appIdentity string, o SniEndpointCreateOpts) (*SniEndpoint, error) {
	var sniEndpoint SniEndpoint
	return &sniEndpoint, s.Post(ctx, &sniEndpoint, fmt.Sprintf("/apps/%v/sni-endpoints", appIdentity), o)
}

// Delete existing SNI endpoint.
func (s *Service) SniEndpointDelete(ctx context.Context, appIdentity string, sniEndpointIdentity string) (*SniEndpoint, error) {
	var sniEndpoint SniEndpoint
	return &sniEndpoint, s.Delete(ctx, &sniEndpoint, fmt.Sprintf("/apps/%v/sni-endpoints/%v", appIdentity, sniEndpointIdentity))
}

// Info for existing SNI endpoint.
func (s *Service) SniEndpointInfo(ctx context.Context, appIdentity string, sniEndpointIdentity string) (*SniEndpoint, error) {
	var sniEndpoint SniEndpoint
	return &sniEndpoint, s.Get(ctx, &sniEndpoint, fmt.Sprintf("/apps/%v/sni-endpoints/%v", appIdentity, sniEndpointIdentity), nil, nil)
}

type SniEndpointListResult []SniEndpoint

// List existing SNI endpoints.
func (s *Service) SniEndpointList(ctx context.Context, appIdentity string, lr *ListRange) (SniEndpointListResult, error) {
	var sniEndpoint SniEndpointListResult
	return sniEndpoint, s.Get(ctx, &sniEndpoint, fmt.Sprintf("/apps/%v/sni-endpoints", appIdentity), nil, lr)
}

type SniEndpointUpdateOpts struct {
	CertificateChain string `json:"certificate_chain" url:"certificate_chain,key"` // raw contents of the public certificate chain (eg: .crt or .pem file)
	PrivateKey       string `json:"private_key" url:"private_key,key"`             // contents of the private key (eg .key file)
}

// Update an existing SNI endpoint.
func (s *Service) SniEndpointUpdate(ctx context.Context, appIdentity string, sniEndpointIdentity string, o SniEndpointUpdateOpts) (*SniEndpoint, error) {
	var sniEndpoint SniEndpoint
	return &sniEndpoint, s.Patch(ctx, &sniEndpoint, fmt.Sprintf("/apps/%v/sni-endpoints/%v", appIdentity, sniEndpointIdentity), o)
}

// A source is a location for uploading and downloading an application's
// source code.
type Source struct {
	SourceBlob struct {
		GetURL string `json:"get_url" url:"get_url,key"` // URL to download the source
		PutURL string `json:"put_url" url:"put_url,key"` // URL to upload the source
	} `json:"source_blob" url:"source_blob,key"` // pointer to the URL where clients can fetch or store the source
}

// Create URLs for uploading and downloading source.
func (s *Service) SourceCreate(ctx context.Context) (*Source, error) {
	var source Source
	return &source, s.Post(ctx, &source, fmt.Sprintf("/sources"), nil)
}

// Create URLs for uploading and downloading source. Deprecated in favor
// of `POST /sources`
func (s *Service) SourceCreateDeprecated(ctx context.Context, appIdentity string) (*Source, error) {
	var source Source
	return &source, s.Post(ctx, &source, fmt.Sprintf("/apps/%v/sources", appIdentity), nil)
}

// A space is an isolated, highly available, secure app execution
// environments, running in the modern VPC substrate.
type Space struct {
	CreatedAt    time.Time `json:"created_at" url:"created_at,key"` // when space was created
	ID           string    `json:"id" url:"id,key"`                 // unique identifier of space
	Name         string    `json:"name" url:"name,key"`             // unique name of space
	Organization struct {
		Name string `json:"name" url:"name,key"` // unique name of organization
	} `json:"organization" url:"organization,key"` // organization that owns this space
	Region struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of region
		Name string `json:"name" url:"name,key"` // unique name of region
	} `json:"region" url:"region,key"` // identity of space region
	Shield bool   `json:"shield" url:"shield,key"` // true if this space has shield enabled
	State  string `json:"state" url:"state,key"`   // availability of this space
	Team   struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of team
		Name string `json:"name" url:"name,key"` // unique name of team
	} `json:"team" url:"team,key"` // team that owns this space
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when space was updated
}
type SpaceListResult []Space

// List existing spaces.
func (s *Service) SpaceList(ctx context.Context, lr *ListRange) (SpaceListResult, error) {
	var space SpaceListResult
	return space, s.Get(ctx, &space, fmt.Sprintf("/spaces"), nil, lr)
}

// Info for existing space.
func (s *Service) SpaceInfo(ctx context.Context, spaceIdentity string) (*Space, error) {
	var space Space
	return &space, s.Get(ctx, &space, fmt.Sprintf("/spaces/%v", spaceIdentity), nil, nil)
}

type SpaceUpdateOpts struct {
	Name *string `json:"name,omitempty" url:"name,omitempty,key"` // unique name of space
}

// Update an existing space.
func (s *Service) SpaceUpdate(ctx context.Context, spaceIdentity string, o SpaceUpdateOpts) (*Space, error) {
	var space Space
	return &space, s.Patch(ctx, &space, fmt.Sprintf("/spaces/%v", spaceIdentity), o)
}

// Delete an existing space.
func (s *Service) SpaceDelete(ctx context.Context, spaceIdentity string) (*Space, error) {
	var space Space
	return &space, s.Delete(ctx, &space, fmt.Sprintf("/spaces/%v", spaceIdentity))
}

type SpaceCreateOpts struct {
	Name         string  `json:"name" url:"name,key"`                         // unique name of space
	Organization string  `json:"organization" url:"organization,key"`         // unique name of organization
	Region       *string `json:"region,omitempty" url:"region,omitempty,key"` // unique identifier of region
	Shield       *bool   `json:"shield,omitempty" url:"shield,omitempty,key"` // true if this space has shield enabled
}

// Create a new space.
func (s *Service) SpaceCreate(ctx context.Context, o SpaceCreateOpts) (*Space, error) {
	var space Space
	return &space, s.Post(ctx, &space, fmt.Sprintf("/spaces"), o)
}

// Space access represents the permissions a particular user has on a
// particular space.
type SpaceAppAccess struct {
	CreatedAt   time.Time `json:"created_at" url:"created_at,key"` // when space was created
	ID          string    `json:"id" url:"id,key"`                 // unique identifier of space
	Permissions []struct {
		Description string `json:"description" url:"description,key"`
		Name        string `json:"name" url:"name,key"`
	} `json:"permissions" url:"permissions,key"` // user space permissions
	Space struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"space" url:"space,key"` // space user belongs to
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when space was updated
	User      struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"user" url:"user,key"` // identity of user account
}

// List permissions for a given user on a given space.
func (s *Service) SpaceAppAccessInfo(ctx context.Context, spaceIdentity string, accountIdentity string) (*SpaceAppAccess, error) {
	var spaceAppAccess SpaceAppAccess
	return &spaceAppAccess, s.Get(ctx, &spaceAppAccess, fmt.Sprintf("/spaces/%v/members/%v", spaceIdentity, accountIdentity), nil, nil)
}

type SpaceAppAccessUpdateOpts struct {
	Permissions *[]*struct {
		Name *string `json:"name,omitempty" url:"name,omitempty,key"`
	} `json:"permissions,omitempty" url:"permissions,omitempty,key"`
}

// Update an existing user's set of permissions on a space.
func (s *Service) SpaceAppAccessUpdate(ctx context.Context, spaceIdentity string, accountIdentity string, o SpaceAppAccessUpdateOpts) (*SpaceAppAccess, error) {
	var spaceAppAccess SpaceAppAccess
	return &spaceAppAccess, s.Patch(ctx, &spaceAppAccess, fmt.Sprintf("/spaces/%v/members/%v", spaceIdentity, accountIdentity), o)
}

type SpaceAppAccessListResult []SpaceAppAccess

// List all users and their permissions on a space.
func (s *Service) SpaceAppAccessList(ctx context.Context, spaceIdentity string, lr *ListRange) (SpaceAppAccessListResult, error) {
	var spaceAppAccess SpaceAppAccessListResult
	return spaceAppAccess, s.Get(ctx, &spaceAppAccess, fmt.Sprintf("/spaces/%v/members", spaceIdentity), nil, lr)
}

// Network address translation (NAT) for stable outbound IP addresses
// from a space
type SpaceNat struct {
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when network address translation for a space was created
	Sources   []string  `json:"sources" url:"sources,key"`       // potential IPs from which outbound network traffic will originate
	State     string    `json:"state" url:"state,key"`           // availability of network address translation for a space
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when network address translation for a space was updated
}

// Current state of network address translation for a space.
func (s *Service) SpaceNatInfo(ctx context.Context, spaceIdentity string) (*SpaceNat, error) {
	var spaceNat SpaceNat
	return &spaceNat, s.Get(ctx, &spaceNat, fmt.Sprintf("/spaces/%v/nat", spaceIdentity), nil, nil)
}

// [SSL Endpoint](https://devcenter.heroku.com/articles/ssl-endpoint) is
// a public address serving custom SSL cert for HTTPS traffic to a
// Heroku app. Note that an app must have the `ssl:endpoint` add-on
// installed before it can provision an SSL Endpoint using these APIs.
type SSLEndpoint struct {
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // application associated with this ssl-endpoint
	CertificateChain string    `json:"certificate_chain" url:"certificate_chain,key"` // raw contents of the public certificate chain (eg: .crt or .pem file)
	CName            string    `json:"cname" url:"cname,key"`                         // canonical name record, the address to point a domain at
	CreatedAt        time.Time `json:"created_at" url:"created_at,key"`               // when endpoint was created
	ID               string    `json:"id" url:"id,key"`                               // unique identifier of this SSL endpoint
	Name             string    `json:"name" url:"name,key"`                           // unique name for SSL endpoint
	UpdatedAt        time.Time `json:"updated_at" url:"updated_at,key"`               // when endpoint was updated
}
type SSLEndpointCreateOpts struct {
	CertificateChain string `json:"certificate_chain" url:"certificate_chain,key"`       // raw contents of the public certificate chain (eg: .crt or .pem file)
	Preprocess       *bool  `json:"preprocess,omitempty" url:"preprocess,omitempty,key"` // allow Heroku to modify an uploaded public certificate chain if deemed
	// advantageous by adding missing intermediaries, stripping unnecessary
	// ones, etc.
	PrivateKey string `json:"private_key" url:"private_key,key"` // contents of the private key (eg .key file)
}

// Create a new SSL endpoint.
func (s *Service) SSLEndpointCreate(ctx context.Context, appIdentity string, o SSLEndpointCreateOpts) (*SSLEndpoint, error) {
	var sslEndpoint SSLEndpoint
	return &sslEndpoint, s.Post(ctx, &sslEndpoint, fmt.Sprintf("/apps/%v/ssl-endpoints", appIdentity), o)
}

// Delete existing SSL endpoint.
func (s *Service) SSLEndpointDelete(ctx context.Context, appIdentity string, sslEndpointIdentity string) (*SSLEndpoint, error) {
	var sslEndpoint SSLEndpoint
	return &sslEndpoint, s.Delete(ctx, &sslEndpoint, fmt.Sprintf("/apps/%v/ssl-endpoints/%v", appIdentity, sslEndpointIdentity))
}

// Info for existing SSL endpoint.
func (s *Service) SSLEndpointInfo(ctx context.Context, appIdentity string, sslEndpointIdentity string) (*SSLEndpoint, error) {
	var sslEndpoint SSLEndpoint
	return &sslEndpoint, s.Get(ctx, &sslEndpoint, fmt.Sprintf("/apps/%v/ssl-endpoints/%v", appIdentity, sslEndpointIdentity), nil, nil)
}

type SSLEndpointListResult []SSLEndpoint

// List existing SSL endpoints.
func (s *Service) SSLEndpointList(ctx context.Context, appIdentity string, lr *ListRange) (SSLEndpointListResult, error) {
	var sslEndpoint SSLEndpointListResult
	return sslEndpoint, s.Get(ctx, &sslEndpoint, fmt.Sprintf("/apps/%v/ssl-endpoints", appIdentity), nil, lr)
}

type SSLEndpointUpdateOpts struct {
	CertificateChain *string `json:"certificate_chain,omitempty" url:"certificate_chain,omitempty,key"` // raw contents of the public certificate chain (eg: .crt or .pem file)
	Preprocess       *bool   `json:"preprocess,omitempty" url:"preprocess,omitempty,key"`               // allow Heroku to modify an uploaded public certificate chain if deemed
	// advantageous by adding missing intermediaries, stripping unnecessary
	// ones, etc.
	PrivateKey *string `json:"private_key,omitempty" url:"private_key,omitempty,key"` // contents of the private key (eg .key file)
	Rollback   *bool   `json:"rollback,omitempty" url:"rollback,omitempty,key"`       // indicates that a rollback should be performed
}

// Update an existing SSL endpoint.
func (s *Service) SSLEndpointUpdate(ctx context.Context, appIdentity string, sslEndpointIdentity string, o SSLEndpointUpdateOpts) (*SSLEndpoint, error) {
	var sslEndpoint SSLEndpoint
	return &sslEndpoint, s.Patch(ctx, &sslEndpoint, fmt.Sprintf("/apps/%v/ssl-endpoints/%v", appIdentity, sslEndpointIdentity), o)
}

// Stacks are the different application execution environments available
// in the Heroku platform.
type Stack struct {
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when stack was introduced
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of stack
	Name      string    `json:"name" url:"name,key"`             // unique name of stack
	State     string    `json:"state" url:"state,key"`           // availability of this stack: beta, deprecated or public
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when stack was last modified
}

// Stack info.
func (s *Service) StackInfo(ctx context.Context, stackIdentity string) (*Stack, error) {
	var stack Stack
	return &stack, s.Get(ctx, &stack, fmt.Sprintf("/stacks/%v", stackIdentity), nil, nil)
}

type StackListResult []Stack

// List available stacks.
func (s *Service) StackList(ctx context.Context, lr *ListRange) (StackListResult, error) {
	var stack StackListResult
	return stack, s.Get(ctx, &stack, fmt.Sprintf("/stacks"), nil, lr)
}

// Teams allow you to manage access to a shared group of applications
// and other resources.
type Team struct {
	CreatedAt             time.Time `json:"created_at" url:"created_at,key"`                           // when the team was created
	CreditCardCollections bool      `json:"credit_card_collections" url:"credit_card_collections,key"` // whether charges incurred by the team are paid by credit card.
	Default               bool      `json:"default" url:"default,key"`                                 // whether to use this team when none is specified
	ID                    string    `json:"id" url:"id,key"`                                           // unique identifier of team
	MembershipLimit       *float64  `json:"membership_limit" url:"membership_limit,key"`               // upper limit of members allowed in a team.
	Name                  string    `json:"name" url:"name,key"`                                       // unique name of team
	ProvisionedLicenses   bool      `json:"provisioned_licenses" url:"provisioned_licenses,key"`       // whether the team is provisioned licenses by salesforce.
	Role                  *string   `json:"role" url:"role,key"`                                       // role in the team
	Type                  string    `json:"type" url:"type,key"`                                       // type of team.
	UpdatedAt             time.Time `json:"updated_at" url:"updated_at,key"`                           // when the team was updated
}
type TeamListResult []Team

// List teams in which you are a member.
func (s *Service) TeamList(ctx context.Context, lr *ListRange) (TeamListResult, error) {
	var team TeamListResult
	return team, s.Get(ctx, &team, fmt.Sprintf("/teams"), nil, lr)
}

// Info for a team.
func (s *Service) TeamInfo(ctx context.Context, teamIdentity string) (*Team, error) {
	var team Team
	return &team, s.Get(ctx, &team, fmt.Sprintf("/teams/%v", teamIdentity), nil, nil)
}

type TeamUpdateOpts struct {
	Default *bool   `json:"default,omitempty" url:"default,omitempty,key"` // whether to use this team when none is specified
	Name    *string `json:"name,omitempty" url:"name,omitempty,key"`       // unique name of team
}

// Update team properties.
func (s *Service) TeamUpdate(ctx context.Context, teamIdentity string, o TeamUpdateOpts) (*Team, error) {
	var team Team
	return &team, s.Patch(ctx, &team, fmt.Sprintf("/teams/%v", teamIdentity), o)
}

type TeamCreateOpts struct {
	Address1        *string `json:"address_1,omitempty" url:"address_1,omitempty,key"`               // street address line 1
	Address2        *string `json:"address_2,omitempty" url:"address_2,omitempty,key"`               // street address line 2
	CardNumber      *string `json:"card_number,omitempty" url:"card_number,omitempty,key"`           // encrypted card number of payment method
	City            *string `json:"city,omitempty" url:"city,omitempty,key"`                         // city
	Country         *string `json:"country,omitempty" url:"country,omitempty,key"`                   // country
	Cvv             *string `json:"cvv,omitempty" url:"cvv,omitempty,key"`                           // card verification value
	ExpirationMonth *string `json:"expiration_month,omitempty" url:"expiration_month,omitempty,key"` // expiration month
	ExpirationYear  *string `json:"expiration_year,omitempty" url:"expiration_year,omitempty,key"`   // expiration year
	FirstName       *string `json:"first_name,omitempty" url:"first_name,omitempty,key"`             // the first name for payment method
	LastName        *string `json:"last_name,omitempty" url:"last_name,omitempty,key"`               // the last name for payment method
	Name            string  `json:"name" url:"name,key"`                                             // unique name of team
	Other           *string `json:"other,omitempty" url:"other,omitempty,key"`                       // metadata
	PostalCode      *string `json:"postal_code,omitempty" url:"postal_code,omitempty,key"`           // postal code
	State           *string `json:"state,omitempty" url:"state,omitempty,key"`                       // state
}

// Create a new team.
func (s *Service) TeamCreate(ctx context.Context, o TeamCreateOpts) (*Team, error) {
	var team Team
	return &team, s.Post(ctx, &team, fmt.Sprintf("/teams"), o)
}

// Delete an existing team.
func (s *Service) TeamDelete(ctx context.Context, teamIdentity string) (*Team, error) {
	var team Team
	return &team, s.Delete(ctx, &team, fmt.Sprintf("/teams/%v", teamIdentity))
}

// An team app encapsulates the team specific functionality of Heroku
// apps.
type TeamApp struct {
	ArchivedAt                   *time.Time `json:"archived_at" url:"archived_at,key"`                                       // when app was archived
	BuildpackProvidedDescription *string    `json:"buildpack_provided_description" url:"buildpack_provided_description,key"` // description from buildpack of app
	CreatedAt                    time.Time  `json:"created_at" url:"created_at,key"`                                         // when app was created
	GitURL                       string     `json:"git_url" url:"git_url,key"`                                               // git repo URL of app
	ID                           string     `json:"id" url:"id,key"`                                                         // unique identifier of app
	Joined                       bool       `json:"joined" url:"joined,key"`                                                 // is the current member a collaborator on this app.
	Locked                       bool       `json:"locked" url:"locked,key"`                                                 // are other team members forbidden from joining this app.
	Maintenance                  bool       `json:"maintenance" url:"maintenance,key"`                                       // maintenance status of app
	Name                         string     `json:"name" url:"name,key"`                                                     // unique name of app
	Owner                        *struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"owner" url:"owner,key"` // identity of app owner
	Region struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of region
		Name string `json:"name" url:"name,key"` // unique name of region
	} `json:"region" url:"region,key"` // identity of app region
	ReleasedAt *time.Time `json:"released_at" url:"released_at,key"` // when app was released
	RepoSize   *int       `json:"repo_size" url:"repo_size,key"`     // git repo size in bytes of app
	SlugSize   *int       `json:"slug_size" url:"slug_size,key"`     // slug size in bytes of app
	Space      *struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of space
		Name string `json:"name" url:"name,key"` // unique name of space
	} `json:"space" url:"space,key"` // identity of space
	Stack struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of stack
		Name string `json:"name" url:"name,key"` // unique name of stack
	} `json:"stack" url:"stack,key"` // identity of app stack
	Team *struct {
		Name string `json:"name" url:"name,key"` // unique name of team
	} `json:"team" url:"team,key"` // team that owns this app
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when app was updated
	WebURL    string    `json:"web_url" url:"web_url,key"`       // web URL of app
}
type TeamAppCreateOpts struct {
	Locked   *bool   `json:"locked,omitempty" url:"locked,omitempty,key"`     // are other team members forbidden from joining this app.
	Name     *string `json:"name,omitempty" url:"name,omitempty,key"`         // unique name of app
	Personal *bool   `json:"personal,omitempty" url:"personal,omitempty,key"` // force creation of the app in the user account even if a default team
	// is set.
	Region *string `json:"region,omitempty" url:"region,omitempty,key"` // unique name of region
	Space  *string `json:"space,omitempty" url:"space,omitempty,key"`   // unique name of space
	Stack  *string `json:"stack,omitempty" url:"stack,omitempty,key"`   // unique name of stack
	Team   *string `json:"team,omitempty" url:"team,omitempty,key"`     // unique name of team
}

// Create a new app in the specified team, in the default team if
// unspecified, or in personal account, if default team is not set.
func (s *Service) TeamAppCreate(ctx context.Context, o TeamAppCreateOpts) (*TeamApp, error) {
	var teamApp TeamApp
	return &teamApp, s.Post(ctx, &teamApp, fmt.Sprintf("/teams/apps"), o)
}

type TeamAppListResult []TeamApp

// List apps in the default team, or in personal account, if default
// team is not set.
func (s *Service) TeamAppList(ctx context.Context, lr *ListRange) (TeamAppListResult, error) {
	var teamApp TeamAppListResult
	return teamApp, s.Get(ctx, &teamApp, fmt.Sprintf("/teams/apps"), nil, lr)
}

// Info for a team app.
func (s *Service) TeamAppInfo(ctx context.Context, teamAppIdentity string) (*TeamApp, error) {
	var teamApp TeamApp
	return &teamApp, s.Get(ctx, &teamApp, fmt.Sprintf("/teams/apps/%v", teamAppIdentity), nil, nil)
}

type TeamAppUpdateLockedOpts struct {
	Locked bool `json:"locked" url:"locked,key"` // are other team members forbidden from joining this app.
}

// Lock or unlock a team app.
func (s *Service) TeamAppUpdateLocked(ctx context.Context, teamAppIdentity string, o TeamAppUpdateLockedOpts) (*TeamApp, error) {
	var teamApp TeamApp
	return &teamApp, s.Patch(ctx, &teamApp, fmt.Sprintf("/teams/apps/%v", teamAppIdentity), o)
}

type TeamAppTransferToAccountOpts struct {
	Owner string `json:"owner" url:"owner,key"` // unique email address of account
}

// Transfer an existing team app to another Heroku account.
func (s *Service) TeamAppTransferToAccount(ctx context.Context, teamAppIdentity string, o TeamAppTransferToAccountOpts) (*TeamApp, error) {
	var teamApp TeamApp
	return &teamApp, s.Patch(ctx, &teamApp, fmt.Sprintf("/teams/apps/%v", teamAppIdentity), o)
}

type TeamAppTransferToTeamOpts struct {
	Owner string `json:"owner" url:"owner,key"` // unique name of team
}

// Transfer an existing team app to another team.
func (s *Service) TeamAppTransferToTeam(ctx context.Context, teamAppIdentity string, o TeamAppTransferToTeamOpts) (*TeamApp, error) {
	var teamApp TeamApp
	return &teamApp, s.Patch(ctx, &teamApp, fmt.Sprintf("/teams/apps/%v", teamAppIdentity), o)
}

type TeamAppListByTeamResult []TeamApp

// List team apps.
func (s *Service) TeamAppListByTeam(ctx context.Context, teamIdentity string, lr *ListRange) (TeamAppListByTeamResult, error) {
	var teamApp TeamAppListByTeamResult
	return teamApp, s.Get(ctx, &teamApp, fmt.Sprintf("/teams/%v/apps", teamIdentity), nil, lr)
}

// A team collaborator represents an account that has been given access
// to a team app on Heroku.
type TeamAppCollaborator struct {
	App struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of app
		Name string `json:"name" url:"name,key"` // unique name of app
	} `json:"app" url:"app,key"` // app collaborator belongs to
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when collaborator was created
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of collaborator
	Role      *string   `json:"role" url:"role,key"`             // role in the team
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when collaborator was updated
	User      struct {
		Email     string `json:"email" url:"email,key"`         // unique email address of account
		Federated bool   `json:"federated" url:"federated,key"` // whether the user is federated and belongs to an Identity Provider
		ID        string `json:"id" url:"id,key"`               // unique identifier of an account
	} `json:"user" url:"user,key"` // identity of collaborated account
}
type TeamAppCollaboratorCreateOpts struct {
	Permissions *[]*string `json:"permissions,omitempty" url:"permissions,omitempty,key"` // An array of permissions to give to the collaborator.
	Silent      *bool      `json:"silent,omitempty" url:"silent,omitempty,key"`           // whether to suppress email invitation when creating collaborator
	User        string     `json:"user" url:"user,key"`                                   // unique email address of account
}

// Create a new collaborator on a team app. Use this endpoint instead of
// the `/apps/{app_id_or_name}/collaborator` endpoint when you want the
// collaborator to be granted [permissions]
// (https://devcenter.heroku.com/articles/org-users-access#roles-and-app-
// permissions) according to their role in the team.
func (s *Service) TeamAppCollaboratorCreate(ctx context.Context, appIdentity string, o TeamAppCollaboratorCreateOpts) (*TeamAppCollaborator, error) {
	var teamAppCollaborator TeamAppCollaborator
	return &teamAppCollaborator, s.Post(ctx, &teamAppCollaborator, fmt.Sprintf("/teams/apps/%v/collaborators", appIdentity), o)
}

// Delete an existing collaborator from a team app.
func (s *Service) TeamAppCollaboratorDelete(ctx context.Context, teamAppIdentity string, teamAppCollaboratorIdentity string) (*TeamAppCollaborator, error) {
	var teamAppCollaborator TeamAppCollaborator
	return &teamAppCollaborator, s.Delete(ctx, &teamAppCollaborator, fmt.Sprintf("/teams/apps/%v/collaborators/%v", teamAppIdentity, teamAppCollaboratorIdentity))
}

// Info for a collaborator on a team app.
func (s *Service) TeamAppCollaboratorInfo(ctx context.Context, teamAppIdentity string, teamAppCollaboratorIdentity string) (*TeamAppCollaborator, error) {
	var teamAppCollaborator TeamAppCollaborator
	return &teamAppCollaborator, s.Get(ctx, &teamAppCollaborator, fmt.Sprintf("/teams/apps/%v/collaborators/%v", teamAppIdentity, teamAppCollaboratorIdentity), nil, nil)
}

type TeamAppCollaboratorUpdateOpts struct {
	Permissions []string `json:"permissions" url:"permissions,key"` // An array of permissions to give to the collaborator.
}

// Update an existing collaborator from a team app.
func (s *Service) TeamAppCollaboratorUpdate(ctx context.Context, teamAppIdentity string, teamAppCollaboratorIdentity string, o TeamAppCollaboratorUpdateOpts) (*TeamAppCollaborator, error) {
	var teamAppCollaborator TeamAppCollaborator
	return &teamAppCollaborator, s.Patch(ctx, &teamAppCollaborator, fmt.Sprintf("/teams/apps/%v/collaborators/%v", teamAppIdentity, teamAppCollaboratorIdentity), o)
}

type TeamAppCollaboratorListResult []TeamAppCollaborator

// List collaborators on a team app.
func (s *Service) TeamAppCollaboratorList(ctx context.Context, teamAppIdentity string, lr *ListRange) (TeamAppCollaboratorListResult, error) {
	var teamAppCollaborator TeamAppCollaboratorListResult
	return teamAppCollaborator, s.Get(ctx, &teamAppCollaborator, fmt.Sprintf("/teams/apps/%v/collaborators", teamAppIdentity), nil, lr)
}

// A team app permission is a behavior that is assigned to a user in a
// team app.
type TeamAppPermission struct {
	Description string `json:"description" url:"description,key"` // A description of what the app permission allows.
	Name        string `json:"name" url:"name,key"`               // The name of the app permission.
}
type TeamAppPermissionListResult []TeamAppPermission

// Lists permissions available to teams.
func (s *Service) TeamAppPermissionList(ctx context.Context, lr *ListRange) (TeamAppPermissionListResult, error) {
	var teamAppPermission TeamAppPermissionListResult
	return teamAppPermission, s.Get(ctx, &teamAppPermission, fmt.Sprintf("/teams/permissions"), nil, lr)
}

// A team feature represents a feature enabled on a team account.
type TeamFeature struct {
	CreatedAt     time.Time `json:"created_at" url:"created_at,key"`         // when team feature was created
	Description   string    `json:"description" url:"description,key"`       // description of team feature
	DisplayName   string    `json:"display_name" url:"display_name,key"`     // user readable feature name
	DocURL        string    `json:"doc_url" url:"doc_url,key"`               // documentation URL of team feature
	Enabled       bool      `json:"enabled" url:"enabled,key"`               // whether or not team feature has been enabled
	FeedbackEmail string    `json:"feedback_email" url:"feedback_email,key"` // e-mail to send feedback about the feature
	ID            string    `json:"id" url:"id,key"`                         // unique identifier of team feature
	Name          string    `json:"name" url:"name,key"`                     // unique name of team feature
	State         string    `json:"state" url:"state,key"`                   // state of team feature
	UpdatedAt     time.Time `json:"updated_at" url:"updated_at,key"`         // when team feature was updated
}

// Info for an existing team feature.
func (s *Service) TeamFeatureInfo(ctx context.Context, teamIdentity string, teamFeatureIdentity string) (*TeamFeature, error) {
	var teamFeature TeamFeature
	return &teamFeature, s.Get(ctx, &teamFeature, fmt.Sprintf("/teams/%v/features/%v", teamIdentity, teamFeatureIdentity), nil, nil)
}

type TeamFeatureListResult []TeamFeature

// List existing team features.
func (s *Service) TeamFeatureList(ctx context.Context, teamIdentity string, lr *ListRange) (TeamFeatureListResult, error) {
	var teamFeature TeamFeatureListResult
	return teamFeature, s.Get(ctx, &teamFeature, fmt.Sprintf("/teams/%v/features", teamIdentity), nil, lr)
}

// A team invitation represents an invite to a team.
type TeamInvitation struct {
	CreatedAt time.Time `json:"created_at" url:"created_at,key"` // when invitation was created
	ID        string    `json:"id" url:"id,key"`                 // unique identifier of an invitation
	InvitedBy struct {
		Email string  `json:"email" url:"email,key"` // unique email address of account
		ID    string  `json:"id" url:"id,key"`       // unique identifier of an account
		Name  *string `json:"name" url:"name,key"`   // full name of the account owner
	} `json:"invited_by" url:"invited_by,key"`
	Role *string `json:"role" url:"role,key"` // role in the team
	Team struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of team
		Name string `json:"name" url:"name,key"` // unique name of team
	} `json:"team" url:"team,key"`
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when invitation was updated
	User      struct {
		Email string  `json:"email" url:"email,key"` // unique email address of account
		ID    string  `json:"id" url:"id,key"`       // unique identifier of an account
		Name  *string `json:"name" url:"name,key"`   // full name of the account owner
	} `json:"user" url:"user,key"`
}
type TeamInvitationListResult []TeamInvitation

// Get a list of a team's Identity Providers
func (s *Service) TeamInvitationList(ctx context.Context, teamName string, lr *ListRange) (TeamInvitationListResult, error) {
	var teamInvitation TeamInvitationListResult
	return teamInvitation, s.Get(ctx, &teamInvitation, fmt.Sprintf("/teams/%v/invitations", teamName), nil, lr)
}

type TeamInvitationCreateOpts struct {
	Email string  `json:"email" url:"email,key"` // unique email address of account
	Role  *string `json:"role" url:"role,key"`   // role in the team
}

// Create Team Invitation
func (s *Service) TeamInvitationCreate(ctx context.Context, teamIdentity string, o TeamInvitationCreateOpts) (*TeamInvitation, error) {
	var teamInvitation TeamInvitation
	return &teamInvitation, s.Put(ctx, &teamInvitation, fmt.Sprintf("/teams/%v/invitations", teamIdentity), o)
}

// Revoke a team invitation.
func (s *Service) TeamInvitationRevoke(ctx context.Context, teamIdentity string, teamInvitationIdentity string) (*TeamInvitation, error) {
	var teamInvitation TeamInvitation
	return &teamInvitation, s.Delete(ctx, &teamInvitation, fmt.Sprintf("/teams/%v/invitations/%v", teamIdentity, teamInvitationIdentity))
}

// Get an invitation by its token
func (s *Service) TeamInvitationGet(ctx context.Context, teamInvitationToken string, lr *ListRange) (*TeamInvitation, error) {
	var teamInvitation TeamInvitation
	return &teamInvitation, s.Get(ctx, &teamInvitation, fmt.Sprintf("/teams/invitations/%v", teamInvitationToken), nil, lr)
}

type TeamInvitationAcceptResult struct {
	CreatedAt               time.Time `json:"created_at" url:"created_at,key"`                               // when the membership record was created
	Email                   string    `json:"email" url:"email,key"`                                         // email address of the team member
	Federated               bool      `json:"federated" url:"federated,key"`                                 // whether the user is federated and belongs to an Identity Provider
	ID                      string    `json:"id" url:"id,key"`                                               // unique identifier of the team member
	Role                    *string   `json:"role" url:"role,key"`                                           // role in the team
	TwoFactorAuthentication bool      `json:"two_factor_authentication" url:"two_factor_authentication,key"` // whether the Enterprise team member has two factor authentication
	// enabled
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when the membership record was updated
	User      struct {
		Email string  `json:"email" url:"email,key"` // unique email address of account
		ID    string  `json:"id" url:"id,key"`       // unique identifier of an account
		Name  *string `json:"name" url:"name,key"`   // full name of the account owner
	} `json:"user" url:"user,key"` // user information for the membership
}

// Accept Team Invitation
func (s *Service) TeamInvitationAccept(ctx context.Context, teamInvitationToken string) (*TeamInvitationAcceptResult, error) {
	var teamInvitation TeamInvitationAcceptResult
	return &teamInvitation, s.Post(ctx, &teamInvitation, fmt.Sprintf("/teams/invitations/%v/accept", teamInvitationToken), nil)
}

// A Team Invoice is an itemized bill of goods for a team which includes
// pricing and charges.
type TeamInvoice struct {
	AddonsTotal       int       `json:"addons_total" url:"addons_total,key"`               // total add-ons charges in on this invoice
	ChargesTotal      int       `json:"charges_total" url:"charges_total,key"`             // total charges on this invoice
	CreatedAt         time.Time `json:"created_at" url:"created_at,key"`                   // when invoice was created
	CreditsTotal      int       `json:"credits_total" url:"credits_total,key"`             // total credits on this invoice
	DatabaseTotal     int       `json:"database_total" url:"database_total,key"`           // total database charges on this invoice
	DynoUnits         float64   `json:"dyno_units" url:"dyno_units,key"`                   // total amount of dyno units consumed across dyno types.
	ID                string    `json:"id" url:"id,key"`                                   // unique identifier of this invoice
	Number            int       `json:"number" url:"number,key"`                           // human readable invoice number
	PaymentStatus     string    `json:"payment_status" url:"payment_status,key"`           // status of the invoice payment
	PeriodEnd         string    `json:"period_end" url:"period_end,key"`                   // the ending date that the invoice covers
	PeriodStart       string    `json:"period_start" url:"period_start,key"`               // the starting date that this invoice covers
	PlatformTotal     int       `json:"platform_total" url:"platform_total,key"`           // total platform charges on this invoice
	State             int       `json:"state" url:"state,key"`                             // payment status for this invoice (pending, successful, failed)
	Total             int       `json:"total" url:"total,key"`                             // combined total of charges and credits on this invoice
	UpdatedAt         time.Time `json:"updated_at" url:"updated_at,key"`                   // when invoice was updated
	WeightedDynoHours float64   `json:"weighted_dyno_hours" url:"weighted_dyno_hours,key"` // The total amount of hours consumed across dyno types.
}

// Info for existing invoice.
func (s *Service) TeamInvoiceInfo(ctx context.Context, teamIdentity string, teamInvoiceIdentity int) (*TeamInvoice, error) {
	var teamInvoice TeamInvoice
	return &teamInvoice, s.Get(ctx, &teamInvoice, fmt.Sprintf("/teams/%v/invoices/%v", teamIdentity, teamInvoiceIdentity), nil, nil)
}

type TeamInvoiceListResult []TeamInvoice

// List existing invoices.
func (s *Service) TeamInvoiceList(ctx context.Context, teamIdentity string, lr *ListRange) (TeamInvoiceListResult, error) {
	var teamInvoice TeamInvoiceListResult
	return teamInvoice, s.Get(ctx, &teamInvoice, fmt.Sprintf("/teams/%v/invoices", teamIdentity), nil, lr)
}

// A team member is an individual with access to a team.
type TeamMember struct {
	CreatedAt               time.Time `json:"created_at" url:"created_at,key"`                               // when the membership record was created
	Email                   string    `json:"email" url:"email,key"`                                         // email address of the team member
	Federated               bool      `json:"federated" url:"federated,key"`                                 // whether the user is federated and belongs to an Identity Provider
	ID                      string    `json:"id" url:"id,key"`                                               // unique identifier of the team member
	Role                    *string   `json:"role" url:"role,key"`                                           // role in the team
	TwoFactorAuthentication bool      `json:"two_factor_authentication" url:"two_factor_authentication,key"` // whether the Enterprise team member has two factor authentication
	// enabled
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when the membership record was updated
	User      struct {
		Email string  `json:"email" url:"email,key"` // unique email address of account
		ID    string  `json:"id" url:"id,key"`       // unique identifier of an account
		Name  *string `json:"name" url:"name,key"`   // full name of the account owner
	} `json:"user" url:"user,key"` // user information for the membership
}
type TeamMemberCreateOrUpdateOpts struct {
	Email     string  `json:"email" url:"email,key"`                             // email address of the team member
	Federated *bool   `json:"federated,omitempty" url:"federated,omitempty,key"` // whether the user is federated and belongs to an Identity Provider
	Role      *string `json:"role" url:"role,key"`                               // role in the team
}

// Create a new team member, or update their role.
func (s *Service) TeamMemberCreateOrUpdate(ctx context.Context, teamIdentity string, o TeamMemberCreateOrUpdateOpts) (*TeamMember, error) {
	var teamMember TeamMember
	return &teamMember, s.Put(ctx, &teamMember, fmt.Sprintf("/teams/%v/members", teamIdentity), o)
}

type TeamMemberCreateOpts struct {
	Email     string  `json:"email" url:"email,key"`                             // email address of the team member
	Federated *bool   `json:"federated,omitempty" url:"federated,omitempty,key"` // whether the user is federated and belongs to an Identity Provider
	Role      *string `json:"role" url:"role,key"`                               // role in the team
}

// Create a new team member.
func (s *Service) TeamMemberCreate(ctx context.Context, teamIdentity string, o TeamMemberCreateOpts) (*TeamMember, error) {
	var teamMember TeamMember
	return &teamMember, s.Post(ctx, &teamMember, fmt.Sprintf("/teams/%v/members", teamIdentity), o)
}

type TeamMemberUpdateOpts struct {
	Email     string  `json:"email" url:"email,key"`                             // email address of the team member
	Federated *bool   `json:"federated,omitempty" url:"federated,omitempty,key"` // whether the user is federated and belongs to an Identity Provider
	Role      *string `json:"role" url:"role,key"`                               // role in the team
}

// Update a team member.
func (s *Service) TeamMemberUpdate(ctx context.Context, teamIdentity string, o TeamMemberUpdateOpts) (*TeamMember, error) {
	var teamMember TeamMember
	return &teamMember, s.Patch(ctx, &teamMember, fmt.Sprintf("/teams/%v/members", teamIdentity), o)
}

// Remove a member from the team.
func (s *Service) TeamMemberDelete(ctx context.Context, teamIdentity string, teamMemberIdentity string) (*TeamMember, error) {
	var teamMember TeamMember
	return &teamMember, s.Delete(ctx, &teamMember, fmt.Sprintf("/teams/%v/members/%v", teamIdentity, teamMemberIdentity))
}

type TeamMemberListResult []TeamMember

// List members of the team.
func (s *Service) TeamMemberList(ctx context.Context, teamIdentity string, lr *ListRange) (TeamMemberListResult, error) {
	var teamMember TeamMemberListResult
	return teamMember, s.Get(ctx, &teamMember, fmt.Sprintf("/teams/%v/members", teamIdentity), nil, lr)
}

type TeamMemberListByMemberResult []struct {
	ArchivedAt                   *time.Time `json:"archived_at" url:"archived_at,key"`                                       // when app was archived
	BuildpackProvidedDescription *string    `json:"buildpack_provided_description" url:"buildpack_provided_description,key"` // description from buildpack of app
	CreatedAt                    time.Time  `json:"created_at" url:"created_at,key"`                                         // when app was created
	GitURL                       string     `json:"git_url" url:"git_url,key"`                                               // git repo URL of app
	ID                           string     `json:"id" url:"id,key"`                                                         // unique identifier of app
	Joined                       bool       `json:"joined" url:"joined,key"`                                                 // is the current member a collaborator on this app.
	Locked                       bool       `json:"locked" url:"locked,key"`                                                 // are other team members forbidden from joining this app.
	Maintenance                  bool       `json:"maintenance" url:"maintenance,key"`                                       // maintenance status of app
	Name                         string     `json:"name" url:"name,key"`                                                     // unique name of app
	Owner                        *struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"owner" url:"owner,key"` // identity of app owner
	Region struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of region
		Name string `json:"name" url:"name,key"` // unique name of region
	} `json:"region" url:"region,key"` // identity of app region
	ReleasedAt *time.Time `json:"released_at" url:"released_at,key"` // when app was released
	RepoSize   *int       `json:"repo_size" url:"repo_size,key"`     // git repo size in bytes of app
	SlugSize   *int       `json:"slug_size" url:"slug_size,key"`     // slug size in bytes of app
	Space      *struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of space
		Name string `json:"name" url:"name,key"` // unique name of space
	} `json:"space" url:"space,key"` // identity of space
	Stack struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of stack
		Name string `json:"name" url:"name,key"` // unique name of stack
	} `json:"stack" url:"stack,key"` // identity of app stack
	Team *struct {
		Name string `json:"name" url:"name,key"` // unique name of team
	} `json:"team" url:"team,key"` // team that owns this app
	UpdatedAt time.Time `json:"updated_at" url:"updated_at,key"` // when app was updated
	WebURL    string    `json:"web_url" url:"web_url,key"`       // web URL of app
}

// List the apps of a team member.
func (s *Service) TeamMemberListByMember(ctx context.Context, teamIdentity string, teamMemberIdentity string, lr *ListRange) (TeamMemberListByMemberResult, error) {
	var teamMember TeamMemberListByMemberResult
	return teamMember, s.Get(ctx, &teamMember, fmt.Sprintf("/teams/%v/members/%v/apps", teamIdentity, teamMemberIdentity), nil, lr)
}

// Tracks a Team's Preferences
type TeamPreferences struct {
	DefaultPermission   *string `json:"default-permission" url:"default-permission,key"`     // The default permission used when adding new members to the team
	WhitelistingEnabled *bool   `json:"whitelisting-enabled" url:"whitelisting-enabled,key"` // Whether whitelisting rules should be applied to add-on installations
}

// Retrieve Team Preferences
func (s *Service) TeamPreferencesList(ctx context.Context, teamPreferencesIdentity string) (*TeamPreferences, error) {
	var teamPreferences TeamPreferences
	return &teamPreferences, s.Get(ctx, &teamPreferences, fmt.Sprintf("/teams/%v/preferences", teamPreferencesIdentity), nil, nil)
}

type TeamPreferencesUpdateOpts struct {
	WhitelistingEnabled *bool `json:"whitelisting-enabled,omitempty" url:"whitelisting-enabled,omitempty,key"` // Whether whitelisting rules should be applied to add-on installations
}

// Update Team Preferences
func (s *Service) TeamPreferencesUpdate(ctx context.Context, teamPreferencesIdentity string, o TeamPreferencesUpdateOpts) (*TeamPreferences, error) {
	var teamPreferences TeamPreferences
	return &teamPreferences, s.Patch(ctx, &teamPreferences, fmt.Sprintf("/teams/%v/preferences", teamPreferencesIdentity), o)
}

// Tracks a user's preferences and message dismissals
type UserPreferences struct {
	DefaultOrganization        *string `json:"default-organization" url:"default-organization,key"`                   // User's default organization
	DismissedGettingStarted    *bool   `json:"dismissed-getting-started" url:"dismissed-getting-started,key"`         // Whether the user has dismissed the getting started banner
	DismissedGithubBanner      *bool   `json:"dismissed-github-banner" url:"dismissed-github-banner,key"`             // Whether the user has dismissed the GitHub link banner
	DismissedOrgAccessControls *bool   `json:"dismissed-org-access-controls" url:"dismissed-org-access-controls,key"` // Whether the user has dismissed the Organization Access Controls
	// banner
	DismissedOrgWizardNotification *bool `json:"dismissed-org-wizard-notification" url:"dismissed-org-wizard-notification,key"` // Whether the user has dismissed the Organization Wizard
	DismissedPipelinesBanner       *bool `json:"dismissed-pipelines-banner" url:"dismissed-pipelines-banner,key"`               // Whether the user has dismissed the Pipelines banner
	DismissedPipelinesGithubBanner *bool `json:"dismissed-pipelines-github-banner" url:"dismissed-pipelines-github-banner,key"` // Whether the user has dismissed the GitHub banner on a pipeline
	// overview
	DismissedPipelinesGithubBanners *[]string `json:"dismissed-pipelines-github-banners" url:"dismissed-pipelines-github-banners,key"` // Which pipeline uuids the user has dismissed the GitHub banner for
	DismissedSmsBanner              *bool     `json:"dismissed-sms-banner" url:"dismissed-sms-banner,key"`                             // Whether the user has dismissed the 2FA SMS banner
	Timezone                        *string   `json:"timezone" url:"timezone,key"`                                                     // User's default timezone
}

// Retrieve User Preferences
func (s *Service) UserPreferencesList(ctx context.Context, userPreferencesIdentity string) (*UserPreferences, error) {
	var userPreferences UserPreferences
	return &userPreferences, s.Get(ctx, &userPreferences, fmt.Sprintf("/users/%v/preferences", userPreferencesIdentity), nil, nil)
}

type UserPreferencesUpdateOpts struct {
	DefaultOrganization        *string `json:"default-organization,omitempty" url:"default-organization,omitempty,key"`                   // User's default organization
	DismissedGettingStarted    *bool   `json:"dismissed-getting-started,omitempty" url:"dismissed-getting-started,omitempty,key"`         // Whether the user has dismissed the getting started banner
	DismissedGithubBanner      *bool   `json:"dismissed-github-banner,omitempty" url:"dismissed-github-banner,omitempty,key"`             // Whether the user has dismissed the GitHub link banner
	DismissedOrgAccessControls *bool   `json:"dismissed-org-access-controls,omitempty" url:"dismissed-org-access-controls,omitempty,key"` // Whether the user has dismissed the Organization Access Controls
	// banner
	DismissedOrgWizardNotification *bool `json:"dismissed-org-wizard-notification,omitempty" url:"dismissed-org-wizard-notification,omitempty,key"` // Whether the user has dismissed the Organization Wizard
	DismissedPipelinesBanner       *bool `json:"dismissed-pipelines-banner,omitempty" url:"dismissed-pipelines-banner,omitempty,key"`               // Whether the user has dismissed the Pipelines banner
	DismissedPipelinesGithubBanner *bool `json:"dismissed-pipelines-github-banner,omitempty" url:"dismissed-pipelines-github-banner,omitempty,key"` // Whether the user has dismissed the GitHub banner on a pipeline
	// overview
	DismissedPipelinesGithubBanners *[]*string `json:"dismissed-pipelines-github-banners,omitempty" url:"dismissed-pipelines-github-banners,omitempty,key"` // Which pipeline uuids the user has dismissed the GitHub banner for
	DismissedSmsBanner              *bool      `json:"dismissed-sms-banner,omitempty" url:"dismissed-sms-banner,omitempty,key"`                             // Whether the user has dismissed the 2FA SMS banner
	Timezone                        *string    `json:"timezone,omitempty" url:"timezone,omitempty,key"`                                                     // User's default timezone
}

// Update User Preferences
func (s *Service) UserPreferencesUpdate(ctx context.Context, userPreferencesIdentity string, o UserPreferencesUpdateOpts) (*UserPreferences, error) {
	var userPreferences UserPreferences
	return &userPreferences, s.Patch(ctx, &userPreferences, fmt.Sprintf("/users/%v/preferences", userPreferencesIdentity), o)
}

// Entities that have been whitelisted to be used by an Organization
type WhitelistedAddOnService struct {
	AddedAt time.Time `json:"added_at" url:"added_at,key"` // when the add-on service was whitelisted
	AddedBy struct {
		Email string `json:"email" url:"email,key"` // unique email address of account
		ID    string `json:"id" url:"id,key"`       // unique identifier of an account
	} `json:"added_by" url:"added_by,key"` // the user which whitelisted the Add-on Service
	AddonService struct {
		HumanName string `json:"human_name" url:"human_name,key"` // human-readable name of the add-on service provider
		ID        string `json:"id" url:"id,key"`                 // unique identifier of this add-on-service
		Name      string `json:"name" url:"name,key"`             // unique name of this add-on-service
	} `json:"addon_service" url:"addon_service,key"` // the Add-on Service whitelisted for use
	ID string `json:"id" url:"id,key"` // unique identifier for this whitelisting entity
}
type WhitelistedAddOnServiceListByOrganizationResult []WhitelistedAddOnService

// List all whitelisted Add-on Services for an Organization
func (s *Service) WhitelistedAddOnServiceListByOrganization(ctx context.Context, organizationIdentity string, lr *ListRange) (WhitelistedAddOnServiceListByOrganizationResult, error) {
	var whitelistedAddOnService WhitelistedAddOnServiceListByOrganizationResult
	return whitelistedAddOnService, s.Get(ctx, &whitelistedAddOnService, fmt.Sprintf("/organizations/%v/whitelisted-addon-services", organizationIdentity), nil, lr)
}

type WhitelistedAddOnServiceCreateByOrganizationOpts struct {
	AddonService *string `json:"addon_service,omitempty" url:"addon_service,omitempty,key"` // name of the Add-on to whitelist
}
type WhitelistedAddOnServiceCreateByOrganizationResult []WhitelistedAddOnService

// Whitelist an Add-on Service
func (s *Service) WhitelistedAddOnServiceCreateByOrganization(ctx context.Context, organizationIdentity string, o WhitelistedAddOnServiceCreateByOrganizationOpts) (WhitelistedAddOnServiceCreateByOrganizationResult, error) {
	var whitelistedAddOnService WhitelistedAddOnServiceCreateByOrganizationResult
	return whitelistedAddOnService, s.Post(ctx, &whitelistedAddOnService, fmt.Sprintf("/organizations/%v/whitelisted-addon-services", organizationIdentity), o)
}

// Remove a whitelisted entity
func (s *Service) WhitelistedAddOnServiceDeleteByOrganization(ctx context.Context, organizationIdentity string, whitelistedAddOnServiceIdentity string) (*WhitelistedAddOnService, error) {
	var whitelistedAddOnService WhitelistedAddOnService
	return &whitelistedAddOnService, s.Delete(ctx, &whitelistedAddOnService, fmt.Sprintf("/organizations/%v/whitelisted-addon-services/%v", organizationIdentity, whitelistedAddOnServiceIdentity))
}

type WhitelistedAddOnServiceListByTeamResult []WhitelistedAddOnService

// List all whitelisted Add-on Services for a Team
func (s *Service) WhitelistedAddOnServiceListByTeam(ctx context.Context, teamIdentity string, lr *ListRange) (WhitelistedAddOnServiceListByTeamResult, error) {
	var whitelistedAddOnService WhitelistedAddOnServiceListByTeamResult
	return whitelistedAddOnService, s.Get(ctx, &whitelistedAddOnService, fmt.Sprintf("/teams/%v/whitelisted-addon-services", teamIdentity), nil, lr)
}

type WhitelistedAddOnServiceCreateByTeamOpts struct {
	AddonService *string `json:"addon_service,omitempty" url:"addon_service,omitempty,key"` // name of the Add-on to whitelist
}
type WhitelistedAddOnServiceCreateByTeamResult []WhitelistedAddOnService

// Whitelist an Add-on Service
func (s *Service) WhitelistedAddOnServiceCreateByTeam(ctx context.Context, teamIdentity string, o WhitelistedAddOnServiceCreateByTeamOpts) (WhitelistedAddOnServiceCreateByTeamResult, error) {
	var whitelistedAddOnService WhitelistedAddOnServiceCreateByTeamResult
	return whitelistedAddOnService, s.Post(ctx, &whitelistedAddOnService, fmt.Sprintf("/teams/%v/whitelisted-addon-services", teamIdentity), o)
}

// Remove a whitelisted entity
func (s *Service) WhitelistedAddOnServiceDeleteByTeam(ctx context.Context, teamIdentity string, whitelistedAddOnServiceIdentity string) (*WhitelistedAddOnService, error) {
	var whitelistedAddOnService WhitelistedAddOnService
	return &whitelistedAddOnService, s.Delete(ctx, &whitelistedAddOnService, fmt.Sprintf("/teams/%v/whitelisted-addon-services/%v", teamIdentity, whitelistedAddOnServiceIdentity))
}

