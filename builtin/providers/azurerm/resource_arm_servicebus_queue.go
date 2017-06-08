package azurerm

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/servicebus"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmServiceBusQueue() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmServiceBusQueueCreate,
		Read:   resourceArmServiceBusQueueRead,
		Update: resourceArmServiceBusQueueCreate,
		Delete: resourceArmServiceBusQueueDelete,
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

			"location": locationSchema(),

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
				Default:  false,
				Optional: true,
			},

			"enable_express": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},

			"enable_partitioning": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
				ForceNew: true,
			},

			"max_size_in_megabytes": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"requires_duplicate_detection": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
				ForceNew: true,
			},

			"support_ordering": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},
		},
	}
}

func resourceArmServiceBusQueueCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).serviceBusQueuesClient
	log.Printf("[INFO] preparing arguments for Azure ARM ServiceBus Queue creation.")

	name := d.Get("name").(string)
	namespaceName := d.Get("namespace_name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)

	parameters := servicebus.QueueCreateOrUpdateParameters{
		Name:            &name,
		Location:        &location,
		QueueProperties: &servicebus.QueueProperties{},
	}

	if autoDeleteOnIdle := d.Get("auto_delete_on_idle").(string); autoDeleteOnIdle != "" {
		parameters.QueueProperties.AutoDeleteOnIdle = &autoDeleteOnIdle
	}

	if defaultTTL := d.Get("default_message_ttl").(string); defaultTTL != "" {
		parameters.QueueProperties.DefaultMessageTimeToLive = &defaultTTL
	}

	if duplicateWindow := d.Get("duplicate_detection_history_time_window").(string); duplicateWindow != "" {
		parameters.QueueProperties.DuplicateDetectionHistoryTimeWindow = &duplicateWindow
	}

	// We need to retrieve the namespace because Premium namespace works differently from Basic and Standard,
	// so it needs different rules applied to it.
	namespace, nsErr := meta.(*ArmClient).serviceBusNamespacesClient.Get(resGroup, namespaceName)
	if nsErr != nil {
		return nsErr
	}

	// Enforce Premium namespace to have partitioning enabled in Terraform. It is always enabled in Azure for
	// Premium SKU.
	if namespace.Sku.Name == servicebus.Premium && !d.Get("enable_partitioning").(bool) {
		return fmt.Errorf("ServiceBus Queue (%s) must have Partitioning enabled for Premium SKU", name)
	}

	// Enforce Premium namespace to have Express Entities disabled in Terraform since they are not supported for
	// Premium SKU.
	if namespace.Sku.Name == servicebus.Premium && d.Get("enable_express").(bool) {
		return fmt.Errorf("ServiceBus Queue (%s) does not support Express Entities in Premium SKU and must be disabled", name)
	}

	enableBatchedOps := d.Get("enable_batched_operations").(bool)
	enableExpress := d.Get("enable_express").(bool)
	enablePartitioning := d.Get("enable_partitioning").(bool)
	maxSize := int64(d.Get("max_size_in_megabytes").(int))
	requiresDuplicateDetection := d.Get("requires_duplicate_detection").(bool)
	supportOrdering := d.Get("support_ordering").(bool)

	parameters.QueueProperties.EnableBatchedOperations = &enableBatchedOps
	parameters.QueueProperties.EnableExpress = &enableExpress
	parameters.QueueProperties.EnablePartitioning = &enablePartitioning
	parameters.QueueProperties.MaxSizeInMegabytes = &maxSize
	parameters.QueueProperties.RequiresDuplicateDetection = &requiresDuplicateDetection
	parameters.QueueProperties.SupportOrdering = &supportOrdering

	_, err := client.CreateOrUpdate(resGroup, namespaceName, name, parameters)
	if err != nil {
		return err
	}

	read, err := client.Get(resGroup, namespaceName, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read ServiceBus Queue %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmServiceBusQueueRead(d, meta)
}

func resourceArmServiceBusQueueRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).serviceBusQueuesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	namespaceName := id.Path["namespaces"]
	name := id.Path["queues"]

	resp, err := client.Get(resGroup, namespaceName, name)
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure ServiceBus Queue %s: %s", name, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("namespace_name", namespaceName)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))

	if resp.QueueProperties == nil {
		return fmt.Errorf("Missing QueueProperties in response for Azure ServiceBus Queue %s: %s", name, err)
	}

	props := resp.QueueProperties
	d.Set("auto_delete_on_idle", props.AutoDeleteOnIdle)
	d.Set("default_message_ttl", props.DefaultMessageTimeToLive)
	d.Set("duplicate_detection_history_time_window", props.DuplicateDetectionHistoryTimeWindow)

	d.Set("enable_batched_operations", props.EnableBatchedOperations)
	d.Set("enable_express", props.EnableExpress)
	d.Set("enable_partitioning", props.EnablePartitioning)
	d.Set("requires_duplicate_detection", props.RequiresDuplicateDetection)
	d.Set("support_ordering", props.SupportOrdering)

	maxSize := int(*props.MaxSizeInMegabytes)

	// If the queue is NOT in a premium namespace (ie. it is Basic or Standard) and partitioning is enabled
	// then the max size returned by the API will be 16 times greater than the value set.
	if *props.EnablePartitioning {
		namespace, err := meta.(*ArmClient).serviceBusNamespacesClient.Get(resGroup, namespaceName)
		if err != nil {
			return err
		}

		if namespace.Sku.Name != servicebus.Premium {
			const partitionCount = 16
			maxSize = int(*props.MaxSizeInMegabytes / partitionCount)
		}
	}

	d.Set("max_size_in_megabytes", maxSize)

	return nil
}

func resourceArmServiceBusQueueDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).serviceBusQueuesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	namespaceName := id.Path["namespaces"]
	name := id.Path["queues"]

	_, err = client.Delete(resGroup, namespaceName, name)

	return err
}
