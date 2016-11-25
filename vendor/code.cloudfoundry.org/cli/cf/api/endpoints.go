package api

import (
	"strings"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/net"
)

type RemoteInfoRepository struct {
	gateway net.Gateway
}

func NewEndpointRepository(gateway net.Gateway) RemoteInfoRepository {
	r := RemoteInfoRepository{
		gateway: gateway,
	}
	return r
}

func (repo RemoteInfoRepository) GetCCInfo(endpoint string) (*coreconfig.CCInfo, string, error) {
	if strings.HasPrefix(endpoint, "http") {
		serverResponse, err := repo.getCCAPIInfo(endpoint)
		if err != nil {
			return nil, "", err
		}

		return serverResponse, endpoint, nil
	}

	finalEndpoint := "https://" + endpoint
	serverResponse, err := repo.getCCAPIInfo(finalEndpoint)
	if err != nil {
		return nil, "", err
	}

	return serverResponse, finalEndpoint, nil
}

func (repo RemoteInfoRepository) getCCAPIInfo(endpoint string) (*coreconfig.CCInfo, error) {
	serverResponse := new(coreconfig.CCInfo)
	err := repo.gateway.GetResource(endpoint+"/v2/info", &serverResponse)
	if err != nil {
		return nil, err
	}

	return serverResponse, nil
}
