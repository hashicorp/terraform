package azure

import "strings"

type Endpoints struct {
	resourceManagerEndpointUrl string
	activeDirectoryEndpointUrl string
}

func GetEndpoints(location string) Endpoints {
	var e Endpoints

	location = strings.Replace(strings.ToLower(location), " ", "", -1)

	switch location {
	case GermanyCentral, GermanyEast:
		e = Endpoints{"https://management.microsoftazure.de", "https://login.microsoftonline.de"}
	case ChinaEast, ChinaNorth:
		e = Endpoints{"https://management.chinacloudapi.cn", "https://login.chinacloudapi.cn"}
	case USGovIowa, USGovVirginia:
		e = Endpoints{"https://management.usgovcloudapi.net", "https://login.microsoftonline.com"}
	default:
		e = Endpoints{"https://management.azure.com", "https://login.microsoftonline.com"}
	}

	return e
}
