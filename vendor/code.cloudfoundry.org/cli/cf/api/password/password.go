package password

import (
	"fmt"
	"strings"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	. "code.cloudfoundry.org/cli/cf/i18n"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . Repository

type Repository interface {
	UpdatePassword(old string, new string) error
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

func (repo CloudControllerRepository) UpdatePassword(old string, new string) error {
	uaaEndpoint := repo.config.UaaEndpoint()
	if uaaEndpoint == "" {
		return errors.New(T("UAA endpoint missing from config file"))
	}

	url := fmt.Sprintf("/Users/%s/password", repo.config.UserGUID())
	body := fmt.Sprintf(`{"password":"%s","oldPassword":"%s"}`, new, old)

	return repo.gateway.UpdateResource(uaaEndpoint, url, strings.NewReader(body))
}
