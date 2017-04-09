package cfapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/api/securitygroups"
	running "code.cloudfoundry.org/cli/cf/api/securitygroups/defaults/running"
	staging "code.cloudfoundry.org/cli/cf/api/securitygroups/defaults/staging"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

// ASGManager -
type ASGManager struct {
	log *Logger

	config    coreconfig.Reader
	ccGateway net.Gateway

	apiEndpoint string

	repo        securitygroups.SecurityGroupRepo
	runningRepo running.SecurityGroupsRepo
	stagingRepo staging.SecurityGroupsRepo
}

// CCASGRule -
type CCASGRule struct {
	Protocol    string `json:"protocol"`
	Destination string `json:"destination"`
	Ports       string `json:"ports,omitempty"`
	Log         bool   `json:"log,omitempty"`
	Code        int    `json:"code,omitempty"`
	Type        int    `json:"type,omitempty"`
}

// CCASG -
type CCASG struct {
	ID               string
	Name             string      `json:"name"`
	Rules            []CCASGRule `json:"rules"`
	IsRunningDefault bool        `json:"running_default"`
	IsStagingDefault bool        `json:"staging_default"`
}

// CCASGResource -
type CCASGResource struct {
	Metadata resources.Metadata `json:"metadata"`
	Entity   CCASG              `json:"entity"`
}

// NewASGManager -
func newASGManager(config coreconfig.Reader, ccGateway net.Gateway, logger *Logger) (dm *ASGManager, err error) {

	dm = &ASGManager{
		log: logger,

		config:    config,
		ccGateway: ccGateway,

		apiEndpoint: config.APIEndpoint(),

		repo:        securitygroups.NewSecurityGroupRepo(config, ccGateway),
		runningRepo: running.NewSecurityGroupsRepo(config, ccGateway),
		stagingRepo: staging.NewSecurityGroupsRepo(config, ccGateway),
	}

	if len(dm.apiEndpoint) == 0 {
		err = errors.New("API endpoint missing from config file")
		return
	}

	return
}

// CreateASG -
func (am *ASGManager) CreateASG(name string, rules []CCASGRule) (id string, err error) {

	body, err := json.Marshal(map[string]interface{}{
		"name":  name,
		"rules": rules,
	})
	if err != nil {
		return
	}

	resource := CCASGResource{}
	if err = am.ccGateway.CreateResource(am.apiEndpoint,
		"/v2/security_groups", bytes.NewReader(body), &resource); err != nil {
		return
	}
	id = resource.Metadata.GUID
	return
}

// UpdateASG -
func (am *ASGManager) UpdateASG(id string, name string, rules []CCASGRule) (err error) {

	body, err := json.Marshal(map[string]interface{}{
		"name":  name,
		"rules": rules,
	})

	request, err := am.ccGateway.NewRequest("PUT",
		fmt.Sprintf("%s/v2/security_groups/%s", am.apiEndpoint, id),
		am.config.AccessToken(), bytes.NewReader(body))
	if err != nil {
		return err
	}

	resource := &CCASGResource{}
	_, err = am.ccGateway.PerformRequestForJSONResponse(request, resource)
	return
}

// GetASG -
func (am *ASGManager) GetASG(id string) (asg CCASG, err error) {

	resource := &CCASGResource{}
	err = am.ccGateway.GetResource(fmt.Sprintf("%s/v2/security_groups/%s", am.apiEndpoint, id), resource)
	asg = resource.Entity
	asg.ID = resource.Metadata.GUID
	return
}

// Delete -
func (am *ASGManager) Delete(id string) (err error) {
	err = am.ccGateway.DeleteResource(am.apiEndpoint, fmt.Sprintf("/v2/security_groups/%s", id))
	return
}

// Read -
func (am *ASGManager) Read(name string) (models.SecurityGroup, error) {
	return am.repo.Read(name)
}

// Running -
func (am *ASGManager) Running() (asgs []string, err error) {
	securityGroups, err := am.runningRepo.List()
	for _, s := range securityGroups {
		asgs = append(asgs, s.GUID)
	}
	return
}

// BindToRunning -
func (am *ASGManager) BindToRunning(id string) error {
	return am.runningRepo.BindToRunningSet(id)
}

// UnbindFromRunning -
func (am *ASGManager) UnbindFromRunning(id string) error {
	return am.runningRepo.UnbindFromRunningSet(id)
}

// UnbindAllFromRunning -
func (am *ASGManager) UnbindAllFromRunning() (err error) {
	securityGroups, err := am.runningRepo.List()
	if err != nil {
		return
	}
	for _, s := range securityGroups {
		err = am.runningRepo.UnbindFromRunningSet(s.GUID)
		if err != nil {
			return
		}
	}
	return
}

// Staging -
func (am *ASGManager) Staging() (asgs []string, err error) {
	securityGroups, err := am.stagingRepo.List()
	for _, s := range securityGroups {
		asgs = append(asgs, s.GUID)
	}
	return
}

// BindToStaging -
func (am *ASGManager) BindToStaging(id string) error {
	return am.stagingRepo.BindToStagingSet(id)
}

// UnbindFromStaging -
func (am *ASGManager) UnbindFromStaging(id string) error {
	return am.stagingRepo.UnbindFromStagingSet(id)
}

// UnbindAllFromStaging -
func (am *ASGManager) UnbindAllFromStaging() (err error) {
	securityGroups, err := am.stagingRepo.List()
	if err != nil {
		return
	}
	for _, s := range securityGroups {
		err = am.stagingRepo.UnbindFromStagingSet(s.GUID)
		if err != nil {
			return
		}
	}
	return
}
