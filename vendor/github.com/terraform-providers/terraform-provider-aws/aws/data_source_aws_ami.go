package aws

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func dataSourceAwsAmi() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsAmiRead,

		Schema: map[string]*schema.Schema{
			"filter": dataSourceFiltersSchema(),
			"executable_users": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"name_regex": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.ValidateRegexp,
			},
			"most_recent": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"owners": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			// Computed values.
			"architecture": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"creation_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hypervisor": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"image_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"image_location": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"image_owner_alias": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"image_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"kernel_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"platform": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"public": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"ramdisk_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"root_device_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"root_device_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"root_snapshot_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"sriov_net_support": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"virtualization_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			// Complex computed values
			"block_device_mappings": {
				Type:     schema.TypeSet,
				Computed: true,
				Set:      amiBlockDeviceMappingHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"no_device": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"virtual_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ebs": {
							Type:     schema.TypeMap,
							Computed: true,
						},
					},
				},
			},
			"product_codes": {
				Type:     schema.TypeSet,
				Computed: true,
				Set:      amiProductCodesHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"product_code_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"product_code_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"state_reason": {
				Type:     schema.TypeMap,
				Computed: true,
			},
			"tags": tagsSchemaComputed(),
		},
	}
}

// dataSourceAwsAmiDescriptionRead performs the AMI lookup.
func dataSourceAwsAmiRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	executableUsers, executableUsersOk := d.GetOk("executable_users")
	filters, filtersOk := d.GetOk("filter")
	nameRegex, nameRegexOk := d.GetOk("name_regex")
	owners, ownersOk := d.GetOk("owners")

	if !executableUsersOk && !filtersOk && !nameRegexOk && !ownersOk {
		return fmt.Errorf("One of executable_users, filters, name_regex, or owners must be assigned")
	}

	params := &ec2.DescribeImagesInput{}
	if executableUsersOk {
		params.ExecutableUsers = expandStringList(executableUsers.([]interface{}))
	}
	if filtersOk {
		params.Filters = buildAwsDataSourceFilters(filters.(*schema.Set))
	}
	if ownersOk {
		o := expandStringList(owners.([]interface{}))

		if len(o) > 0 {
			params.Owners = o
		}
	}

	// Deprecated: pre-2.0.0 warning logging
	if !ownersOk {
		log.Print("[WARN] The \"owners\" argument will become required in the next major version.")
		log.Print("[WARN] Documentation can be found at: https://www.terraform.io/docs/providers/aws/d/ami.html#owners")

		missingOwnerFilter := true

		if filtersOk {
			for _, filter := range params.Filters {
				if aws.StringValue(filter.Name) == "owner-alias" || aws.StringValue(filter.Name) == "owner-id" {
					missingOwnerFilter = false
					break
				}
			}
		}

		if missingOwnerFilter {
			log.Print("[WARN] Potential security issue: missing \"owners\" filtering for AMI. Check AMI to ensure it came from trusted source.")
		}
	}

	log.Printf("[DEBUG] Reading AMI: %s", params)
	resp, err := conn.DescribeImages(params)
	if err != nil {
		return err
	}

	var filteredImages []*ec2.Image
	if nameRegexOk {
		r := regexp.MustCompile(nameRegex.(string))
		for _, image := range resp.Images {
			// Check for a very rare case where the response would include no
			// image name. No name means nothing to attempt a match against,
			// therefore we are skipping such image.
			if image.Name == nil || *image.Name == "" {
				log.Printf("[WARN] Unable to find AMI name to match against "+
					"for image ID %q owned by %q, nothing to do.",
					*image.ImageId, *image.OwnerId)
				continue
			}
			if r.MatchString(*image.Name) {
				filteredImages = append(filteredImages, image)
			}
		}
	} else {
		filteredImages = resp.Images[:]
	}

	if len(filteredImages) < 1 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(filteredImages) > 1 {
		if !d.Get("most_recent").(bool) {
			return fmt.Errorf("Your query returned more than one result. Please try a more " +
				"specific search criteria, or set `most_recent` attribute to true.")
		}
		sort.Slice(filteredImages, func(i, j int) bool {
			itime, _ := time.Parse(time.RFC3339, aws.StringValue(filteredImages[i].CreationDate))
			jtime, _ := time.Parse(time.RFC3339, aws.StringValue(filteredImages[j].CreationDate))
			return itime.Unix() > jtime.Unix()
		})
	}

	return amiDescriptionAttributes(d, filteredImages[0])
}

