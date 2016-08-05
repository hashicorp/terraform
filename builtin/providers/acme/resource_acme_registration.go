package acme

import "github.com/hashicorp/terraform/helper/schema"

func resourceACMERegistration() *schema.Resource {
	return &schema.Resource{
		Create: resourceACMERegistrationCreate,
		Read:   resourceACMERegistrationRead,
		Delete: resourceACMERegistrationDelete,

		Schema: registrationSchemaFull(),
	}
}

func resourceACMERegistrationCreate(d *schema.ResourceData, meta interface{}) error {
	// register and agree to the TOS
	client, user, err := expandACMEClient(d, "")
	if err != nil {
		return err
	}
	reg, err := client.Register()
	if err != nil {
		return err
	}
	user.Registration = reg
	err = client.AgreeToTOS()
	if err != nil {
		return err
	}

	// save the reg
	err = saveACMERegistration(d, reg)
	if err != nil {
		return err
	}

	return nil
}

func resourceACMERegistrationRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceACMERegistrationDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO: Add deletion support using *acme.Client.DeleteRegistration().
	// I had this, but I think I jumped the gun on it still being in draft :)
	d.SetId("")
	return nil
}
