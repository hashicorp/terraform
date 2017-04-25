package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSesConfigurationSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSesConfigurationSetCreate,
		Read:   resourceAwsSesConfigurationSetRead,
		Delete: resourceAwsSesConfigurationSetDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsSesConfigurationSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	configurationSetName := d.Get("name").(string)

	createOpts := &ses.CreateConfigurationSetInput{
		ConfigurationSet: &ses.ConfigurationSet{
			Name: aws.String(configurationSetName),
		},
	}

	_, err := conn.CreateConfigurationSet(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating SES configuration set: %s", err)
	}

	d.SetId(configurationSetName)

	return resourceAwsSesConfigurationSetRead(d, meta)
}

func resourceAwsSesConfigurationSetRead(d *schema.ResourceData, meta interface{}) error {
	configurationSetExists, err := findConfigurationSet(d.Id(), nil, meta)

	if !configurationSetExists {
		log.Printf("[WARN] SES Configuration Set (%s) not found", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return err
	}

	d.Set("name", d.Id())

	return nil
}

func resourceAwsSesConfigurationSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	log.Printf("[DEBUG] SES Delete Configuration Rule Set: %s", d.Id())
	_, err := conn.DeleteConfigurationSet(&ses.DeleteConfigurationSetInput{
		ConfigurationSetName: aws.String(d.Id()),
	})

	if err != nil {
		return err
	}

	return nil
}

func findConfigurationSet(name string, token *string, meta interface{}) (bool, error) {
	conn := meta.(*AWSClient).sesConn

	configurationSetExists := false

	listOpts := &ses.ListConfigurationSetsInput{
		NextToken: token,
	}

	response, err := conn.ListConfigurationSets(listOpts)
	for _, element := range response.ConfigurationSets {
		if *element.Name == name {
			configurationSetExists = true
		}
	}

	if err != nil && !configurationSetExists && response.NextToken != nil {
		configurationSetExists, err = findConfigurationSet(name, response.NextToken, meta)
	}

	if err != nil {
		return false, err
	}

	return configurationSetExists, nil
}
