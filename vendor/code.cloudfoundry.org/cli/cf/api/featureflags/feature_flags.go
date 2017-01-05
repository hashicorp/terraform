package featureflags

import (
	"fmt"
	"strings"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . FeatureFlagRepository

type FeatureFlagRepository interface {
	List() ([]models.FeatureFlag, error)
	FindByName(string) (models.FeatureFlag, error)
	Update(string, bool) error
}

type CloudControllerFeatureFlagRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerFeatureFlagRepository(config coreconfig.Reader, gateway net.Gateway) CloudControllerFeatureFlagRepository {
	return CloudControllerFeatureFlagRepository{
		config:  config,
		gateway: gateway,
	}
}

func (repo CloudControllerFeatureFlagRepository) List() ([]models.FeatureFlag, error) {
	flags := []models.FeatureFlag{}
	apiError := repo.gateway.GetResource(
		fmt.Sprintf("%s/v2/config/feature_flags", repo.config.APIEndpoint()),
		&flags)

	if apiError != nil {
		return nil, apiError
	}

	return flags, nil
}

func (repo CloudControllerFeatureFlagRepository) FindByName(name string) (models.FeatureFlag, error) {
	flag := models.FeatureFlag{}
	apiError := repo.gateway.GetResource(
		fmt.Sprintf("%s/v2/config/feature_flags/%s", repo.config.APIEndpoint(), name),
		&flag)

	if apiError != nil {
		return models.FeatureFlag{}, apiError
	}

	return flag, nil
}

func (repo CloudControllerFeatureFlagRepository) Update(flag string, set bool) error {
	url := fmt.Sprintf("/v2/config/feature_flags/%s", flag)
	body := fmt.Sprintf(`{"enabled": %v}`, set)

	return repo.gateway.UpdateResource(repo.config.APIEndpoint(), url, strings.NewReader(body))
}
