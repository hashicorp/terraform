package azurerm

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"regexp"

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
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAzureRmDocumentDbName,
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
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
							ValidateFunc: validation.StringInSlice([]string{
								string(documentdb.BoundedStaleness),
								string(documentdb.Eventual),
								string(documentdb.Session),
								string(documentdb.Strong),
							}, true),
						},

						"max_interval_in_seconds": {
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.IntBetween(1, 100),
						},

						"max_staleness_prefix": {
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.IntBetween(1, 2147483647),
						},
					},
				},
				Set: resourceAzureRMDocumentDbConsistencyPolicyHash,
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

	_, error := client.CreateOrUpdate(resGroup, name, parameters, make(chan struct{}))
	err = <-error
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

	d.Set("name", resp.Name)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))
	d.Set("resource_group_name", resGroup)
	d.Set("offer_type", string(resp.DatabaseAccountOfferType))
	flattenAndSetAzureRmDocumentDbConsistencyPolicy(d, resp.ConsistencyPolicy)
	flattenAndSetAzureRmDocumentDbFailoverPolicy(d, resp.FailoverPolicies)

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

	deleteResp, error := client.Delete(resGroup, name, make(chan struct{}))
	resp := <-deleteResp
	err = <-error

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error issuing AzureRM delete request for DocumentDB instance '%s': %s", name, err)
	}

	if err != nil {
		return err
	}

	return nil
}

func expandAzureRmDocumentDbConsistencyPolicy(d *schema.ResourceData) documentdb.ConsistencyPolicy {
	inputs := d.Get("consistency_policy").(*schema.Set).List()
	input := inputs[0].(map[string]interface{})

	consistencyLevel := input["consistency_level"].(string)
	maxStalenessPrefix := int64(input["max_staleness_prefix"].(int))
	maxIntervalInSeconds := int32(input["max_interval_in_seconds"].(int))

	policy := documentdb.ConsistencyPolicy{
		DefaultConsistencyLevel: documentdb.DefaultConsistencyLevel(consistencyLevel),
		MaxIntervalInSeconds:    &maxIntervalInSeconds,
		MaxStalenessPrefix:      &maxStalenessPrefix,
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

	// all priorities must be unique
	locationIds := make(map[int]struct{}, len(locations))
	for _, location := range locations {
		priority := int(*location.FailoverPriority)
		if _, ok := locationIds[priority]; ok {
			err := fmt.Errorf("Each DocumentDB Failover Policy needs to be unique")
			return nil, err
		}

		locationIds[priority] = struct{}{}
	}

	if !containsWriteLocation {
		err := fmt.Errorf("DocumentDB Failover Policy should contain a Write Location (Location '0')")
		return nil, err
	}

	return locations, nil
}

func flattenAndSetAzureRmDocumentDbConsistencyPolicy(d *schema.ResourceData, policy *documentdb.ConsistencyPolicy) {
	results := schema.Set{
		F: resourceAzureRMDocumentDbConsistencyPolicyHash,
	}

	result := map[string]interface{}{}
	result["consistency_level"] = string(policy.DefaultConsistencyLevel)
	result["max_interval_in_seconds"] = int(*policy.MaxIntervalInSeconds)
	result["max_staleness_prefix"] = int(*policy.MaxStalenessPrefix)
	results.Add(result)

	d.Set("consistency_policy", &results)
}

func flattenAndSetAzureRmDocumentDbFailoverPolicy(d *schema.ResourceData, list *[]documentdb.FailoverPolicy) {
	results := schema.Set{
		F: resourceAzureRMDocumentDbFailoverPolicyHash,
	}

	for _, i := range *list {
		result := map[string]interface{}{
			"id":       *i.ID,
			"location": azureRMNormalizeLocation(*i.LocationName),
			"priority": int(*i.FailoverPriority), // TODO: check we're parsing this out correctly
		}

		results.Add(result)
	}

	d.Set("failover_policy", &results)
}

func resourceAzureRMDocumentDbConsistencyPolicyHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	consistencyLevel := m["consistency_level"].(string)
	maxInterval := m["max_interval_in_seconds"].(int)
	maxStalenessPrefix := m["max_staleness_prefix"].(int)

	buf.WriteString(fmt.Sprintf("%s-%d-%d", consistencyLevel, maxInterval, maxStalenessPrefix))

	return hashcode.String(buf.String())
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

func validateAzureRmDocumentDbName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	r, _ := regexp.Compile("[a-z0-9-]")
	if !r.MatchString(value) {
		errors = append(errors, fmt.Errorf("DocumentDB Name can only contain lower-case characters, numbers and the `-` character."))
	}

	length := len(value)
	if length > 50 || 3 > length {
		errors = append(errors, fmt.Errorf("DocumentDB Name can only be between 3 and 50 seconds."))
	}

	return
}