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
			"timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func CreateTest(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*statuscake.Client)

	newTest := &statuscake.Test{
		WebsiteName: d.Get("website_name").(string),
		WebsiteURL:  d.Get("website_url").(string),
		TestType:    d.Get("test_type").(string),
		CheckRate:   d.Get("check_rate").(int),
	}

	log.Printf("[DEBUG] Creating new StatusCake Test: %s", d.Get("website_name").(string))

	response, err := client.Tests().Update(newTest)
	if err != nil {
		return fmt.Errorf("Error creating StatusCake Test: %s", err.Error())
	}

	d.Set("test_id", fmt.Sprintf("%d", response.TestID))
	d.SetId(fmt.Sprintf("%d", response.TestID))

	return ReadTest(d, meta)
}

func UpdateTest(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*statuscake.Client)

	params := getStatusCakeTestInput(d)

	log.Printf("[DEBUG] StatusCake Test Update for %s", d.Id())
	_, err := client.Tests().Update(params)
	if err != nil {
		return fmt.Errorf("Error Updating StatusCake Test: %s", err.Error())
	}
	return nil
}

func DeleteTest(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*statuscake.Client)

	testId, parseErr := strconv.Atoi(d.Id())
	if parseErr != nil {
		return parseErr
	}
	log.Printf("[DEBUG] Deleting StatusCake Test: %s", d.Id())
	err := client.Tests().Delete(testId)
	if err != nil {
		return err
	}

	return nil
}

func ReadTest(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*statuscake.Client)

	testId, parseErr := strconv.Atoi(d.Id())
	if parseErr != nil {
		return parseErr
	}
	testResp, err := client.Tests().Detail(testId)
	if err != nil {
		return fmt.Errorf("Error Getting StatusCake Test Details for %s: Error: %s", d.Id(), err)
	}
	d.Set("check_rate", testResp.CheckRate)

	return nil
}

func getStatusCakeTestInput(d *schema.ResourceData) *statuscake.Test {
	testId, parseErr := strconv.Atoi(d.Id())
	if parseErr != nil {
		log.Printf("[DEBUG] Error Parsing StatusCake TestID: %s", d.Id())
	}
	test := &statuscake.Test{
		TestID: testId,
	}
	if v, ok := d.GetOk("website_name"); ok {
		test.WebsiteName = v.(string)
	}
	if v, ok := d.GetOk("website_url"); ok {
		test.WebsiteURL = v.(string)
	}
	if v, ok := d.GetOk("check_rate"); ok {
		test.CheckRate = v.(int)
	}
	if v, ok := d.GetOk("test_type"); ok {
		test.TestType = v.(string)
	}
	if v, ok := d.GetOk("paused"); ok {
		test.Paused = v.(bool)
	}
	if v, ok := d.GetOk("timeout"); ok {
		test.Timeout = v.(int)
	}
	return test
}
