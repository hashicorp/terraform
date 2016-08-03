package profitbricks

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func resourceProfitBricksDatacenter() *schema.Resource {
	return &schema.Resource{
		Create: resourceProfitBricksDatacenterCreate,
		Read:   resourceProfitBricksDatacenterRead,
		Update: resourceProfitBricksDatacenterUpdate,
		Delete: resourceProfitBricksDatacenterDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
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
	username, password, _ := getCredentials(meta)
	profitbricks.SetAuth(username, password)

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
	username, password, _ := getCredentials(meta)

	profitbricks.SetAuth(username, password)
	datacenter := profitbricks.GetDatacenter(d.Id())
	if datacenter.StatusCode > 299 {
		return fmt.Errorf("Error while fetching a data center ID %s %s", d.Id(), datacenter.Response)
	}

	d.Set("name", datacenter.Properties.Name)
	d.Set("location", datacenter.Properties.Location)
	d.Set("description", datacenter.Properties.Description)
	return nil
}

func resourceProfitBricksDatacenterUpdate(d *schema.ResourceData, meta interface{}) error {
	username, password, _ := getCredentials(meta)

	profitbricks.SetAuth(username, password)

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
	username, password, _ := getCredentials(meta)

	profitbricks.SetAuth(username, password)
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
	username, password, timeout := getCredentials(meta)
	profitbricks.SetAuth(username, password)
	//log.Printf("[DEBUG] Request status path: %s", path)
	waitCount := 50

	if timeout != 0 {
		waitCount = timeout
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

func getCredentials(meta interface{}) (username, password string, timeout int) {
	creds := meta.(string)

	splitv := strings.Split(creds, ",")
	username, password, to := splitv[0], splitv[1], splitv[2]
	timeout, _ = strconv.Atoi(to)
	return username, password, timeout
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
			if imgName != "" && strings.Contains(strings.ToLower(imgName), strings.ToLower(imageName)) && i.Properties.ImageType == imageType && i.Properties.Location == dc.Properties.Location {
				return i.Id
			}
		}
	}
	return ""
}
