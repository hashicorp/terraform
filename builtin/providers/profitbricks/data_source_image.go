package profitbricks

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
	"strings"
)

func dataSourceImage() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceImageRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"type": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"location": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"version": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func dataSourceImageRead(d *schema.ResourceData, meta interface{}) error {
	profitbricks.SetDepth("5")

	images := profitbricks.ListImages()

	if images.StatusCode > 299 {
		return fmt.Errorf("An error occured while fetching ProfitBricks locations %s", images.Response)
	}

	name := d.Get("name").(string)
	imageType, imageTypeOk := d.GetOk("type")
	location, locationOk := d.GetOk("location")
	version, versionOk := d.GetOk("version")

	results := []profitbricks.Image{}

	// if version value is present then concatenate name - version
	// otherwise search by name or part of the name
	if versionOk {
		name_ver := fmt.Sprintf("%s-%s", name, version.(string))
		for _, img := range images.Items {
			if strings.Contains(strings.ToLower(img.Properties.Name), strings.ToLower(name_ver)) {
				results = append(results, img)
			}
		}
	} else {
		for _, img := range images.Items {
			if strings.Contains(strings.ToLower(img.Properties.Name), strings.ToLower(name)) {
				results = append(results, img)
			}
		}
	}

	if imageTypeOk {
		imageTypeResults := []profitbricks.Image{}
		for _, img := range results {
			if img.Properties.ImageType == imageType.(string) {
				imageTypeResults = append(imageTypeResults, img)
			}

		}
		results = imageTypeResults
	}

	if locationOk {
		locationResults := []profitbricks.Image{}
		for _, img := range results {
			if img.Properties.Location == location.(string) {
				locationResults = append(locationResults, img)
			}

		}
		results = locationResults
	}

	if len(results) > 1 {
		return fmt.Errorf("There is more than one image that match the search criteria")
	}

	if len(results) == 0 {
		return fmt.Errorf("There are no images that match the search criteria")
	}

	d.Set("name", results[0].Properties.Name)

	d.SetId(results[0].Id)

	return nil
}
