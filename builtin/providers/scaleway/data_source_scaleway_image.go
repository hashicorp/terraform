package scaleway

import (
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/scaleway/scaleway-cli/pkg/api"
)

func dataSourceScalewayImage() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceScalewayImageRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"name_filter": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"architecture": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			// Computed values.
			"organization": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"public": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
			"creation_date": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func scalewayImageAttributes(d *schema.ResourceData, img imageMatch) error {
	d.Set("architecture", img.imageDefinition.Arch)
	d.Set("organization", img.marketImage.Organization)
	d.Set("public", img.marketImage.Public)
	d.Set("creation_date", img.marketImage.CreationDate)
	d.Set("name", img.marketImage.Name)
	d.SetId(img.imageDefinition.ID)

	return nil
}

type imageMatch struct {
	marketImage     api.MarketImage
	imageDefinition api.MarketLocalImageDefinition
}

func dataSourceScalewayImageRead(d *schema.ResourceData, meta interface{}) error {
	scaleway := meta.(*Client).scaleway

	images, err := scaleway.GetImages()
	log.Printf("[DEBUG] %#v", images)
	if err != nil {
		return err
	}

	var isNameMatch = func(api.MarketImage) bool { return true }
	var isArchMatch = func(api.MarketLocalImageDefinition) bool { return true }

	if name, ok := d.GetOk("name"); ok {
		isNameMatch = func(img api.MarketImage) bool {
			return img.Name == name.(string)
		}
	} else if nameFilter, ok := d.GetOk("name_filter"); ok {
		exp, err := regexp.Compile(nameFilter.(string))
		if err != nil {
			return err
		}

		isNameMatch = func(img api.MarketImage) bool {
			return exp.MatchString(img.Name)
		}
	}

	var architecture = d.Get("architecture").(string)
	if architecture != "" {
		isArchMatch = func(img api.MarketLocalImageDefinition) bool {
			return img.Arch == architecture
		}
	}

	var matches []imageMatch
	for _, img := range *images {
		if !isNameMatch(img) {
			continue
		}

		var imageDefinition *api.MarketLocalImageDefinition
		for _, version := range img.Versions {
			for _, def := range version.LocalImages {
				if isArchMatch(def) {
					imageDefinition = &def
					break
				}
			}
		}

		if imageDefinition == nil {
			continue
		}
		matches = append(matches, imageMatch{
			marketImage:     img,
			imageDefinition: *imageDefinition,
		})
	}

	if len(matches) > 1 {
		return fmt.Errorf("The query returned more than one result. Please refine your query.")
	}
	if len(matches) == 0 {
		return fmt.Errorf("The query returned no result. Please refine your query.")
	}

	return scalewayImageAttributes(d, matches[0])
}
