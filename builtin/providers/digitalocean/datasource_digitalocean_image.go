package digitalocean

import (
	"context"
	"fmt"
	"strconv"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceDigitalOceanImage() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDigitalOceanImageRead,
		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "name of the image",
			},
			// computed attributes
			"image": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "slug or id of the image",
			},
			"min_disk_size": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "minimum disk size required by the image",
			},
			"private": &schema.Schema{
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Is the image private or non-private",
			},
			"regions": &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of the regions that the image is available in",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"type": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "type of the image",
			},
		},
	}
}

func dataSourceDigitalOceanImageRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	opts := &godo.ListOptions{}

	images, _, err := client.Images.ListUser(context.Background(), opts)
	if err != nil {
		d.SetId("")
		return err
	}
	image, err := findImageByName(images, d.Get("name").(string))

	if err != nil {
		return err
	}

	d.SetId(image.Name)
	d.Set("name", image.Name)
	d.Set("image", strconv.Itoa(image.ID))
	d.Set("min_disk_size", image.MinDiskSize)
	d.Set("private", !image.Public)
	d.Set("regions", image.Regions)
	d.Set("type", image.Type)

	return nil
}

func findImageByName(images []godo.Image, name string) (*godo.Image, error) {
	results := make([]godo.Image, 0)
	for _, v := range images {
		if v.Name == name {
			results = append(results, v)
		}
	}
	if len(results) == 1 {
		return &results[0], nil
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no user image found with name %s", name)
	}
	return nil, fmt.Errorf("too many user images found with name %s (found %d, expected 1)", name, len(results))
}
