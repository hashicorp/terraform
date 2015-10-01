package statuscake

import (
	"fmt"
	"strconv"

	"log"

	"github.com/DreamItGetIT/statuscake"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceStatusCakeTest() *schema.Resource {
	return &schema.Resource{
		Create: CreateTest,
		Update: UpdateTest,
		Delete: DeleteTest,
		Read:   ReadTest,

		Schema: map[string]*schema.Schema{
			"test_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"website_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"website_url": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"check_rate": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  300,
			},

			"test_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"paused": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func CreateTest(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*statuscake.Client)

	newTest := &statuscake.Test{
		WebsiteName: "posters.dreamitget.it",
		WebsiteURL:  "https://posters.dreamitget.it",
		TestType:    "HTTP",
		CheckRate:   500,
	}

	//	newTest := &statuscake.Test{
	//		WebsiteName: d.Get("website_name").(string),
	//		WebsiteURL:  d.Get("website_url").(string),
	//		TestType:    d.Get("test_type").(string),
	//		CheckRate:   500,
	//	}

	log.Printf("[DEBUG] Check Rate: %d", d.Get("check_rate").(int))
	log.Printf("[DEBUG] TestType: %s", d.Get("test_type").(string))
	log.Printf("[DEBUG] Creating new StatusCake Test: %s", d.Get("website_name").(string))

	response, err := client.Tests().Put(newTest)
	if err != nil {
		return fmt.Errorf("Error creating StatusCake Test: %s", err.Error())
	}

	d.Set("test_id", fmt.Sprintf("%d", response.TestID))
	d.SetId(fmt.Sprintf("%d", response.TestID))

	return UpdateTest(d, meta)
}

func UpdateTest(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func DeleteTest(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*statuscake.Client)

	testId, parseErr := strconv.Atoi(d.Id())
	if parseErr != nil {
		return parseErr
	}
	testIntId := int(testId)

	log.Printf("[DEBUG] Deleting StatusCake Test: %s", d.Id())
	err := client.Tests().Delete(testIntId)
	if err != nil {
		return err
	}

	return nil
}

func ReadTest(d *schema.ResourceData, meta interface{}) error {
	return nil
}
