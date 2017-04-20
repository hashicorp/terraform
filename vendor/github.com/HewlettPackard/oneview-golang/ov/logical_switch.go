package ov

import (
	"github.com/HewlettPackard/oneview-golang/utils"
)

type LogicalSwitch struct {
	Category                string                  `json:"category,omitempty"`          // "category": "logcial-switch-groups",
	ConstitencyStatus       string                  `json:"constitencyStatus,omitempty"` //"consitencyStatue": "CONSISTENT",
	Created                 string                  `json:"created,omitempty"`           // "created": "20150831T154835.250Z",
	Description             utils.Nstring           `json:"description,omitempty"`       // "description": "Logical Switch 1",
	ETAG                    string                  `json:"eTag,omitempty"`              // "eTag": "1441036118675/8",
	FabricUri               utils.Nstring           `json:"fabricUri,omitempty"`         // "fabricUri": "/rest/fabrics/9b8f7ec0-52b3-475e-84f4-c4eac51c2c20",
	LogicalSwitchDomainInfo LogicalSwitchDomainInfo `json:"logicalSwitchDomainInfo"`
	Modified                string                  `json:"modified,omitempty"` // "modified": "20150831T154835.250Z",
	Name                    string                  `json:"name,omitempty"`     // "name": "Logical Switch Group1",
	State                   string                  `json:"state,omitempty"`    // "state": "Normal",
	Status                  string                  `json:"status,omitempty"`   // "status": "Critical",
	Type                    string                  `json:"type,omitempty"`     // "type": "logical-switch-groups",
	URI                     utils.Nstring           `json:"uri,omitempty"`      // "uri": "/rest/logical-switch-groups/e2f0031b-52bd-4223-9ac1-d91cb519d548",
	SwitchMapTemplate       SwitchMapTemplate       `json:"switchMapTemplate"`
}

type LogicalSwitchDomainInfo struct {
	DomainId         string              `json:"domainId"`         //"domainId": "NA",
	MasterMacAddress string              `json:"masterMacAddress"` //"masterMacAddress": "NA",
	PerSwitchDomain  []LSDomainPerSwitch `json:"perSwitchDomain"`
}

type LSDomainPerSwitch struct {
	FirmwareVersion string `json:"firmwareVersion"` //"firmwareVersion": "unknown",
	IPAddress       string `json:"ipAddress"`       //"ipAddress": "172.18.1.11",
}
