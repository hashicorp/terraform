package aws

import (
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

func dataSourceAwsAmiIds() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsAmiIdsRead,

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
			"owners": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"ids": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"sort_ascending": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},
		},
	}
}

func dataSourceAwsAmiIdsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	executableUsers, executableUsersOk := d.GetOk("executable_users")
	filters, filtersOk := d.GetOk("filter")
	nameRegex, nameRegexOk := d.GetOk("name_regex")
	owners, ownersOk := d.GetOk("owners")
	sortAscending := d.Get("sort_ascending").(bool)

	if executableUsersOk == false && filtersOk == false && nameRegexOk == false && ownersOk == false {
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

	log.Printf("[DEBUG] Reading AMI IDs: %s", params)
	resp, err := conn.DescribeImages(params)
	if err != nil {
		return err
	}

	var filteredImages []*ec2.Image
	imageIds := make([]string, 0)

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

	sort.Slice(filteredImages, func(i, j int) bool {
		itime, _ := time.Parse(time.RFC3339, aws.StringValue(filteredImages[i].CreationDate))
		jtime, _ := time.Parse(time.RFC3339, aws.StringValue(filteredImages[j].CreationDate))
		if sortAscending {
			return itime.Unix() < jtime.Unix()
		}
		return itime.Unix() > jtime.Unix()
	})
	for _, image := range filteredImages {
		imageIds = append(imageIds, *image.ImageId)
	}

	d.SetId(fmt.Sprintf("%d", hashcode.String(params.String())))
	d.Set("ids", imageIds)

	return nil
}
