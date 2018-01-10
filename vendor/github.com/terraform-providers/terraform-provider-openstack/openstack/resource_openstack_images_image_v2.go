package openstack

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceImagesImageV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceImagesImageV2Create,
		Read:   resourceImagesImageV2Read,
		Update: resourceImagesImageV2Update,
		Delete: resourceImagesImageV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"checksum": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"container_format": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: resourceImagesImageV2ValidateContainerFormat,
			},

			"created_at": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"disk_format": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: resourceImagesImageV2ValidateDiskFormat,
			},

			"file": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"image_cache_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  fmt.Sprintf("%s/.terraform/image_cache", os.Getenv("HOME")),
			},

			"image_source_url": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"local_file_path"},
			},

			"local_file_path": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"image_source_url"},
			},

			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},

			"min_disk_gb": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validatePositiveInt,
				Default:      0,
			},

			"min_ram_mb": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validatePositiveInt,
				Default:      0,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"owner": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"protected": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},

			"schema": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"size_bytes": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"update_at": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"visibility": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     false,
				ValidateFunc: resourceImagesImageV2ValidateVisibility,
				Default:      "private",
			},
		},
	}
}

func resourceImagesImageV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	imageClient, err := config.imageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack image client: %s", err)
	}

	protected := d.Get("protected").(bool)
	visibility := resourceImagesImageV2VisibilityFromString(d.Get("visibility").(string))
	createOpts := &images.CreateOpts{
		Name:            d.Get("name").(string),
		ContainerFormat: d.Get("container_format").(string),
		DiskFormat:      d.Get("disk_format").(string),
		MinDisk:         d.Get("min_disk_gb").(int),
		MinRAM:          d.Get("min_ram_mb").(int),
		Protected:       &protected,
		Visibility:      &visibility,
	}

	if v, ok := d.GetOk("tags"); ok {
		tags := v.(*schema.Set).List()
		createOpts.Tags = resourceImagesImageV2BuildTags(tags)
	}

	d.Partial(true)

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	newImg, err := images.Create(imageClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating Image: %s", err)
	}

	d.SetId(newImg.ID)

	// downloading/getting image file props
	imgFilePath, err := resourceImagesImageV2File(d)
	if err != nil {
		return fmt.Errorf("Error opening file for Image: %s", err)

	}
	fileSize, fileChecksum, err := resourceImagesImageV2FileProps(imgFilePath)
	if err != nil {
		return fmt.Errorf("Error getting file props: %s", err)
	}

	// upload
	imgFile, err := os.Open(imgFilePath)
	if err != nil {
		return fmt.Errorf("Error opening file %q: %s", imgFilePath, err)
	}
	defer imgFile.Close()
	log.Printf("[WARN] Uploading image %s (%d bytes). This can be pretty long.", d.Id(), fileSize)

	res := imagedata.Upload(imageClient, d.Id(), imgFile)
	if res.Err != nil {
		return fmt.Errorf("Error while uploading file %q: %s", imgFilePath, res.Err)
	}

	//wait for active
	stateConf := &resource.StateChangeConf{
		Pending:    []string{string(images.ImageStatusQueued), string(images.ImageStatusSaving)},
		Target:     []string{string(images.ImageStatusActive)},
		Refresh:    resourceImagesImageV2RefreshFunc(imageClient, d.Id(), fileSize, fileChecksum),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	if _, err = stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Image: %s", err)
	}

	d.Partial(false)

	return resourceImagesImageV2Read(d, meta)
}

func resourceImagesImageV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	imageClient, err := config.imageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack image client: %s", err)
	}

	img, err := images.Get(imageClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "image")
	}

	log.Printf("[DEBUG] Retrieved Image %s: %#v", d.Id(), img)

	d.Set("owner", img.Owner)
	d.Set("status", img.Status)
	d.Set("file", img.File)
	d.Set("schema", img.Schema)
	d.Set("checksum", img.Checksum)
	d.Set("size_bytes", img.SizeBytes)
	d.Set("metadata", img.Metadata)
	d.Set("created_at", img.CreatedAt)
	d.Set("update_at", img.UpdatedAt)
	d.Set("container_format", img.ContainerFormat)
	d.Set("disk_format", img.DiskFormat)
	d.Set("min_disk_gb", img.MinDiskGigabytes)
	d.Set("min_ram_mb", img.MinRAMMegabytes)
	d.Set("file", img.File)
	d.Set("name", img.Name)
	d.Set("protected", img.Protected)
	d.Set("size_bytes", img.SizeBytes)
	d.Set("tags", img.Tags)
	d.Set("visibility", img.Visibility)
	d.Set("region", GetRegion(d, config))

	return nil
}

func resourceImagesImageV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	imageClient, err := config.imageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack image client: %s", err)
	}

	updateOpts := make(images.UpdateOpts, 0)

	if d.HasChange("visibility") {
		visibility := resourceImagesImageV2VisibilityFromString(d.Get("visibility").(string))
		v := images.UpdateVisibility{Visibility: visibility}
		updateOpts = append(updateOpts, v)
	}

	if d.HasChange("name") {
		v := images.ReplaceImageName{NewName: d.Get("name").(string)}
		updateOpts = append(updateOpts, v)
	}

	if d.HasChange("tags") {
		tags := d.Get("tags").(*schema.Set).List()
		v := images.ReplaceImageTags{
			NewTags: resourceImagesImageV2BuildTags(tags),
		}
		updateOpts = append(updateOpts, v)
	}

	log.Printf("[DEBUG] Update Options: %#v", updateOpts)

	_, err = images.Update(imageClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating image: %s", err)
	}

	return resourceImagesImageV2Read(d, meta)
}

