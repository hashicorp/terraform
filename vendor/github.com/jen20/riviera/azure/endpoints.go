package azure

import (
	"reflect"
	"strings"
)

type Endpoints struct {
	resourceManagerEndpointUrl string
	activeDirectoryEndpointUrl string
}

var (
	ChineseEndpoints = Endpoints{"https://management.chinacloudapi.cn", "https://login.chinacloudapi.cn"}
	DefaultEndpoints = Endpoints{"https://management.azure.com", "https://login.microsoftonline.com"}
	GermanEndpoints  = Endpoints{"https://management.microsoftazure.de", "https://login.microsoftonline.de"}
	USGovEndpoints   = Endpoints{"https://management.usgovcloudapi.net", "https://login.microsoftonline.com"}
)

func GetEndpointsForLocation(location string) Endpoints {
	location = strings.Replace(strings.ToLower(location), " ", "", -1)

	switch location {
	case GermanyCentral, GermanyEast:
		return GermanEndpoints
	case ChinaEast, ChinaNorth:
		return ChineseEndpoints
	case USGovIowa, USGovVirginia:
		return USGovEndpoints
	default:
		return DefaultEndpoints
	}
}

func GetEndpointsForCommand(command APICall) Endpoints {
	locationField := reflect.Indirect(reflect.ValueOf(command)).FieldByName("Location")
	if locationField.IsValid() {
		location := locationField.Interface().(string)
		return GetEndpointsForLocation(location)
	}

	return DefaultEndpoints
}
