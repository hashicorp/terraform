package environmentvariablegroups

import (
	"fmt"
	"strings"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . Repository

type Repository interface {
	ListRunning() (variables []models.EnvironmentVariable, apiErr error)
	ListStaging() (variables []models.EnvironmentVariable, apiErr error)
	SetStaging(string) error
	SetRunning(string) error
}

type CloudControllerRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerRepository) ListRunning() (variables []models.EnvironmentVariable, apiErr error) {
	var rawResponse interface{}
	url := fmt.Sprintf("%s/v2/config/environment_variable_groups/running", repo.config.APIEndpoint())
	apiErr = repo.gateway.GetResource(url, &rawResponse)
	if apiErr != nil {
		return
	}

	variables, err := repo.marshalToEnvironmentVariables(rawResponse)
	if err != nil {
		return nil, err
	}

	return variables, nil
}

func (repo CloudControllerRepository) ListStaging() (variables []models.EnvironmentVariable, apiErr error) {
	var rawResponse interface{}
	url := fmt.Sprintf("%s/v2/config/environment_variable_groups/staging", repo.config.APIEndpoint())
	apiErr = repo.gateway.GetResource(url, &rawResponse)
	if apiErr != nil {
		return
	}

	variables, err := repo.marshalToEnvironmentVariables(rawResponse)
	if err != nil {
		return nil, err
	}

	return variables, nil
}

func (repo CloudControllerRepository) SetStaging(stagingVars string) error {
	return repo.gateway.UpdateResource(repo.config.APIEndpoint(), "/v2/config/environment_variable_groups/staging", strings.NewReader(stagingVars))
}

func (repo CloudControllerRepository) SetRunning(runningVars string) error {
	return repo.gateway.UpdateResource(repo.config.APIEndpoint(), "/v2/config/environment_variable_groups/running", strings.NewReader(runningVars))
}

func (repo CloudControllerRepository) marshalToEnvironmentVariables(rawResponse interface{}) ([]models.EnvironmentVariable, error) {
	var variables []models.EnvironmentVariable
	for key, value := range rawResponse.(map[string]interface{}) {
		stringvalue, err := repo.convertValueToString(value)
		if err != nil {
			return nil, err
		}
		variable := models.EnvironmentVariable{Name: key, Value: stringvalue}
		variables = append(variables, variable)
	}
	return variables, nil
}

func (repo CloudControllerRepository) convertValueToString(value interface{}) (string, error) {
	stringvalue, ok := value.(string)
	if !ok {
		floatvalue, ok := value.(float64)
		if !ok {
			return "", fmt.Errorf("Attempted to read environment variable value of unknown type: %#v", value)
		}
		stringvalue = fmt.Sprintf("%d", int(floatvalue))
	}
	return stringvalue, nil
}
