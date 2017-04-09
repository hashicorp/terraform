package cfapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/cli/cf/api"
	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

// UserManager -
type UserManager struct {
	log *Logger

	config     coreconfig.Reader
	uaaGateway net.Gateway
	ccGateway  net.Gateway

	clientToken string

	groupMap      map[string]string
	defaultGroups map[string]byte

	repo api.UserRepository
}

// UAAUser -
type UAAUser struct {
	ID       string `json:"id,omitempty"`
	Username string `json:"userName,omitempty"`
	Password string `json:"password,omitempty"`
	Origin   string `json:"origin,omitempty"`

	Name   UAAUserName    `json:"name,omitempty"`
	Emails []UAAUserEmail `json:"emails,omitempty"`
	Groups []UAAUserGroup `json:"groups,omitempty"`
}

// UAAUserEmail -
type UAAUserEmail struct {
	Value string `json:"value"`
}

// UAAUserName -
type UAAUserName struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
}

// UAAUserGroup -
type UAAUserGroup struct {
	Value   string `json:"value"`
	Display string `json:"display"`
	Type    string `json:"type"`
}

// UAAGroupResourceList -
type UAAGroupResourceList struct {
	Resources []struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"resources"`
}

// CCUser -
type CCUser struct {
	ID string

	UserName         string `json:"username"`
	IsAdmin          bool   `json:"admin,omitempty"`
	IsActive         bool   `json:"active,omitempty"`
	DefaultSpaceGUID bool   `json:"default_space_guid,omitempty"`
}

// CCUserResource -
type CCUserResource struct {
	Metadata resources.Metadata `json:"metadata"`
	Entity   CCUser             `json:"entity"`
}

// CCUserList -
type CCUserList struct {
	Resources []CCUserResource `json:"resources"`
}

// UserRoleInOrg -
type UserRoleInOrg string

// UserIsOrgManager -
const UserIsOrgManager = UserRoleInOrg("managed_organizations")

// UserIsOrgBillingManager -
const UserIsOrgBillingManager = UserRoleInOrg("billing_managed_organizations")

// UserIsOrgAuditor -
const UserIsOrgAuditor = UserRoleInOrg("audited_organizations")

// UserIsOrgMember -
const UserIsOrgMember = UserRoleInOrg("organizations")

// NewUserManager -
func newUserManager(config coreconfig.Reader, uaaGateway net.Gateway, ccGateway net.Gateway, logger *Logger) (um *UserManager, err error) {

	um = &UserManager{
		log: logger,

		config:        config,
		uaaGateway:    uaaGateway,
		ccGateway:     ccGateway,
		groupMap:      make(map[string]string),
		defaultGroups: make(map[string]byte),

		repo: api.NewCloudControllerUserRepository(config, uaaGateway, ccGateway),
	}
	err = um.loadGroups()
	return
}

func (um *UserManager) loadGroups() (err error) {

	uaaEndpoint := um.config.UaaEndpoint()
	if len(uaaEndpoint) == 0 {
		err = errors.New("UAA endpoint missing from config file")
		return
	}

	// Retrieve alls groups
	groupList := &UAAGroupResourceList{}
	err = um.uaaGateway.GetResource(
		fmt.Sprintf("%s/Groups", uaaEndpoint),
		groupList)
	if err != nil {
		return
	}
	for _, r := range groupList.Resources {
		um.groupMap[r.DisplayName] = r.ID
	}

	// Retrieve default scope/groups for a new user by creating
	// a dummy user and extracting the default scope of that user
	username, err := newUUID()
	if err != nil {
		return
	}
	userResource := UAAUser{
		Username: username,
		Password: "password",
		Origin:   "uaa",
		Emails:   []UAAUserEmail{{Value: "email@domain.com"}},
	}
	body, err := json.Marshal(userResource)
	if err != nil {
		return
	}
	user := &UAAUser{}
	err = um.uaaGateway.CreateResource(uaaEndpoint, "/Users", bytes.NewReader(body), user)
	if err != nil {
		return err
	}
	err = um.uaaGateway.DeleteResource(uaaEndpoint, fmt.Sprintf("/Users/%s", user.ID))
	if err != nil {
		return err
	}
	for _, g := range user.Groups {
		um.defaultGroups[g.Display] = 1
	}

	return
}

// IsDefaultGroup -
func (um *UserManager) IsDefaultGroup(group string) (ok bool) {
	_, ok = um.defaultGroups[group]
	return
}

// GetUser -
func (um *UserManager) GetUser(id string) (user *UAAUser, err error) {

	uaaEndpoint := um.config.UaaEndpoint()
	if len(uaaEndpoint) == 0 {
		err = errors.New("UAA endpoint missing from config file")
		return
	}

	user = &UAAUser{}
	err = um.uaaGateway.GetResource(
		fmt.Sprintf("%s/Users/%s", uaaEndpoint, id),
		user)

	return
}

// CreateUser -
func (um *UserManager) CreateUser(
	username, password, origin, givenName, familyName, email string) (user *UAAUser, err error) {

	uaaEndpoint := um.config.UaaEndpoint()
	if len(uaaEndpoint) == 0 {
		err = errors.New("UAA endpoint missing from config file")
		return
	}

	userResource := UAAUser{
		Username: username,
		Password: password,
		Origin:   origin,
		Name: UAAUserName{
			GivenName:  givenName,
			FamilyName: familyName,
		},
	}
	if len(email) > 0 {
		userResource.Emails = append(userResource.Emails, UAAUserEmail{email})
	} else {
		userResource.Emails = append(userResource.Emails, UAAUserEmail{username})
	}

	body, err := json.Marshal(userResource)
	if err != nil {
		return
	}

	user = &UAAUser{}
	err = um.uaaGateway.CreateResource(uaaEndpoint, "/Users", bytes.NewReader(body), user)
	switch httpErr := err.(type) {
	case nil:
	case errors.HTTPError:
		if httpErr.StatusCode() == http.StatusConflict {
			err = errors.NewModelAlreadyExistsError("user", username)
			return
		}
		return
	default:
		return
	}

	body, err = json.Marshal(resources.Metadata{
		GUID: user.ID,
	})
	if err == nil {
		err = um.ccGateway.CreateResource(um.config.APIEndpoint(), "/v2/users", bytes.NewReader(body))
	}
	return
}

// UpdateUser -
func (um *UserManager) UpdateUser(
	id, username, givenName, familyName, email string) (user *UAAUser, err error) {

	uaaEndpoint := um.config.UaaEndpoint()
	if len(uaaEndpoint) == 0 {
		err = errors.New("UAA endpoint missing from config file")
		return
	}

	userResource := UAAUser{
		Username: username,
		Name: UAAUserName{
			GivenName:  givenName,
			FamilyName: familyName,
		},
	}
	if len(email) > 0 {
		userResource.Emails = append(userResource.Emails, UAAUserEmail{email})
	} else {
		userResource.Emails = append(userResource.Emails, UAAUserEmail{username})
	}

	body, err := json.Marshal(userResource)
	if err != nil {
		return
	}

	request, err := um.uaaGateway.NewRequest("PUT",
		fmt.Sprintf("%s/Users/%s", uaaEndpoint, id),
		um.config.AccessToken(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.HTTPReq.Header.Set("If-Match", "*")

	user = &UAAUser{}
	_, err = um.uaaGateway.PerformRequestForJSONResponse(request, user)

	return
}

// ChangePassword -
func (um *UserManager) ChangePassword(
	id, oldPassword, newPassword string) (err error) {

	uaaEndpoint := um.config.UaaEndpoint()
	if len(uaaEndpoint) == 0 {
		err = errors.New("UAA endpoint missing from config file")
		return
	}

	body, err := json.Marshal(map[string]string{
		"oldPassword": oldPassword,
		"password":    newPassword,
	})
	if err != nil {
		return
	}

	request, err := um.uaaGateway.NewRequest("PUT",
		uaaEndpoint+fmt.Sprintf("/Users/%s/password", id),
		um.config.AccessToken(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.HTTPReq.Header.Set("Authorization", um.clientToken)

	response := make(map[string]interface{})
	_, err = um.uaaGateway.PerformRequestForJSONResponse(request, response)
	if err != nil {
		return err
	}
	return
}

// UpdateRoles -
func (um *UserManager) UpdateRoles(
	id string, scopesToDelete, scopesToAdd []string, origin string) (err error) {

	uaaEndpoint := um.config.UaaEndpoint()
	if len(uaaEndpoint) == 0 {
		err = errors.New("UAA endpoint missing from config file")
		return
	}

	for _, s := range scopesToDelete {
		roleID := um.groupMap[s]
		err = um.uaaGateway.DeleteResource(uaaEndpoint,
			fmt.Sprintf("/Groups/%s/members/%s", roleID, id))
	}
	for _, s := range scopesToAdd {
		roleID, exists := um.groupMap[s]
		if !exists {
			err = fmt.Errorf("Group '%s' was not found", s)
			return
		}

		var body []byte
		body, err = json.Marshal(map[string]string{
			"origin": origin,
			"type":   "USER",
			"value":  id,
		})
		if err != nil {
			return
		}

		response := make(map[string]interface{})
		err = um.uaaGateway.CreateResource(uaaEndpoint,
			fmt.Sprintf("/Groups/%s/members", roleID),
			bytes.NewReader(body), &response)
		if err != nil {
			return
		}
	}

	return
}

// RemoveUserFromOrg -
func (um *UserManager) RemoveUserFromOrg(userID string, orgID string) error {

	return um.ccGateway.DeleteResource(um.config.APIEndpoint(),
		fmt.Sprintf("/v2/users/%s/organizations/%s", userID, orgID))
}

// ListOrgsForUser -
func (um *UserManager) ListOrgsForUser(userID string, orgRole UserRoleInOrg) (orgIDs []string, err error) {

	orgList := &CCOrgResourceList{}
	err = um.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/users/%s/%s", um.config.APIEndpoint(), userID, orgRole), orgList)

	orgIDs = []string{}
	for _, o := range orgList.Resources {
		orgIDs = append(orgIDs, o.Metadata.GUID)
	}
	return
}

// FindByUsername -
func (um *UserManager) FindByUsername(username string) (models.UserFields, error) {
	return um.repo.FindByUsername(username)
}

// Delete -
func (um *UserManager) Delete(userID string) error {
	return um.repo.Delete(userID)
}
