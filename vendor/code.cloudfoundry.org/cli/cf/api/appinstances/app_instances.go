package appinstances

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

type InstancesAPIResponse map[string]InstanceAPIResponse

type InstanceAPIResponse struct {
	State   string
	Since   float64
	Details string
}

type StatsAPIResponse map[string]InstanceStatsAPIResponse

type InstanceStatsAPIResponse struct {
	Stats struct {
		DiskQuota int64 `json:"disk_quota"`
		MemQuota  int64 `json:"mem_quota"`
		Usage     struct {
			CPU  float64
			Disk int64
			Mem  int64
		}
	}
}

//go:generate counterfeiter . Repository

type Repository interface {
	GetInstances(appGUID string) (instances []models.AppInstanceFields, apiErr error)
	DeleteInstance(appGUID string, instance int) error
}

type CloudControllerAppInstancesRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerAppInstancesRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerAppInstancesRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerAppInstancesRepository) GetInstances(appGUID string) (instances []models.AppInstanceFields, err error) {
	instancesResponse := InstancesAPIResponse{}
	err = repo.gateway.GetResource(
		fmt.Sprintf("%s/v2/apps/%s/instances", repo.config.APIEndpoint(), appGUID),
		&instancesResponse)
	if err != nil {
		return
	}

	instances = make([]models.AppInstanceFields, len(instancesResponse), len(instancesResponse))
	for k, v := range instancesResponse {
		index, err := strconv.Atoi(k)
		if err != nil {
			continue
		}

		instances[index] = models.AppInstanceFields{
			State:   models.InstanceState(strings.ToLower(v.State)),
			Details: v.Details,
			Since:   time.Unix(int64(v.Since), 0),
		}
	}

	return repo.updateInstancesWithStats(appGUID, instances)
}

func (repo CloudControllerAppInstancesRepository) DeleteInstance(appGUID string, instance int) error {
	err := repo.gateway.DeleteResource(repo.config.APIEndpoint(), fmt.Sprintf("/v2/apps/%s/instances/%d", appGUID, instance))
	if err != nil {
		return err
	}
	return nil
}

func (repo CloudControllerAppInstancesRepository) updateInstancesWithStats(guid string, instances []models.AppInstanceFields) (updatedInst []models.AppInstanceFields, apiErr error) {
	path := fmt.Sprintf("%s/v2/apps/%s/stats", repo.config.APIEndpoint(), guid)
	statsResponse := StatsAPIResponse{}
	apiErr = repo.gateway.GetResource(path, &statsResponse)
	if apiErr != nil {
		return
	}

	updatedInst = make([]models.AppInstanceFields, len(statsResponse), len(statsResponse))
	for k, v := range statsResponse {
		index, err := strconv.Atoi(k)
		if err != nil {
			continue
		}

		instance := instances[index]
		instance.CPUUsage = v.Stats.Usage.CPU
		instance.DiskQuota = v.Stats.DiskQuota
		instance.DiskUsage = v.Stats.Usage.Disk
		instance.MemQuota = v.Stats.MemQuota
		instance.MemUsage = v.Stats.Usage.Mem

		updatedInst[index] = instance
	}
	return
}
