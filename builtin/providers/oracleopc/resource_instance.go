package opc

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/oracle/terraform-provider-compute/sdk/compute"
	"log"
)

func resourceInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceInstanceCreate,
		Read:   resourceInstanceRead,
		Delete: resourceInstanceDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"shape": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"imageList": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"label": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"ip": {
				Type:     schema.TypeString,
				Optional: false,
				Computed: true,
			},

			"opcId": {
				Type:     schema.TypeString,
				Optional: false,
				Computed: true,
			},

			"sshKeys": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"attributes": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"vcable": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"storage": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"index": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
						"volume": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"bootOrder": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeInt},
			},
		},
	}
}

func getAttrs(d *schema.ResourceData) (*map[string]interface{}, error) {
	var attrs map[string]interface{}

	attrString := d.Get("attributes").(string)
	if attrString == "" {
		return &attrs, nil
	}
	if err := json.Unmarshal([]byte(attrString), &attrs); err != nil {
		return &attrs, fmt.Errorf("Cannot parse '%s' as json", attrString)
	}
	return &attrs, nil
}

func resourceInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource data: %#v", d.State())

	client := meta.(*OPCClient).Instances()
	name := d.Get("name").(string)
	shape := d.Get("shape").(string)
	imageList := d.Get("imageList").(string)
	label := d.Get("label").(string)
	storage := getStorageAttachments(d)
	sshKeys := getSSHKeys(d)
	bootOrder := getBootOrder(d)

	attrs, err := getAttrs(d)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating instance with name %s, shape %s, imageList %s, storage %s, bootOrder %s, label %s, sshKeys %s, attrs %#v",
		name, shape, imageList, storage, bootOrder, label, sshKeys, attrs)

	id, err := client.LaunchInstance(name, label, shape, imageList, storage, bootOrder, sshKeys, *attrs)
	if err != nil {
		return fmt.Errorf("Error creating instance %s: %s", name, err)
	}

	log.Printf("[DEBUG] Waiting for instance %s to come online", id.String())
	info, err := client.WaitForInstanceRunning(id, meta.(*OPCClient).MaxRetryTimeout)
	if err != nil {
		return fmt.Errorf("Error waiting for instance %s to come online: %s", id, err)
	}

	log.Printf("[DEBUG] Created instance %s: %#v", id, info)

	attachStorage(
		&compute.InstanceName{
			Name: info.Name,
			ID:   info.ID,
		},
		d, meta)

	d.SetId(info.Name)
	updateInstanceResourceData(d, info)
	return nil
}

func attachStorage(name *compute.InstanceName, d *schema.ResourceData, meta interface{}) error {
	storageClient := meta.(*OPCClient).StorageAttachments()
	storage := d.Get("storage").(*schema.Set)
	updatedStorage := schema.NewSet(storage.F, []interface{}{})

	for _, i := range storage.List() {
		attrs := i.(map[string]interface{})
		attachmentInfo, err := storageClient.CreateStorageAttachment(
			attrs["index"].(int),
			name,
			attrs["volume"].(string))

		if err != nil {
			return err
		}

		log.Printf("[DEBUG] Waiting for storage attachment %#v to come online", attachmentInfo)
		storageClient.WaitForStorageAttachmentCreated(attachmentInfo.Name, meta.(*OPCClient).MaxRetryTimeout)
		log.Printf("[DEBUG] Storage attachment %s: %s-%s created",
			attachmentInfo.Name, attachmentInfo.InstanceName, attachmentInfo.StorageVolumeName)
		attrs["name"] = attachmentInfo.Name
		updatedStorage.Add(attrs)
	}

	d.Set("storage", updatedStorage)
	return nil
}

func getSSHKeys(d *schema.ResourceData) []string {
	sshKeys := []string{}
	for _, i := range d.Get("sshKeys").([]interface{}) {
		sshKeys = append(sshKeys, i.(string))
	}
	return sshKeys
}

