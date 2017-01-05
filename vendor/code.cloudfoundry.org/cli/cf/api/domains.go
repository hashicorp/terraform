package api

import (
	"bytes"
	"encoding/json"

	. "code.cloudfoundry.org/cli/cf/i18n"

	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/api/strategy"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . DomainRepository

type DomainRepository interface {
	ListDomainsForOrg(orgGUID string, cb func(models.DomainFields) bool) error
	FindSharedByName(name string) (domain models.DomainFields, apiErr error)
	FindPrivateByName(name string) (domain models.DomainFields, apiErr error)
	FindByNameInOrg(name string, owningOrgGUID string) (domain models.DomainFields, apiErr error)
	Create(domainName string, owningOrgGUID string) (createdDomain models.DomainFields, apiErr error)
	CreateSharedDomain(domainName string, routerGroupGUID string) (apiErr error)
	Delete(domainGUID string) (apiErr error)
	DeleteSharedDomain(domainGUID string) (apiErr error)
	FirstOrDefault(orgGUID string, name *string) (domain models.DomainFields, error error)
}

type CloudControllerDomainRepository struct {
	config   coreconfig.Reader
	gateway  net.Gateway
	strategy strategy.EndpointStrategy
}

func NewCloudControllerDomainRepository(config coreconfig.Reader, gateway net.Gateway, strategy strategy.EndpointStrategy) CloudControllerDomainRepository {
	return CloudControllerDomainRepository{
		config:   config,
		gateway:  gateway,
		strategy: strategy,
	}
}

func (repo CloudControllerDomainRepository) ListDomainsForOrg(orgGUID string, cb func(models.DomainFields) bool) error {
	err := repo.listDomains(repo.strategy.PrivateDomainsByOrgURL(orgGUID), cb)
	if err != nil {
		return err
	}
	err = repo.listDomains(repo.strategy.SharedDomainsURL(), cb)
	return err
}

func (repo CloudControllerDomainRepository) listDomains(path string, cb func(models.DomainFields) bool) error {
	return repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		path,
		resources.DomainResource{},
		func(resource interface{}) bool {
			return cb(resource.(resources.DomainResource).ToFields())
		})
}

func (repo CloudControllerDomainRepository) isOrgDomain(orgGUID string, domain models.DomainFields) bool {
	return orgGUID == domain.OwningOrganizationGUID || domain.Shared
}

func (repo CloudControllerDomainRepository) FindSharedByName(name string) (domain models.DomainFields, apiErr error) {
	return repo.findOneWithPath(repo.strategy.SharedDomainURL(name), name)
}

func (repo CloudControllerDomainRepository) FindPrivateByName(name string) (domain models.DomainFields, apiErr error) {
	return repo.findOneWithPath(repo.strategy.PrivateDomainURL(name), name)
}

func (repo CloudControllerDomainRepository) FindByNameInOrg(name string, orgGUID string) (models.DomainFields, error) {
	domain, err := repo.findOneWithPath(repo.strategy.OrgDomainURL(orgGUID, name), name)

	switch err.(type) {
	case *errors.ModelNotFoundError:
		domain, err = repo.FindSharedByName(name)
		if err != nil {
			return models.DomainFields{}, err
		}
		if !domain.Shared {
			err = errors.NewModelNotFoundError("Domain", name)
		}
	}

	return domain, err
}

func (repo CloudControllerDomainRepository) findOneWithPath(path, name string) (models.DomainFields, error) {
	var domain models.DomainFields

	foundDomain := false
	err := repo.listDomains(path, func(result models.DomainFields) bool {
		domain = result
		foundDomain = true
		return false
	})

	if err == nil && !foundDomain {
		err = errors.NewModelNotFoundError("Domain", name)
	}

	return domain, err
}

func (repo CloudControllerDomainRepository) Create(domainName string, owningOrgGUID string) (createdDomain models.DomainFields, err error) {
	data, err := json.Marshal(resources.DomainEntity{
		Name: domainName,
		OwningOrganizationGUID: owningOrgGUID,
		Wildcard:               true,
	})

	if err != nil {
		return
	}

	resource := new(resources.DomainResource)
	err = repo.gateway.CreateResource(
		repo.config.APIEndpoint(),
		repo.strategy.PrivateDomainsURL(),
		bytes.NewReader(data),
		resource)

	if err != nil {
		return
	}

	createdDomain = resource.ToFields()
	return
}

func (repo CloudControllerDomainRepository) CreateSharedDomain(domainName string, routerGroupGUID string) error {
	data, err := json.Marshal(resources.DomainEntity{
		Name:            domainName,
		RouterGroupGUID: routerGroupGUID,
		Wildcard:        true,
	})
	if err != nil {
		return err
	}

	return repo.gateway.CreateResource(
		repo.config.APIEndpoint(),
		repo.strategy.SharedDomainsURL(),
		bytes.NewReader(data),
	)
}

func (repo CloudControllerDomainRepository) Delete(domainGUID string) error {
	return repo.gateway.DeleteResource(
		repo.config.APIEndpoint(),
		repo.strategy.DeleteDomainURL(domainGUID))
}

func (repo CloudControllerDomainRepository) DeleteSharedDomain(domainGUID string) error {
	return repo.gateway.DeleteResource(
		repo.config.APIEndpoint(),
		repo.strategy.DeleteSharedDomainURL(domainGUID))
}

func (repo CloudControllerDomainRepository) FirstOrDefault(orgGUID string, name *string) (domain models.DomainFields, error error) {
	if name == nil {
		domain, error = repo.defaultDomain(orgGUID)
	} else {
		domain, error = repo.FindByNameInOrg(*name, orgGUID)
	}
	return
}

func (repo CloudControllerDomainRepository) defaultDomain(orgGUID string) (models.DomainFields, error) {
	var foundDomain *models.DomainFields
	err := repo.ListDomainsForOrg(orgGUID, func(domain models.DomainFields) bool {
		foundDomain = &domain
		return !domain.Shared
	})
	if err != nil {
		return models.DomainFields{}, err
	}

	if foundDomain == nil {
		return models.DomainFields{}, errors.New(T("Could not find a default domain"))
	}

	return *foundDomain, nil
}
