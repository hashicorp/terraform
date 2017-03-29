package profitbricks

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
	"log"
	"regexp"
	"runtime"
	"strings"
	"time"
)

func resourceProfitBricksDatacenter() *schema.Resource {
	return &schema.Resource{
		Create: resourceProfitBricksDatacenterCreate,
		Read:   resourceProfitBricksDatacenterRead,
		Update: resourceProfitBricksDatacenterUpdate,
		Delete: resourceProfitBricksDatacenterDelete,
		Schema: map[string]*schema.Schema{

			//Datacenter parameters
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"location": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceProfitBricksDatacenterCreate(d *schema.ResourceData, meta interface{}) error {
	datacenter := profitbricks.Datacenter{
		Properties: profitbricks.DatacenterProperties{
			Name:     d.Get("name").(string),
			Location: d.Get("location").(string),
		},
	}

	if attr, ok := d.GetOk("description"); ok {
		datacenter.Properties.Description = attr.(string)
	}
	dc := profitbricks.CreateDatacenter(datacenter)

	if dc.StatusCode > 299 {
		return fmt.Errorf(
			"Error creating data center (%s) (%s)", d.Id(), dc.Response)
	}
	d.SetId(dc.Id)

	log.Printf("[INFO] DataCenter Id: %s", d.Id())

	err := waitTillProvisioned(meta, dc.Headers.Get("Location"))
	if err != nil {
		return err
	}
	return resourceProfitBricksDatacenterRead(d, meta)
}

func resourceProfitBricksDatacenterRead(d *schema.ResourceData, meta interface{}) error {
	datacenter := profitbricks.GetDatacenter(d.Id())
	if datacenter.StatusCode > 299 {
		if datacenter.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error while fetching a data center ID %s %s", d.Id(), datacenter.Response)
	}

	d.Set("name", datacenter.Properties.Name)
	d.Set("location", datacenter.Properties.Location)
	d.Set("description", datacenter.Properties.Description)
	return nil
}

func resourceProfitBricksDatacenterUpdate(d *schema.ResourceData, meta interface{}) error {
	obj := profitbricks.DatacenterProperties{}

	if d.HasChange("name") {
		_, newName := d.GetChange("name")

		obj.Name = newName.(string)
	}

	if d.HasChange("description") {
		_, newDescription := d.GetChange("description")
		obj.Description = newDescription.(string)
	}

	resp := profitbricks.PatchDatacenter(d.Id(), obj)
	waitTillProvisioned(meta, resp.Headers.Get("Location"))
	return resourceProfitBricksDatacenterRead(d, meta)
}

func resourceProfitBricksDatacenterDelete(d *schema.ResourceData, meta interface{}) error {
	dcid := d.Id()
	resp := profitbricks.DeleteDatacenter(dcid)

	if resp.StatusCode > 299 {
		return fmt.Errorf("An error occured while deleting the data center ID %s %s", d.Id(), string(resp.Body))
	}
	err := waitTillProvisioned(meta, resp.Headers.Get("Location"))
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func waitTillProvisioned(meta interface{}, path string) error {
	config := meta.(*Config)
	waitCount := 50

	if config.Retries != 0 {
		waitCount = config.Retries
	}
	for i := 0; i < waitCount; i++ {
		request := profitbricks.GetRequestStatus(path)
		pc, _, _, ok := runtime.Caller(1)
		details := runtime.FuncForPC(pc)
		if ok && details != nil {
			log.Printf("[DEBUG] Called from %s", details.Name())
		}
		log.Printf("[DEBUG] Request status: %s", request.Metadata.Status)
		log.Printf("[DEBUG] Request status path: %s", path)

		if request.Metadata.Status == "DONE" {
			return nil
		}
		if request.Metadata.Status == "FAILED" {

			return fmt.Errorf("Request failed with following error: %s", request.Metadata.Message)
		}
		time.Sleep(10 * time.Second)
		i++
	}
	return fmt.Errorf("Timeout has expired")
}

func getImageId(dcId string, imageName string, imageType string) string {
	if imageName == "" {
		return ""
	}
	dc := profitbricks.GetDatacenter(dcId)
	if dc.StatusCode > 299 {
		log.Print(fmt.Errorf("Error while fetching a data center ID %s %s", dcId, dc.Response))
	}

	images := profitbricks.ListImages()
	if images.StatusCode > 299 {
		log.Print(fmt.Errorf("Error while fetching the list of images %s", images.Response))
	}

	if len(images.Items) > 0 {
		for _, i := range images.Items {
			imgName := ""
			if i.Properties.Name != "" {
				imgName = i.Properties.Name
			}

			if imageType == "SSD" {
				imageType = "HDD"
			}
			if imgName != "" && strings.Contains(strings.ToLower(imgName), strings.ToLower(imageName)) && i.Properties.ImageType == imageType && i.Properties.Location == dc.Properties.Location && i.Properties.Public == true {
				return i.Id
			}
		}
	}
	return ""
}

func IsValidUUID(uuid string) bool {
	r := regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$")
	return r.MatchString(uuid)
}
