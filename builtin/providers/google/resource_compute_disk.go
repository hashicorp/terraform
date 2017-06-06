package google

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

const (
	computeDiskUserRegexString = "^(?:https://www.googleapis.com/compute/v1/projects/)?([-_a-zA-Z0-9]*)/zones/([-_a-zA-Z0-9]*)/instances/([-_a-zA-Z0-9]*)$"
)

var (
	computeDiskUserRegex = regexp.MustCompile(computeDiskUserRegexString)
)

func resourceComputeDisk() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeDiskCreate,
		Read:   resourceComputeDiskRead,
		Update: resourceComputeDiskUpdate,
		Delete: resourceComputeDiskDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"disk_encryption_key_raw": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				ForceNew:  true,
				Sensitive: true,
			},

			"disk_encryption_key_sha256": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"image": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"snapshot": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"users": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceComputeDiskCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Get the zone
	log.Printf("[DEBUG] Loading zone: %s", d.Get("zone").(string))
	zone, err := config.clientCompute.Zones.Get(
		project, d.Get("zone").(string)).Do()
	if err != nil {
		return fmt.Errorf(
			"Error loading zone '%s': %s", d.Get("zone").(string), err)
	}

	// Build the disk parameter
	disk := &compute.Disk{
		Name:   d.Get("name").(string),
		SizeGb: int64(d.Get("size").(int)),
	}

	// If we were given a source image, load that.
	if v, ok := d.GetOk("image"); ok {
		log.Printf("[DEBUG] Resolving image name: %s", v.(string))
		imageUrl, err := resolveImage(config, v.(string))
		if err != nil {
			return fmt.Errorf(
				"Error resolving image name '%s': %s",
				v.(string), err)
		}

		disk.SourceImage = imageUrl
		log.Printf("[DEBUG] Image name resolved to: %s", imageUrl)
	}

	if v, ok := d.GetOk("type"); ok {
		log.Printf("[DEBUG] Loading disk type: %s", v.(string))
		diskType, err := readDiskType(config, zone, v.(string))
		if err != nil {
			return fmt.Errorf(
				"Error loading disk type '%s': %s",
				v.(string), err)
		}

		disk.Type = diskType.SelfLink
	}

	if v, ok := d.GetOk("snapshot"); ok {
		snapshotName := v.(string)
		match, _ := regexp.MatchString("^https://www.googleapis.com/compute", snapshotName)
		if match {
			disk.SourceSnapshot = snapshotName
		} else {
			log.Printf("[DEBUG] Loading snapshot: %s", snapshotName)
			snapshotData, err := config.clientCompute.Snapshots.Get(
				project, snapshotName).Do()

			if err != nil {
				return fmt.Errorf(
					"Error loading snapshot '%s': %s",
					snapshotName, err)
			}
			disk.SourceSnapshot = snapshotData.SelfLink
		}
	}

	if v, ok := d.GetOk("disk_encryption_key_raw"); ok {
		disk.DiskEncryptionKey = &compute.CustomerEncryptionKey{}
		disk.DiskEncryptionKey.RawKey = v.(string)
	}

	op, err := config.clientCompute.Disks.Insert(
		project, d.Get("zone").(string), disk).Do()
	if err != nil {
		return fmt.Errorf("Error creating disk: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(disk.Name)

	err = computeOperationWaitZone(config, op, project, d.Get("zone").(string), "Creating Disk")
	if err != nil {
		return err
	}
	return resourceComputeDiskRead(d, meta)
}

func resourceComputeDiskUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	if d.HasChange("size") {
		rb := &compute.DisksResizeRequest{
			SizeGb: int64(d.Get("size").(int)),
		}
		_, err := config.clientCompute.Disks.Resize(
			project, d.Get("zone").(string), d.Id(), rb).Do()
		if err != nil {
			return fmt.Errorf("Error resizing disk: %s", err)
		}
	}

	return resourceComputeDiskRead(d, meta)
}

func resourceComputeDiskRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	getDisk := func(zone string) (interface{}, error) {
		return config.clientCompute.Disks.Get(project, zone, d.Id()).Do()
	}

	var disk *compute.Disk
	if zone, ok := d.GetOk("zone"); ok {
		disk, err = config.clientCompute.Disks.Get(
			project, zone.(string), d.Id()).Do()
		if err != nil {
			return handleNotFoundError(err, d, fmt.Sprintf("Disk %q", d.Get("name").(string)))
		}
	} else {
		// If the resource was imported, the only info we have is the ID. Try to find the resource
		// by searching in the region of the project.
		var resource interface{}
		resource, err = getZonalResourceFromRegion(getDisk, region, config.clientCompute, project)

		if err != nil {
			return err
		}

		disk = resource.(*compute.Disk)
	}

	zoneUrlParts := strings.Split(disk.Zone, "/")
	typeUrlParts := strings.Split(disk.Type, "/")
	d.Set("name", disk.Name)
	d.Set("self_link", disk.SelfLink)
	d.Set("type", typeUrlParts[len(typeUrlParts)-1])
	d.Set("zone", zoneUrlParts[len(zoneUrlParts)-1])
	d.Set("size", disk.SizeGb)
	d.Set("users", disk.Users)
	if disk.DiskEncryptionKey != nil && disk.DiskEncryptionKey.Sha256 != "" {
		d.Set("disk_encryption_key_sha256", disk.DiskEncryptionKey.Sha256)
	}
	if disk.SourceImage != "" {
		imageUrlParts := strings.Split(disk.SourceImage, "/")
		d.Set("image", imageUrlParts[len(imageUrlParts)-1])
	}
	d.Set("snapshot", disk.SourceSnapshot)

	return nil
}

func resourceComputeDiskDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// if disks are attached, they must be detached before the disk can be deleted
	if instances, ok := d.Get("users").([]interface{}); ok {
		type detachArgs struct{ project, zone, instance, deviceName string }
		var detachCalls []detachArgs
		self := d.Get("self_link").(string)
		for _, instance := range instances {
			if !computeDiskUserRegex.MatchString(instance.(string)) {
				return fmt.Errorf("Unknown user %q of disk %q", instance, self)
			}
			matches := computeDiskUserRegex.FindStringSubmatch(instance.(string))
			instanceProject := matches[1]
			instanceZone := matches[2]
			instanceName := matches[3]
			i, err := config.clientCompute.Instances.Get(instanceProject, instanceZone, instanceName).Do()
			if err != nil {
				if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
					log.Printf("[WARN] instance %q not found, not bothering to detach disks", instance.(string))
					continue
				}
				return fmt.Errorf("Error retrieving instance %s: %s", instance.(string), err.Error())
			}
			for _, disk := range i.Disks {
				if disk.Source == self {
					detachCalls = append(detachCalls, detachArgs{
						project:    project,
						zone:       i.Zone,
						instance:   i.Name,
						deviceName: disk.DeviceName,
					})
				}
			}
		}
		for _, call := range detachCalls {
			op, err := config.clientCompute.Instances.DetachDisk(call.project, call.zone, call.instance, call.deviceName).Do()
			if err != nil {
				return fmt.Errorf("Error detaching disk %s from instance %s/%s/%s: %s", call.deviceName, call.project,
					call.zone, call.instance, err.Error())
			}
			err = computeOperationWaitZone(config, op, call.project, call.zone,
				fmt.Sprintf("Detaching disk from %s/%s/%s", call.project, call.zone, call.instance))
			if err != nil {
				return err
			}
		}
	}

	// Delete the disk
	op, err := config.clientCompute.Disks.Delete(
		project, d.Get("zone").(string), d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Disk %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error deleting disk: %s", err)
	}

	zone := d.Get("zone").(string)
	err = computeOperationWaitZone(config, op, project, zone, "Deleting Disk")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
