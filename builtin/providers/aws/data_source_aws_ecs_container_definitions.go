package aws

import (
	"encoding/json"
	"strconv"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsEcsContainerDefinitions() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEcsContainerDefinitionsRead,

		Schema: map[string]*schema.Schema{
			"container": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateDataSourceAwsEcsContainerDefinitionName,
						},
						"image": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateDataSourceAwsEcsContainerDefinitionImage,
						},
						"memory": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validateDataSourceAwsEcsContainerDefinitionMemory,
						},
						"memory_reservation": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validateDataSourceAwsEcsContainerDefinitionMemoryReservation,
						},
						"port_mapping": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host_port": {
										Type:         schema.TypeInt,
										Optional:     true,
										ValidateFunc: validateDataSourceAwsEcsContainerDefinitionHostPort,
									},
									"container_port": {
										Type:         schema.TypeInt,
										Required:     true,
										ValidateFunc: validateDataSourceAwsEcsContainerDefinitionContainerPort,
									},
									"protocol": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateDataSourceAwsEcsContainerDefinitionContainerPort,
									},
								},
							},
						},
						"cpu": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validateDataSourceAwsEcsContainerDefinitionCpu,
						},
						"essential": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"entry_point": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"json": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsEcsContainerDefinitionsRead(d *schema.ResourceData, meta interface{}) error {
	containersConfig := d.Get("container").([]interface{})
	containers := make([]*ecsContainerDefinition, len(containersConfig))

	for i, configMap := range containersConfig {
		container := &ecsContainerDefinition{}
		config := configMap.(map[string]interface{})

		container.Name = config["name"].(string)
		container.Image = config["image"].(string)
		container.Memory = config["memory"].(int)
		container.MemoryReservation = config["memory_reservation"].(int)

		portMappings := config["port_mapping"].([]interface{})
		container.PortMappings = make([]*ecsContainerDefinitionPortMapping, len(portMappings))
		for i, portMappingMap := range portMappings {
			portMapping := portMappingMap.(map[string]interface{})
			container.PortMappings[i] = &ecsContainerDefinitionPortMapping{
				HostPort:      portMapping["host_port"].(int),
				ContainerPort: portMapping["container_port"].(int),
				Protocol:      portMapping["protocol"].(string),
			}
		}

		container.CPU = config["cpu"].(int)
		container.Essential = config["essential"].(bool)
		container.EntryPoint = stringSliceFromInterfaceSlice(config["entry_point"])

		containers[i] = container
	}

	jsonDoc, err := json.MarshalIndent(containers, "", "  ")
	if err != nil {
		// should never happen if the above code is correct
		return err
	}
	jsonString := string(jsonDoc)

	d.Set("json", jsonString)
	d.SetId(strconv.Itoa(hashcode.String(jsonString)))

	return nil
}

func stringSliceFromInterfaceSlice(value interface{}) []string {
	v := value.([]interface{})

	result := make([]string, len(v))
	for i, val := range v {
		result[i] = val.(string)
	}

	return result
}

func validateDataSourceAwsEcsContainerDefinitionName(v interface{}, k string) (ws []string, es []error) {
	//TODO(jen20) up to 255 letters (uppercase and lowercase), numbers, hyphens, and underscores are allowed.
	return
}

func validateDataSourceAwsEcsContainerDefinitionImage(v interface{}, k string) (ws []string, es []error) {
	//TODO(jen20) Up to 255 letters (uppercase and lowercase), numbers, hyphens, underscores, colons, periods, forward slashes, and number signs are allowed.
	return
}

func validateDataSourceAwsEcsContainerDefinitionMemory(v interface{}, k string) (ws []string, es []error) {
	return
}

func validateDataSourceAwsEcsContainerDefinitionMemoryReservation(v interface{}, k string) (ws []string, es []error) {
	return
}

func validateDataSourceAwsEcsContainerDefinitionHostPort(v interface{}, k string) (ws []string, es []error) {
	return
}

func validateDataSourceAwsEcsContainerDefinitionContainerPort(v interface{}, k string) (ws []string, es []error) {
	return
}

func validateDataSourceAwsEcsContainerDefinitionCpu(v interface{}, k string) (ws []string, es []error) {
	return
}
