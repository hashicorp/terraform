package aws

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsAmi() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsAmiRead,

		Schema: map[string]*schema.Schema{
			"executable_users": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"filter": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"values": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"most_recent": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"owners": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			// Computed values.
			"architecture": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"creation_date": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"hypervisor": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"image_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"image_location": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"image_owner_alias": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"image_type": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"kernel_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"platform": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"public": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
			"ramdisk_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"root_device_name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"root_device_type": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"sriov_net_support": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"virtualization_type": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			// Complex computed values
			"block_device_mappings": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Set:      amiBlockDeviceMappingHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"no_device": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"virtual_name": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"ebs": &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
						},
					},
				},
			},
			"product_codes": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Set:      amiProductCodesHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"product_code_id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"product_code_type": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"state_reason": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},
			"tags": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Set:      amiTagsHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// dataSourceAwsAmiDescriptionRead performs the AMI lookup.
func dataSourceAwsAmiRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	executableUsers, executableUsersOk := d.GetOk("executable_users")
	filters, filtersOk := d.GetOk("filter")
	owners, ownersOk := d.GetOk("owners")

	if executableUsersOk == false && filtersOk == false && ownersOk == false {
		return fmt.Errorf("One of executable_users, filters, or owners must be assigned")
	}

	params := &ec2.DescribeImagesInput{}
	if executableUsersOk {
		params.ExecutableUsers = expandStringList(executableUsers.([]interface{}))
	}
	if filtersOk {
		params.Filters = buildAmiFilters(filters.(*schema.Set))
	}
	if ownersOk {
		params.Owners = expandStringList(owners.([]interface{}))
	}

	resp, err := conn.DescribeImages(params)
	if err != nil {
		return err
	}
	var image *ec2.Image
	if len(resp.Images) < 1 {
		return fmt.Errorf("Your query returned no results. Please change your filters and try again.")
	} else if len(resp.Images) > 1 {
		if (d.Get("most_recent").(bool)) == true {
			log.Printf("[DEBUG] aws_ami - multiple results found and most_recent is set")
			image = mostRecentAmi(resp.Images)
		} else {
			log.Printf("[DEBUG] aws_ami - multiple results found and most_recent not set")
			return fmt.Errorf("Your query returned more than one result. Please try a more specific search, or set most_recent to true.")
		}
	} else {
		log.Printf("[DEBUG] aws_ami - Single AMI found: %s", *resp.Images[0].ImageId)
		image = resp.Images[0]
	}
	return amiDescriptionAttributes(d, image)
}

// Build a slice of AMI filter options from the filters provided.
func buildAmiFilters(set *schema.Set) []*ec2.Filter {
	var filters []*ec2.Filter
	for _, v := range set.List() {
		m := v.(map[string]interface{})
		var filterValues []*string
		for _, e := range m["values"].([]interface{}) {
			filterValues = append(filterValues, aws.String(e.(string)))
		}
		filters = append(filters, &ec2.Filter{
			Name:   aws.String(m["name"].(string)),
			Values: filterValues,
		})
	}
	return filters
}

type imageSort []*ec2.Image

func (a imageSort) Len() int      { return len(a) }
func (a imageSort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a imageSort) Less(i, j int) bool {
	itime, _ := time.Parse(time.RFC3339, *a[i].CreationDate)
	jtime, _ := time.Parse(time.RFC3339, *a[j].CreationDate)
	return itime.Unix() < jtime.Unix()
}

// Returns the most recent AMI out of a slice of images.
func mostRecentAmi(images []*ec2.Image) *ec2.Image {
	sortedImages := images
	sort.Sort(imageSort(sortedImages))
	return sortedImages[len(sortedImages)-1]
}

// populate the numerous fields that the image description returns.
func amiDescriptionAttributes(d *schema.ResourceData, image *ec2.Image) error {
	// Simple attributes first
	d.SetId(*image.ImageId)
	d.Set("architecture", *image.Architecture)
	d.Set("creation_date", *image.CreationDate)
	if image.Description != nil {
		d.Set("description", *image.Description)
	}
	d.Set("hypervisor", *image.Hypervisor)
	d.Set("image_id", *image.ImageId)
	d.Set("image_location", *image.ImageLocation)
	if image.ImageOwnerAlias != nil {
		d.Set("image_owner_alias", *image.ImageOwnerAlias)
	}
	d.Set("image_type", *image.ImageType)
	if image.KernelId != nil {
		d.Set("kernel_id", *image.KernelId)
	}
	d.Set("name", *image.Name)
	d.Set("owner_id", *image.OwnerId)
	if image.Platform != nil {
		d.Set("platform", *image.Platform)
	}
	d.Set("public", *image.Public)
	if image.RamdiskId != nil {
		d.Set("ramdisk_id", *image.RamdiskId)
	}
	if image.RootDeviceName != nil {
		d.Set("root_device_name", *image.RootDeviceName)
	}
	d.Set("root_device_type", *image.RootDeviceType)
	if image.SriovNetSupport != nil {
		d.Set("sriov_net_support", *image.SriovNetSupport)
	}
	d.Set("state", *image.State)
	d.Set("virtualization_type", *image.VirtualizationType)
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
	if err := d.Set("tags", amiTags(image.Tags)); err != nil {
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
			"device_name": *v.DeviceName,
		}
		if v.Ebs != nil {
			ebs := map[string]interface{}{
				"delete_on_termination": fmt.Sprintf("%t", *v.Ebs.DeleteOnTermination),
				"encrypted":             fmt.Sprintf("%t", *v.Ebs.Encrypted),
				"volume_size":           fmt.Sprintf("%d", *v.Ebs.VolumeSize),
				"volume_type":           *v.Ebs.VolumeType,
			}
			// Iops is not always set
			if v.Ebs.Iops != nil {
				ebs["iops"] = fmt.Sprintf("%d", *v.Ebs.Iops)
			} else {
				ebs["iops"] = "0"
			}
			// snapshot id may not be set
			if v.Ebs.SnapshotId != nil {
				ebs["snapshot_id"] = *v.Ebs.SnapshotId
			}

			mapping["ebs"] = ebs
		}
		if v.VirtualName != nil {
			mapping["virtual_name"] = *v.VirtualName
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
			"product_code_id":   *v.ProductCodeId,
			"product_code_type": *v.ProductCodeType,
		}
		s.Add(code)
	}
	return s
}

// Returns the state reason.
func amiStateReason(m *ec2.StateReason) map[string]interface{} {
	s := make(map[string]interface{})
	if m != nil {
		s["code"] = *m.Code
		s["message"] = *m.Message
	} else {
		s["code"] = "UNSET"
		s["message"] = "UNSET"
	}
	return s
}

// Returns a set of tags.
func amiTags(m []*ec2.Tag) *schema.Set {
	s := &schema.Set{
		F: amiTagsHash,
	}
	for _, v := range m {
		tag := map[string]interface{}{
			"key":   *v.Key,
			"value": *v.Value,
		}
		s.Add(tag)
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

// Generates a hash for the set hash function used by the tags
// attribute.
func amiTagsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	// All keys added in alphabetical order.
	buf.WriteString(fmt.Sprintf("%s-", m["key"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["value"].(string)))
	return hashcode.String(buf.String())
}