func resourceImagesImageV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	imageClient, err := config.imageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack image client: %s", err)
	}

	log.Printf("[DEBUG] Deleting Image %s", d.Id())
	if err := images.Delete(imageClient, d.Id()).Err; err != nil {
		return fmt.Errorf("Error deleting Image: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceImagesImageV2ValidateVisibility(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	validVisibilities := []string{
		"public",
		"private",
		"shared",
		"community",
	}

	for _, v := range validVisibilities {
		if value == v {
			return
		}
	}

	err := fmt.Errorf("%s must be one of %s", k, validVisibilities)
	errors = append(errors, err)
	return
}

func validatePositiveInt(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value > 0 {
		return
	}
	errors = append(errors, fmt.Errorf("%q must be a positive integer", k))
	return
}

var DiskFormats = [9]string{"ami", "ari", "aki", "vhd", "vmdk", "raw", "qcow2", "vdi", "iso"}

func resourceImagesImageV2ValidateDiskFormat(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	for i := range DiskFormats {
		if value == DiskFormats[i] {
			return
		}
	}
	errors = append(errors, fmt.Errorf("%q must be one of %v", k, DiskFormats))
	return
}

var ContainerFormats = [9]string{"ami", "ari", "aki", "bare", "ovf"}

func resourceImagesImageV2ValidateContainerFormat(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	for i := range ContainerFormats {
		if value == ContainerFormats[i] {
			return
		}
	}
	errors = append(errors, fmt.Errorf("%q must be one of %v", k, ContainerFormats))
	return
}

func resourceImagesImageV2VisibilityFromString(v string) images.ImageVisibility {
	switch v {
	case "public":
		return images.ImageVisibilityPublic
	case "private":
		return images.ImageVisibilityPrivate
	case "shared":
		return images.ImageVisibilityShared
	case "community":
		return images.ImageVisibilityCommunity
	}

	return ""
}

func fileMD5Checksum(f *os.File) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func resourceImagesImageV2FileProps(filename string) (int64, string, error) {
	var filesize int64
	var filechecksum string

	file, err := os.Open(filename)
	if err != nil {
		return -1, "", fmt.Errorf("Error opening file for Image: %s", err)

	}
	defer file.Close()

	fstat, err := file.Stat()
	if err != nil {
		return -1, "", fmt.Errorf("Error reading image file %q: %s", file.Name(), err)
	}

	filesize = fstat.Size()
	filechecksum, err = fileMD5Checksum(file)

	if err != nil {
		return -1, "", fmt.Errorf("Error computing image file %q checksum: %s", file.Name(), err)
	}

	return filesize, filechecksum, nil
}

func resourceImagesImageV2File(d *schema.ResourceData) (string, error) {
	if filename := d.Get("local_file_path").(string); filename != "" {
		return filename, nil
	} else if furl := d.Get("image_source_url").(string); furl != "" {
		dir := d.Get("image_cache_path").(string)
		os.MkdirAll(dir, 0700)
		filename := filepath.Join(dir, fmt.Sprintf("%x.img", md5.Sum([]byte(furl))))

		if _, err := os.Stat(filename); err != nil {
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("Error while trying to access file %q: %s", filename, err)
			}
			log.Printf("[DEBUG] File doens't exists %s. will download from %s", filename, furl)
			file, err := os.Create(filename)
			if err != nil {
				return "", fmt.Errorf("Error creating file %q: %s", filename, err)
			}
			defer file.Close()
			resp, err := http.Get(furl)
			if err != nil {
				return "", fmt.Errorf("Error downloading image from %q", furl)
			}
			defer resp.Body.Close()

			if _, err = io.Copy(file, resp.Body); err != nil {
				return "", fmt.Errorf("Error downloading image %q to file %q: %s", furl, filename, err)
			}
			return filename, nil
		} else {
			log.Printf("[DEBUG] File exists %s", filename)
			return filename, nil
		}
	} else {
		return "", fmt.Errorf("Error in config. no file specified")
	}
}

func resourceImagesImageV2RefreshFunc(client *gophercloud.ServiceClient, id string, fileSize int64, checksum string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		img, err := images.Get(client, id).Extract()
		if err != nil {
			return nil, "", err
		}
		log.Printf("[DEBUG] OpenStack image status is: %s", img.Status)

		if img.Checksum != checksum || int64(img.SizeBytes) != fileSize {
			return img, fmt.Sprintf("%s", img.Status), fmt.Errorf("Error wrong size %v or checksum %q", img.SizeBytes, img.Checksum)
		}

		return img, fmt.Sprintf("%s", img.Status), nil
	}
}

func resourceImagesImageV2BuildTags(v []interface{}) []string {
	var tags []string
	for _, tag := range v {
		tags = append(tags, tag.(string))
	}

	return tags
}