// populate the numerous fields that the image description returns.
func amiDescriptionAttributes(d *schema.ResourceData, image *ec2.Image) error {
	// Simple attributes first
	d.SetId(*image.ImageId)
	d.Set("architecture", image.Architecture)
	d.Set("creation_date", image.CreationDate)
	if image.Description != nil {
		d.Set("description", image.Description)
	}
	d.Set("hypervisor", image.Hypervisor)
	d.Set("image_id", image.ImageId)
	d.Set("image_location", image.ImageLocation)
	if image.ImageOwnerAlias != nil {
		d.Set("image_owner_alias", image.ImageOwnerAlias)
	}
	d.Set("image_type", image.ImageType)
	if image.KernelId != nil {
		d.Set("kernel_id", image.KernelId)
	}
	d.Set("name", image.Name)
	d.Set("owner_id", image.OwnerId)
	if image.Platform != nil {
		d.Set("platform", image.Platform)
	}
	d.Set("public", image.Public)
	if image.RamdiskId != nil {
		d.Set("ramdisk_id", image.RamdiskId)
	}
	if image.RootDeviceName != nil {
		d.Set("root_device_name", image.RootDeviceName)
	}
	d.Set("root_device_type", image.RootDeviceType)
	d.Set("root_snapshot_id", amiRootSnapshotId(image))
	if image.SriovNetSupport != nil {
		d.Set("sriov_net_support", image.SriovNetSupport)
	}
	d.Set("state", image.State)
	d.Set("virtualization_type", image.VirtualizationType)
	// Complex types get their own functions
	if err := d.Set("block_device_mappings", amiBlockDeviceMappings(image.BlockDeviceMappings)); err != nil {
		return err
	}
	if err := d.Set("product_codes", amiProductCodes(image.ProductCodes)); err != nil {
		return err
	}
	if err := d.Set("state_reason", amiStateReason(image.StateReason)); err != nil {
		return err
	}
	if err := d.Set("tags", tagsToMap(image.Tags)); err != nil {
		return err
	}
	return nil
}

// Returns a set of block device mappings.
func amiBlockDeviceMappings(m []*ec2.BlockDeviceMapping) *schema.Set {
	s := &schema.Set{
		F: amiBlockDeviceMappingHash,
	}
	for _, v := range m {
		mapping := map[string]interface{}{
			"device_name":  aws.StringValue(v.DeviceName),
			"virtual_name": aws.StringValue(v.VirtualName),
		}

		if v.Ebs != nil {
			ebs := map[string]interface{}{
				"delete_on_termination": fmt.Sprintf("%t", aws.BoolValue(v.Ebs.DeleteOnTermination)),
				"encrypted":             fmt.Sprintf("%t", aws.BoolValue(v.Ebs.Encrypted)),
				"iops":                  fmt.Sprintf("%d", aws.Int64Value(v.Ebs.Iops)),
				"volume_size":           fmt.Sprintf("%d", aws.Int64Value(v.Ebs.VolumeSize)),
				"snapshot_id":           aws.StringValue(v.Ebs.SnapshotId),
				"volume_type":           aws.StringValue(v.Ebs.VolumeType),
			}

			mapping["ebs"] = ebs
		}

		log.Printf("[DEBUG] aws_ami - adding block device mapping: %v", mapping)
		s.Add(mapping)
	}
	return s
}

// Returns a set of product codes.
func amiProductCodes(m []*ec2.ProductCode) *schema.Set {
	s := &schema.Set{
		F: amiProductCodesHash,
	}
	for _, v := range m {
		code := map[string]interface{}{
			"product_code_id":   aws.StringValue(v.ProductCodeId),
			"product_code_type": aws.StringValue(v.ProductCodeType),
		}
		s.Add(code)
	}
	return s
}

// Returns the root snapshot ID for an image, if it has one
func amiRootSnapshotId(image *ec2.Image) string {
	if image.RootDeviceName == nil {
		return ""
	}
	for _, bdm := range image.BlockDeviceMappings {
		if bdm.DeviceName == nil || *bdm.DeviceName != *image.RootDeviceName {
			continue
		}
		if bdm.Ebs != nil && bdm.Ebs.SnapshotId != nil {
			return *bdm.Ebs.SnapshotId
		}
	}
	return ""
}

// Returns the state reason.
func amiStateReason(m *ec2.StateReason) map[string]interface{} {
	s := make(map[string]interface{})
	if m != nil {
		s["code"] = aws.StringValue(m.Code)
		s["message"] = aws.StringValue(m.Message)
	} else {
		s["code"] = "UNSET"
		s["message"] = "UNSET"
	}
	return s
}

// Generates a hash for the set hash function used by the block_device_mappings
// attribute.
func amiBlockDeviceMappingHash(v interface{}) int {
	var buf bytes.Buffer
	// All keys added in alphabetical order.
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
	if d, ok := m["ebs"]; ok {
		if len(d.(map[string]interface{})) > 0 {
			e := d.(map[string]interface{})
			buf.WriteString(fmt.Sprintf("%s-", e["delete_on_termination"].(string)))
			buf.WriteString(fmt.Sprintf("%s-", e["encrypted"].(string)))
			buf.WriteString(fmt.Sprintf("%s-", e["iops"].(string)))
			buf.WriteString(fmt.Sprintf("%s-", e["volume_size"].(string)))
			buf.WriteString(fmt.Sprintf("%s-", e["volume_type"].(string)))
		}
	}
	if d, ok := m["no_device"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", d.(string)))
	}
	if d, ok := m["virtual_name"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", d.(string)))
	}
	if d, ok := m["snapshot_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", d.(string)))
	}
	return hashcode.String(buf.String())
}

// Generates a hash for the set hash function used by the product_codes
// attribute.
func amiProductCodesHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	// All keys added in alphabetical order.
	buf.WriteString(fmt.Sprintf("%s-", m["product_code_id"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["product_code_type"].(string)))
	return hashcode.String(buf.String())
}
