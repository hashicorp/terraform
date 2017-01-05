package repository

import (
	"encoding/json"
	"net/url"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/v3/models"
	"github.com/cloudfoundry/go-ccapi/v3/client"
)

//go:generate counterfeiter . Repository

type Repository interface {
	GetApplications() ([]models.V3Application, error)
	GetProcesses(path string) ([]models.V3Process, error)
	GetRoutes(path string) ([]models.V3Route, error)
}

type repository struct {
	client client.Client
	config coreconfig.ReadWriter
}

func NewRepository(config coreconfig.ReadWriter, client client.Client) Repository {
	return &repository{
		client: client,
		config: config,
	}
}

func (r *repository) handleUpdatedTokens() {
	if r.client.TokensUpdated() {
		accessToken, refreshToken := r.client.GetUpdatedTokens()
		r.config.SetAccessToken(accessToken)
		r.config.SetRefreshToken(refreshToken)
	}
}

func (r *repository) GetApplications() ([]models.V3Application, error) {
	jsonResponse, err := r.client.GetApplications(url.Values{})
	if err != nil {
		return []models.V3Application{}, err
	}

	r.handleUpdatedTokens()

	applications := []models.V3Application{}
	err = json.Unmarshal(jsonResponse, &applications)
	if err != nil {
		return []models.V3Application{}, err
	}

	return applications, nil
}

func (r *repository) GetProcesses(path string) ([]models.V3Process, error) {
	jsonResponse, err := r.client.GetResources(path, 0)
	if err != nil {
		return []models.V3Process{}, err
	}

	r.handleUpdatedTokens()

	processes := []models.V3Process{}
	err = json.Unmarshal(jsonResponse, &processes)
	if err != nil {
		return []models.V3Process{}, err
	}

	return processes, nil
}

func (r *repository) GetRoutes(path string) ([]models.V3Route, error) {
	jsonResponse, err := r.client.GetResources(path, 0)
	if err != nil {
		return []models.V3Route{}, err
	}

	r.handleUpdatedTokens()

	routes := []models.V3Route{}
	err = json.Unmarshal(jsonResponse, &routes)
	if err != nil {
		return []models.V3Route{}, err
	}

	return routes, nil
}
