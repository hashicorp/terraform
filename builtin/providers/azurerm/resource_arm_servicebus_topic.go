package azurerm

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/servicebus"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmServiceBusTopic() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmServiceBusTopicCreate,
		Read:   resourceArmServiceBusTopicRead,
		Update: resourceArmServiceBusTopicCreate,
		Delete: resourceArmServiceBusTopicDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"namespace_name": {
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

			"auto_delete_on_idle": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"default_message_ttl": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"duplicate_detection_history_time_window": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"enable_batched_operations": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"enable_express": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"enable_filtering_messages_before_publishing": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"enable_partitioning": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"max_size_in_megabytes": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateArmServiceBusTopicMaxSize,
			},

			"requires_duplicate_detection": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"support_ordering": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceArmServiceBusTopicCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).serviceBusTopicsClient
	log.Printf("[INFO] preparing arguments for Azure ARM ServiceBus Topic creation.")

	name := d.Get("name").(string)
	namespaceName := d.Get("namespace_name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)

	parameters := servicebus.TopicCreateOrUpdateParameters{
		Name:       &name,
		Location:   &location,
		Properties: &servicebus.TopicProperties{},
	}

	if autoDeleteOnIdle := d.Get("auto_delete_on_idle").(string); autoDeleteOnIdle != "" {
		parameters.Properties.AutoDeleteOnIdle = &autoDeleteOnIdle
	}

	if defaultTTL := d.Get("default_message_ttl").(string); defaultTTL != "" {
		parameters.Properties.DefaultMessageTimeToLive = &defaultTTL
	}

	if duplicateWindow := d.Get("duplicate_detection_history_time_window").(string); duplicateWindow != "" {
		parameters.Properties.DuplicateDetectionHistoryTimeWindow = &duplicateWindow
	}

	enableBatchedOps := d.Get("enable_batched_operations").(bool)
	enableExpress := d.Get("enable_express").(bool)
	enableFiltering := d.Get("enable_filtering_messages_before_publishing").(bool)
	enablePartitioning := d.Get("enable_partitioning").(bool)
	maxSize := int64(d.Get("max_size_in_megabytes").(int))
	requiresDuplicateDetection := d.Get("requires_duplicate_detection").(bool)
	supportOrdering := d.Get("support_ordering").(bool)

	parameters.Properties.EnableBatchedOperations = &enableBatchedOps
	parameters.Properties.EnableExpress = &enableExpress
	parameters.Properties.FilteringMessagesBeforePublishing = &enableFiltering
	parameters.Properties.EnablePartitioning = &enablePartitioning
	parameters.Properties.MaxSizeInMegabytes = &maxSize
	parameters.Properties.RequiresDuplicateDetection = &requiresDuplicateDetection
	parameters.Properties.SupportOrdering = &supportOrdering

	_, err := client.CreateOrUpdate(resGroup, namespaceName, name, parameters)
	if err != nil {
		return err
	}

	read, err := client.Get(resGroup, namespaceName, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read ServiceBus Topic %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmServiceBusTopicRead(d, meta)
}

func resourceArmServiceBusTopicRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).serviceBusTopicsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	namespaceName := id.Path["namespaces"]
	name := id.Path["topics"]

	resp, err := client.Get(resGroup, namespaceName, name)
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure ServiceBus Topic %s: %s", name, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("namespace_name", namespaceName)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))

	props := resp.Properties
	d.Set("auto_delete_on_idle", props.AutoDeleteOnIdle)
	d.Set("default_message_ttl", props.DefaultMessageTimeToLive)

	if props.DuplicateDetectionHistoryTimeWindow != nil && *props.DuplicateDetectionHistoryTimeWindow != "" {
		d.Set("duplicate_detection_history_time_window", props.DuplicateDetectionHistoryTimeWindow)
	}

	d.Set("enable_batched_operations", props.EnableBatchedOperations)
	d.Set("enable_express", props.EnableExpress)
	d.Set("enable_filtering_messages_before_publishing", props.FilteringMessagesBeforePublishing)
	d.Set("enable_partitioning", props.EnablePartitioning)
	d.Set("requires_duplicate_detection", props.RequiresDuplicateDetection)
	d.Set("support_ordering", props.SupportOrdering)

	// if partitioning is enabled then the max size returned by the API will be
	// 16 times greater than the value set by the user
	if *props.EnablePartitioning {
		const partitionCount = 16
		d.Set("max_size_in_megabytes", int(*props.MaxSizeInMegabytes/partitionCount))
	} else {
		d.Set("max_size_in_megabytes", int(*props.MaxSizeInMegabytes))
	}

	return nil
}

func resourceArmServiceBusTopicDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).serviceBusTopicsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	namespaceName := id.Path["namespaces"]
	name := id.Path["topics"]

	_, err = client.Delete(resGroup, namespaceName, name)

	return err
}

func validateArmServiceBusTopicMaxSize(i interface{}, k string) (s []string, es []error) {
	v := i.(int)
	if v%1024 != 0 || v < 0 || v > 5120 {
		es = append(es, fmt.Errorf("%q must be a multiple of 1024 up to and including 5120", k))
	}

	return
}
