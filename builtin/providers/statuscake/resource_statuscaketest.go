package statuscake

import "github.com/hashicorp/terraform/helper/schema"

func resourceStatusCakeTest() *schema.Resource {
	return &schema.Resource{
		Create: CreateTest,
		Update: UpdateTest,
		Delete: DeleteTest,
		Read:   ReadTest,
	}
}

func CreateTest(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func UpdateTest(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func DeleteTest(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func ReadTest(d *schema.ResourceData, meta interface{}) error {
	return nil
}
