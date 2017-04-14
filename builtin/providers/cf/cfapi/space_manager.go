package cfapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/api/spaces"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/net"
)

// SpaceManager -
type SpaceManager struct {
	log *Logger

	config    coreconfig.Reader
	ccGateway net.Gateway

	apiEndpoint string

	repo spaces.SpaceRepository
}

// CCSpace -
type CCSpace struct {
	ID string

	Name      string `json:"name"`
	AllowSSH  bool   `json:"allow_ssh"`
	OrgGUID   string `json:"organization_guid"`
	QuotaGUID string `json:"space_quota_definition_guid,omitempty"`
}

// CCSpaceResource -
type CCSpaceResource struct {
	Metadata resources.Metadata `json:"metadata"`
	Entity   CCSpace            `json:"entity"`
}

// SpaceRole -
type SpaceRole string

// SpaceRoleManager -
const SpaceRoleManager = SpaceRole("managers")

// SpaceRoleDeveloper -
const SpaceRoleDeveloper = SpaceRole("developers")

// SpaceRoleAuditor -
const SpaceRoleAuditor = SpaceRole("auditors")

// NewSpaceManager -
func newSpaceManager(config coreconfig.Reader, ccGateway net.Gateway, logger *Logger) (dm *SpaceManager, err error) {

	dm = &SpaceManager{
		log: logger,

		config:    config,
		ccGateway: ccGateway,

		apiEndpoint: config.APIEndpoint(),

		repo: spaces.NewCloudControllerSpaceRepository(config, ccGateway),
	}

	if len(dm.apiEndpoint) == 0 {
		err = errors.New("API endpoint missing from config file")
		return
	}

	return
}

// FindSpaceInOrg -
func (sm *SpaceManager) FindSpaceInOrg(name string, org string) (space CCSpace, err error) {
	spaceModel, err := sm.repo.FindByNameInOrg(name, org)
	space.ID = spaceModel.GUID
	space.Name = spaceModel.Name
	space.OrgGUID = org
	space.QuotaGUID = spaceModel.SpaceQuotaGUID
	return
}

// FindSpace -
func (sm *SpaceManager) FindSpace(name string) (space CCSpace, err error) {
	spaceModel, err := sm.repo.FindByName(name)
	space.ID = spaceModel.GUID
	space.Name = spaceModel.Name
	space.OrgGUID = sm.config.OrganizationFields().GUID
	space.QuotaGUID = spaceModel.SpaceQuotaGUID
	return
}

// ReadSpace -
func (sm *SpaceManager) ReadSpace(spaceID string) (space CCSpace, err error) {

	resource := &CCSpaceResource{}
	err = sm.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/spaces/%s", sm.apiEndpoint, spaceID), &resource)

	space = resource.Entity
	space.ID = resource.Metadata.GUID
	return
}

// CreateSpace -
func (sm *SpaceManager) CreateSpace(
	name string, orgID string, quotaID string,
	allowSSH bool, asgs []interface{}) (id string, err error) {

	payload := map[string]interface{}{
		"name":              name,
		"organization_guid": orgID,
		"allow_ssh":         allowSSH,
	}
	if len(quotaID) > 0 {
		payload["space_quota_definition_guid"] = quotaID
	}
	if len(asgs) > 0 {
		payload["security_group_guids"] = asgs
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	resource := CCSpaceResource{}
	if err = sm.ccGateway.CreateResource(sm.apiEndpoint,
		"/v2/spaces", bytes.NewReader(body), &resource); err != nil {
		return
	}
	id = resource.Metadata.GUID
	return
}

// UpdateSpace -
func (sm *SpaceManager) UpdateSpace(space CCSpace, asgs []interface{}) (err error) {

	payload := map[string]interface{}{
		"name":                        space.Name,
		"organization_guid":           space.OrgGUID,
		"space_quota_definition_guid": space.QuotaGUID,
		"allow_ssh":                   strconv.FormatBool(space.AllowSSH),
	}
	if len(asgs) > 0 {
		payload["security_group_guids"] = asgs
	}

	body, err := json.Marshal(payload)

	request, err := sm.ccGateway.NewRequest("PUT",
		fmt.Sprintf("%s/v2/spaces/%s", sm.apiEndpoint, space.ID),
		sm.config.AccessToken(), bytes.NewReader(body))
	if err != nil {
		return err
	}

	resource := &CCSpaceResource{}
	_, err = sm.ccGateway.PerformRequestForJSONResponse(request, resource)
	return
}

// AddUsers -
func (sm *SpaceManager) AddUsers(spaceID string, userIDs []string, role SpaceRole) (err error) {

	for _, uid := range userIDs {
		err = sm.ccGateway.UpdateResource(sm.apiEndpoint,
			fmt.Sprintf("/v2/spaces/%s/%s/%s", spaceID, role, uid),
			strings.NewReader(""))
	}
	return
}

// RemoveUsers -
func (sm *SpaceManager) RemoveUsers(spaceID string, userIDs []string, role SpaceRole) (err error) {

	for _, uid := range userIDs {
		err = sm.ccGateway.DeleteResource(sm.apiEndpoint,
			fmt.Sprintf("/v2/spaces/%s/%s/%s", spaceID, role, uid))
	}
	return
}

// ListUsers -
func (sm *SpaceManager) ListUsers(spaceID string, role SpaceRole) (userIDs []interface{}, err error) {

	userList := &CCUserList{}
	err = sm.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/spaces/%s/%s", sm.apiEndpoint, spaceID, role), userList)
	for _, r := range userList.Resources {
		userIDs = append(userIDs, r.Metadata.GUID)
	}
	return
}

// ListASGs -
func (sm *SpaceManager) ListASGs(spaceID string) (asgIDs []interface{}, err error) {

	asgList := struct {
		Resources []struct {
			Metadata resources.Metadata `json:"metadata"`
		} `json:"resources"`
	}{}

	err = sm.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/spaces/%s/security_groups", sm.apiEndpoint, spaceID), &asgList)
	for _, r := range asgList.Resources {
		asgIDs = append(asgIDs, r.Metadata.GUID)
	}
	return
}

// DeleteSpace -
func (sm *SpaceManager) DeleteSpace(id string) (err error) {
	err = sm.ccGateway.DeleteResource(sm.apiEndpoint, fmt.Sprintf("/v2/spaces/%s", id))
	return
}
