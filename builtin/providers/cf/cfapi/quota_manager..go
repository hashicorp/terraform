package cfapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"code.cloudfoundry.org/cli/cf/api/quotas"
	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/api/spacequotas"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/net"
)

// QuotaManager -
type QuotaManager struct {
	log *Logger

	config    coreconfig.Reader
	ccGateway net.Gateway

	apiEndpoint string

	repo      quotas.QuotaRepository
	spaceRepo spacequotas.SpaceQuotaRepository
}

// CCQuota -
type CCQuota struct {
	ID                      string
	OrgGUID                 string `json:"organization_guid,omitempty"`
	Name                    string `json:"name"`
	AppInstanceLimit        int    `json:"app_instance_limit"`
	AppTaskLimit            int    `json:"app_task_limit"`
	InstanceMemoryLimit     int64  `json:"instance_memory_limit"`
	MemoryLimit             int64  `json:"memory_limit"`
	NonBasicServicesAllowed bool   `json:"non_basic_services_allowed"`
	TotalServices           int    `json:"total_services"`
	TotalServiceKeys        int    `json:"total_service_keys"`
	TotalRoutes             int    `json:"total_routes"`
	TotalReserveredPorts    int    `json:"total_reserved_route_ports,omitempty"`
	TotalPrivateDomains     int    `json:"total_private_domains"`
}

// CCQuotaResource -
type CCQuotaResource struct {
	Metadata resources.Metadata `json:"metadata"`
	Entity   CCQuota            `json:"entity"`
}

// NewQuotaManager -
func newQuotaManager(config coreconfig.Reader, ccGateway net.Gateway, logger *Logger) (dm *QuotaManager, err error) {

	dm = &QuotaManager{
		log: logger,

		config:    config,
		ccGateway: ccGateway,

		apiEndpoint: config.APIEndpoint(),

		repo:      quotas.NewCloudControllerQuotaRepository(config, ccGateway),
		spaceRepo: spacequotas.NewCloudControllerSpaceQuotaRepository(config, ccGateway),
	}

	if len(dm.apiEndpoint) == 0 {
		err = errors.New("API endpoint missing from config file")
		return
	}

	return
}

// CreateQuota -
func (qm *QuotaManager) CreateQuota(quota CCQuota) (id string, err error) {

	body, err := json.Marshal(quota)
	if err != nil {
		return
	}

	var url string
	if len(quota.OrgGUID) > 0 {
		url = "/v2/space_quota_definitions"
	} else {
		url = "/v2/quota_definitions"
	}

	resource := CCQuotaResource{}
	if err = qm.ccGateway.CreateResource(qm.apiEndpoint, url,
		bytes.NewReader(body), &resource); err != nil {
		return
	}
	id = resource.Metadata.GUID
	return
}

// UpdateQuota -
func (qm *QuotaManager) UpdateQuota(quota CCQuota) (err error) {

	body, err := json.Marshal(quota)
	if err != nil {
		return
	}

	var url string
	if len(quota.OrgGUID) > 0 {
		url = fmt.Sprintf("/v2/space_quota_definitions/%s", quota.ID)
	} else {
		url = fmt.Sprintf("/v2/quota_definitions/%s", quota.ID)
	}

	resource := CCQuotaResource{}
	err = qm.ccGateway.UpdateResource(qm.apiEndpoint, url,
		bytes.NewReader(body), &resource)
	return
}

// ReadQuota -
func (qm *QuotaManager) ReadQuota(id string) (quota CCQuota, err error) {

	resource := CCQuotaResource{}

	if err = qm.ccGateway.GetResource(
		fmt.Sprintf("%s/v2/quota_definitions/%s", qm.apiEndpoint, id),
		&resource); err != nil {

		if err = qm.ccGateway.GetResource(
			fmt.Sprintf("%s/v2/space_quota_definitions/%s", qm.apiEndpoint, id),
			&resource); err != nil {

			return
		}
	}
	quota = resource.Entity
	quota.ID = resource.Metadata.GUID
	return
}

// FindQuota -
func (qm *QuotaManager) FindQuota(name string) (quota CCQuota, err error) {

	quotaFields, err := qm.repo.FindByName(name)
	if err == nil {
		quota.ID = quotaFields.GUID
		quota.Name = quotaFields.Name
		quota.AppInstanceLimit = quotaFields.AppInstanceLimit
		quota.InstanceMemoryLimit = quotaFields.InstanceMemoryLimit
		quota.MemoryLimit = quotaFields.MemoryLimit
		quota.NonBasicServicesAllowed = quotaFields.NonBasicServicesAllowed
		quota.TotalServices = quotaFields.ServicesLimit
		quota.TotalRoutes = quotaFields.RoutesLimit
		quota.TotalReserveredPorts, _ = strconv.Atoi(quotaFields.ReservedRoutePorts.String())
	}
	return
}

// FindSpaceQuota -
func (qm *QuotaManager) FindSpaceQuota(name string, orgGUID string) (quota CCQuota, err error) {

	spaceQuota, err := qm.spaceRepo.FindByNameAndOrgGUID(name, orgGUID)
	if err == nil {
		quota.ID = spaceQuota.GUID
		quota.OrgGUID = spaceQuota.OrgGUID
		quota.Name = spaceQuota.Name
		quota.AppInstanceLimit = spaceQuota.AppInstanceLimit
		quota.InstanceMemoryLimit = spaceQuota.InstanceMemoryLimit
		quota.MemoryLimit = spaceQuota.MemoryLimit
		quota.NonBasicServicesAllowed = spaceQuota.NonBasicServicesAllowed
		quota.TotalServices = spaceQuota.ServicesLimit
		quota.TotalRoutes = spaceQuota.RoutesLimit
	}
	return
}

// DeleteQuota -
func (qm *QuotaManager) DeleteQuota(id string, orgGUID string) (err error) {

	var url string
	if len(orgGUID) > 0 {
		url = fmt.Sprintf("/v2/space_quota_definitions/%s", id)
	} else {
		url = fmt.Sprintf("/v2/quota_definitions/%s", id)
	}

	err = qm.ccGateway.DeleteResource(qm.apiEndpoint, url)
	return
}
