package azurerm

import (
	"fmt"
	"log"
	"net/http"

	"strings"

	"bytes"

	"github.com/Azure/azure-sdk-for-go/arm/documentdb"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceArmDocumentDb() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmDocumentDBCreateUpdate,
		Read:   resourceArmDocumentDBRead,
		Update: resourceArmDocumentDBCreateUpdate,
		Delete: resourceArmDocumentDBDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"offer_type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(documentdb.Standard),
				}, true),
			},

			"consistency_policy": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"consistency_level": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(documentdb.BoundedStaleness),
								string(documentdb.Eventual),
								string(documentdb.Session),
								string(documentdb.Strong),
							}, true),
						},

						"max_interval_in_seconds": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validateAzureRmDocumentDbMaxIntervalInSeconds,
						},

						"max_staleness": {
							Type:     schema.TypeInt,
							Required: true,
							// TODO: validation
						},
					},
				},
			},

			"failover_policy": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"location": {
							Type:      schema.TypeString,
							Required:  true,
							StateFunc: azureRMNormalizeLocation,
						},

						"priority": {
							Type:     schema.TypeInt,
							Required: true,
							// TODO: validation
						},
					},
				},
				Set: resourceAzureRMDocumentDbFailoverPolicyHash,
			},

			"primary_master_key": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"secondary_master_key": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"primary_readonly_master_key": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"secondary_readonly_master_key": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmDocumentDBCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).documentDbClient
	log.Printf("[INFO] preparing arguments for Azure ARM Document DB creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	offerType := d.Get("offer_type").(string)

	consistencyPolicy := expandAzureRmDocumentDbConsistencyPolicy(d)
	failoverPolicies, err := expandAzureRmDocumentDbFailoverPolicies(name, d)
	if err != nil {
		return err
	}
	tags := d.Get("tags").(map[string]interface{})

	parameters := documentdb.DatabaseAccountCreateUpdateParameters{
		Location: &location,
		DatabaseAccountCreateUpdateProperties: &documentdb.DatabaseAccountCreateUpdateProperties{
			ConsistencyPolicy:        &consistencyPolicy,
			DatabaseAccountOfferType: &offerType,
			Locations:                &failoverPolicies,
		},
		Tags: expandTags(tags),
	}

	_, err = client.CreateOrUpdate(resGroup, name, parameters, make(chan struct{}))
	if err != nil {
		return err
	}

	read, err := client.Get(resGroup, name)
	if err != nil {
		return err
	}

	if read.ID == nil {
		return fmt.Errorf("Cannot read Document DB instance %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmDocumentDBRead(d, meta)
}

func resourceArmDocumentDBRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).documentDbClient
	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["databaseAccounts"]

	resp, err := client.Get(resGroup, name)
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure DocumentDB %s: %s", name, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	properties := resp.DatabaseAccountProperties

	d.Set("name", resp.Name)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))
	d.Set("resource_group_name", resGroup)
	d.Set("offer_type", string(properties.DatabaseAccountOfferType))
	d.Set("consistency_policy", flattenAzureRmDocumentDbConsistencyPolicy(properties.ConsistencyPolicy))
	d.Set("failover_policy", flattenAzureRmDocumentDbFailoverPolicy(properties.FailoverPolicies))

	keys, err := client.ListKeys(resGroup, name)
	if err != nil {
		log.Printf("[ERROR] Unable to List Write keys for DocumentDB %s: %s", name, err)
	} else {
		d.Set("primary_master_key", keys.PrimaryMasterKey)
		d.Set("secondary_master_key", keys.SecondaryMasterKey)
	}

	readonlyKeys, err := client.ListReadOnlyKeys(resGroup, name)
	if err != nil {
		log.Printf("[ERROR] Unable to List read-only keys for DocumentDB %s: %s", name, err)
	} else {
		d.Set("primary_readonly_master_key", readonlyKeys.PrimaryReadonlyMasterKey)
		d.Set("secondary_readonly_master_key", readonlyKeys.SecondaryReadonlyMasterKey)
	}

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmDocumentDBDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).documentDbClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["databaseAccounts"]

	resp, err := client.Delete(resGroup, name, make(chan struct{}))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error issuing AzureRM delete request for DocumentDB instance '%s': %s", name, err)
	}

	return nil
}

func expandAzureRmDocumentDbConsistencyPolicy(d *schema.ResourceData) documentdb.ConsistencyPolicy {
	inputs := d.Get("consistency_policy").(*schema.Set).List()
	input := inputs[0].(map[string]interface{})

	consistencyLevel := input["consistency_level"].(string)
	maxStaleness := int64(input["max_staleness"].(int))
	maxIntervalInSeconds := int32(input["max_interval_in_seconds"].(int))

	policy := documentdb.ConsistencyPolicy{
		DefaultConsistencyLevel: documentdb.DefaultConsistencyLevel(consistencyLevel),
		MaxIntervalInSeconds:    &maxIntervalInSeconds,
		MaxStalenessPrefix:      &maxStaleness,
	}

	return policy
}

func expandAzureRmDocumentDbFailoverPolicies(databaseName string, d *schema.ResourceData) ([]documentdb.Location, error) {
	input := d.Get("failover_policy").(*schema.Set).List()
	locations := make([]documentdb.Location, 0, len(input))

	for _, configRaw := range input {
		data := configRaw.(map[string]interface{})

		locationName := azureRMNormalizeLocation(data["location"].(string))
		id := fmt.Sprintf("%s-%s", databaseName, locationName)
		failoverPriority := int32(data["priority"].(int))

		location := documentdb.Location{
			ID:               &id,
			LocationName:     &locationName,
			FailoverPriority: &failoverPriority,
		}

		locations = append(locations, location)
	}

	containsWriteLocation := false
	writeFailoverPriority := int32(0)
	for _, location := range locations {
		if *location.FailoverPriority == writeFailoverPriority {
			containsWriteLocation = true
			break
		}
	}

	// TODO: all priorities must be unique

	if !containsWriteLocation {
		err := fmt.Errorf("DocumentDB Offer Type can only be 'Standard' or 'Premium'")
		return nil, err
	}

	return locations, nil
}

func flattenAzureRmDocumentDbConsistencyPolicy(policy *documentdb.ConsistencyPolicy) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)
	item := map[string]interface{}{
		"consistency_level":       string(policy.DefaultConsistencyLevel),
		"max_interval_in_seconds": policy.MaxIntervalInSeconds,
		"max_staleness":           policy.MaxStalenessPrefix,
	}
	result = append(result, item)
	return result
}

func flattenAzureRmDocumentDbFailoverPolicy(list *[]documentdb.FailoverPolicy) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(*list))
	for _, i := range *list {
		l := map[string]interface{}{
			"id":       *i.ID,
			"location": azureRMNormalizeLocation(*i.LocationName),
			"priority": *i.FailoverPriority, // TODO: check we're parsing this out correctly
		}

		result = append(result, l)
	}
	return result
}

func validateAzureRmDocumentDbMaxIntervalInSeconds(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)

	if value > 100 || 1 > value {
		errors = append(errors, fmt.Errorf("DocumentDB Max Interval In Seconds can only be between 1 and 100 seconds"))
	}

	return
}

func resourceAzureRMDocumentDbFailoverPolicyHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	locationName := m["location"].(string)
	location := azureRMNormalizeLocation(locationName)
	priority := int32(m["priority"].(int))

	buf.WriteString(fmt.Sprintf("%s-%d", location, priority))

	return hashcode.String(buf.String())
}
