package aws

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsServiceDiscoveryService() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsServiceDiscoveryServiceCreate,
		Read:   resourceAwsServiceDiscoveryServiceRead,
		Update: resourceAwsServiceDiscoveryServiceUpdate,
		Delete: resourceAwsServiceDiscoveryServiceDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"dns_config": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"namespace_id": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"dns_records": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ttl": {
										Type:     schema.TypeInt,
										Required: true,
									},
									"type": {
										Type:         schema.TypeString,
										Required:     true,
										ForceNew:     true,
										ValidateFunc: validateServiceDiscoveryServiceDnsRecordsType,
									},
								},
							},
						},
					},
				},
			},
			"health_check_config": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"failure_threshold": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"resource_path": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"type": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateServiceDiscoveryServiceHealthCheckConfigType,
						},
					},
				},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsServiceDiscoveryServiceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sdconn

	input := &servicediscovery.CreateServiceInput{
		Name:      aws.String(d.Get("name").(string)),
		DnsConfig: expandServiceDiscoveryDnsConfig(d.Get("dns_config").([]interface{})[0].(map[string]interface{})),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	hcconfig := d.Get("health_check_config").([]interface{})
	if len(hcconfig) > 0 {
		input.HealthCheckConfig = expandServiceDiscoveryHealthCheckConfig(hcconfig[0].(map[string]interface{}))
	}

	resp, err := conn.CreateService(input)
	if err != nil {
		return err
	}

	d.SetId(*resp.Service.Id)
	d.Set("arn", resp.Service.Arn)
	return nil
}

func resourceAwsServiceDiscoveryServiceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sdconn

	input := &servicediscovery.GetServiceInput{
		Id: aws.String(d.Id()),
	}

	resp, err := conn.GetService(input)
	if err != nil {
		if isAWSErr(err, servicediscovery.ErrCodeServiceNotFound, "") {
			log.Printf("[WARN] Service Discovery Service (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	service := resp.Service
	d.Set("arn", service.Arn)
	d.Set("name", service.Name)
	d.Set("description", service.Description)
	d.Set("dns_config", flattenServiceDiscoveryDnsConfig(service.DnsConfig))
	d.Set("health_check_config", flattenServiceDiscoveryHealthCheckConfig(service.HealthCheckConfig))
	return nil
}

func resourceAwsServiceDiscoveryServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sdconn

	input := &servicediscovery.UpdateServiceInput{
		Id: aws.String(d.Id()),
	}

	sc := &servicediscovery.ServiceChange{
		DnsConfig: expandServiceDiscoveryDnsConfigChange(d.Get("dns_config").([]interface{})[0].(map[string]interface{})),
	}

	if d.HasChange("description") {
		sc.Description = aws.String(d.Get("description").(string))
	}
	if d.HasChange("health_check_config") {
		hcconfig := d.Get("health_check_config").([]interface{})
		sc.HealthCheckConfig = expandServiceDiscoveryHealthCheckConfig(hcconfig[0].(map[string]interface{}))
	}

	input.Service = sc

	resp, err := conn.UpdateService(input)
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{servicediscovery.OperationStatusSubmitted, servicediscovery.OperationStatusPending},
		Target:     []string{servicediscovery.OperationStatusSuccess},
		Refresh:    servicediscoveryOperationRefreshStatusFunc(conn, *resp.OperationId),
		Timeout:    5 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsServiceDiscoveryServiceRead(d, meta)
}

func resourceAwsServiceDiscoveryServiceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sdconn

	input := &servicediscovery.DeleteServiceInput{
		Id: aws.String(d.Id()),
	}

	_, err := conn.DeleteService(input)
	if err != nil {
		return err
	}

	return nil
}

func expandServiceDiscoveryDnsConfig(configured map[string]interface{}) *servicediscovery.DnsConfig {
	result := &servicediscovery.DnsConfig{}

	result.NamespaceId = aws.String(configured["namespace_id"].(string))
	dnsRecords := configured["dns_records"].([]interface{})
	drs := make([]*servicediscovery.DnsRecord, len(dnsRecords))
	for i := range drs {
		raw := dnsRecords[i].(map[string]interface{})
		dr := &servicediscovery.DnsRecord{
			TTL:  aws.Int64(int64(raw["ttl"].(int))),
			Type: aws.String(raw["type"].(string)),
		}
		drs[i] = dr
	}
	result.DnsRecords = drs

	return result
}

func flattenServiceDiscoveryDnsConfig(config *servicediscovery.DnsConfig) []map[string]interface{} {
	result := map[string]interface{}{}

	result["namespace_id"] = *config.NamespaceId
	drs := make([]map[string]interface{}, 0)
	for _, v := range config.DnsRecords {
		dr := map[string]interface{}{}
		dr["ttl"] = *v.TTL
		dr["type"] = *v.Type
		drs = append(drs, dr)
	}
	result["dns_records"] = drs

	return []map[string]interface{}{result}
}

func expandServiceDiscoveryDnsConfigChange(configured map[string]interface{}) *servicediscovery.DnsConfigChange {
	result := &servicediscovery.DnsConfigChange{}

	dnsRecords := configured["dns_records"].([]interface{})
	drs := make([]*servicediscovery.DnsRecord, len(dnsRecords))
	for i := range drs {
		raw := dnsRecords[i].(map[string]interface{})
		dr := &servicediscovery.DnsRecord{
			TTL:  aws.Int64(int64(raw["ttl"].(int))),
			Type: aws.String(raw["type"].(string)),
		}
		drs[i] = dr
	}
	result.DnsRecords = drs

	return result
}

func expandServiceDiscoveryHealthCheckConfig(configured map[string]interface{}) *servicediscovery.HealthCheckConfig {
	if len(configured) < 1 {
		return nil
	}
	result := &servicediscovery.HealthCheckConfig{}

	if v, ok := configured["failure_threshold"]; ok && v.(int) != 0 {
		result.FailureThreshold = aws.Int64(int64(v.(int)))
	}
	if v, ok := configured["resource_path"]; ok && v.(string) != "" {
		result.ResourcePath = aws.String(v.(string))
	}
	if v, ok := configured["type"]; ok && v.(string) != "" {
		result.Type = aws.String(v.(string))
	}

	return result
}

func flattenServiceDiscoveryHealthCheckConfig(config *servicediscovery.HealthCheckConfig) []map[string]interface{} {
	if config == nil {
		return nil
	}
	result := map[string]interface{}{}

	if config.FailureThreshold != nil {
		result["failure_threshold"] = *config.FailureThreshold
	}
	if config.ResourcePath != nil {
		result["resource_path"] = *config.ResourcePath
	}
	if config.Type != nil {
		result["type"] = *config.Type
	}

	if len(result) < 1 {
		return nil
	}

	return []map[string]interface{}{result}
}
