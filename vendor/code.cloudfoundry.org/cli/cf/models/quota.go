package models

import "encoding/json"

type QuotaFields struct {
	GUID                    string      `json:"guid,omitempty"`
	Name                    string      `json:"name"`
	MemoryLimit             int64       `json:"memory_limit"`          // in Megabytes
	InstanceMemoryLimit     int64       `json:"instance_memory_limit"` // in Megabytes
	RoutesLimit             int         `json:"total_routes"`
	ServicesLimit           int         `json:"total_services"`
	NonBasicServicesAllowed bool        `json:"non_basic_services_allowed"`
	AppInstanceLimit        int         `json:"app_instance_limit"`
	ReservedRoutePorts      json.Number `json:"total_reserved_route_ports,omitempty"`
}

type QuotaResponse struct {
	GUID                    string      `json:"guid,omitempty"`
	Name                    string      `json:"name"`
	MemoryLimit             int64       `json:"memory_limit"`          // in Megabytes
	InstanceMemoryLimit     int64       `json:"instance_memory_limit"` // in Megabytes
	RoutesLimit             int         `json:"total_routes"`
	ServicesLimit           int         `json:"total_services"`
	NonBasicServicesAllowed bool        `json:"non_basic_services_allowed"`
	AppInstanceLimit        json.Number `json:"app_instance_limit"`
	ReservedRoutePorts      json.Number `json:"total_reserved_route_ports"`
}