func getBootOrder(d *schema.ResourceData) []int {
	bootOrder := []int{}
	for _, i := range d.Get("bootOrder").([]interface{}) {
		bootOrder = append(bootOrder, i.(int))
	}
	return bootOrder
}

func getStorageAttachments(d *schema.ResourceData) []compute.LaunchPlanStorageAttachmentSpec {
	storageAttachments := []compute.LaunchPlanStorageAttachmentSpec{}
	storage := d.Get("storage").(*schema.Set)
	for _, i := range storage.List() {
		attrs := i.(map[string]interface{})
		storageAttachments = append(storageAttachments, compute.LaunchPlanStorageAttachmentSpec{
			Index:  attrs["index"].(int),
			Volume: attrs["volume"].(string),
		})
	}
	return storageAttachments
}

func updateInstanceResourceData(d *schema.ResourceData, info *compute.InstanceInfo) error {
	d.Set("name", info.Name)
	d.Set("opcId", info.ID)
	d.Set("imageList", info.ImageList)
	d.Set("bootOrder", info.BootOrder)
	d.Set("sshKeys", info.SSHKeys)
	d.Set("label", info.Label)
	d.Set("ip", info.IPAddress)
	d.Set("vcable", info.VCableID)

	return nil
}

func resourceInstanceRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource data: %#v", d.State())
	client := meta.(*OPCClient).Instances()
	name := d.Get("name").(string)
	instanceName := &compute.InstanceName{
		Name: name,
		ID:   d.Get("opcId").(string),
	}

	log.Printf("[DEBUG] Reading state of instance %s", instanceName)
	result, err := client.GetInstance(instanceName)
	if err != nil {
		// Instance doesn't exist
		if compute.WasNotFoundError(err) {
			log.Printf("[DEBUG] Instance %s not found", instanceName)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading instance %s: %s", instanceName, err)
	}

	log.Printf("[DEBUG] Read state of instance %s: %#v", instanceName, result)

	attachments, err := meta.(*OPCClient).StorageAttachments().GetStorageAttachmentsForInstance(instanceName)
	if err != nil {
		return fmt.Errorf("Error reading storage attachments for instance %s: %s", instanceName, err)
	}
	updateInstanceResourceData(d, result)
	updateAttachmentResourceData(d, attachments)
	return nil
}

func updateAttachmentResourceData(d *schema.ResourceData, attachments *[]compute.StorageAttachmentInfo) {
	attachmentSet := schema.NewSet(d.Get("storage").(*schema.Set).F, []interface{}{})
	for _, attachment := range *attachments {
		properties := map[string]interface{}{
			"index":  attachment.Index,
			"volume": attachment.StorageVolumeName,
			"name":   attachment.Name,
		}
		attachmentSet.Add(properties)
	}
	d.Set("storage", attachmentSet)
}

func resourceInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource data: %#v", d.State())
	client := meta.(*OPCClient).Instances()
	name := d.Get("name").(string)

	instanceName := &compute.InstanceName{
		Name: name,
		ID:   d.Get("opcId").(string),
	}

	log.Printf("[DEBUG] Deleting instance %s", instanceName)
	if err := client.DeleteInstance(instanceName); err != nil {
		return fmt.Errorf("Error deleting instance %s: %s", instanceName, err)
	}
	if err := client.WaitForInstanceDeleted(instanceName, meta.(*OPCClient).MaxRetryTimeout); err != nil {
		return fmt.Errorf("Error deleting instance %s: %s", instanceName, err)
	}

	for _, attachment := range d.Get("storage").(*schema.Set).List() {
		name := attachment.(map[string]interface{})["name"].(string)
		log.Printf("[DEBUG] Deleting storage attachment %s", name)
		client.StorageAttachments().DeleteStorageAttachment(name)
		client.StorageAttachments().WaitForStorageAttachmentDeleted(name, meta.(*OPCClient).MaxRetryTimeout)
	}

	return nil
}
