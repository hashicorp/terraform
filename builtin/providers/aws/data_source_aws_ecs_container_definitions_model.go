package aws

type ecsContainerDefinition struct {
	Name              string                               `json:"name"`
	Image             string                               `json:"image"`
	Memory            int                                  `json:"memory,omitempty"`
	MemoryReservation int                                  `json:"memory_reservation,omitempty"`
	PortMappings      []*ecsContainerDefinitionPortMapping `json:"portMappings,omitempty"`
	CPU               int                                  `json:"cpu,omitempty"`
	Essential         bool                                 `json:"essential"`
	EntryPoint        []string                             `json:"entryPoint,omitempty"`
}

type ecsContainerDefinitionPortMapping struct {
	HostPort      int    `json:"hostPort,omitempty"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
}
