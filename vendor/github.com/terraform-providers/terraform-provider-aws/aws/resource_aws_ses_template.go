package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsSesTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSesTemplateCreate,
		Read:   resourceAwsSesTemplateRead,
		Update: resourceAwsSesTemplateUpdate,
		Delete: resourceAwsSesTemplateDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 64),
			},
			"html": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateMaxLength(512000),
			},
			"subject": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"text": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateMaxLength(512000),
			},
		},
	}
}
func resourceAwsSesTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	templateName := d.Get("name").(string)

	template := ses.Template{
		TemplateName: aws.String(templateName),
	}

	if v, ok := d.GetOk("html"); ok {
		template.HtmlPart = aws.String(v.(string))
	}

	if v, ok := d.GetOk("subject"); ok {
		template.SubjectPart = aws.String(v.(string))
	}

	if v, ok := d.GetOk("text"); ok {
		template.TextPart = aws.String(v.(string))
	}

	input := ses.CreateTemplateInput{
		Template: &template,
	}

	log.Printf("[DEBUG] Creating SES template: %#v", input)
	_, err := conn.CreateTemplate(&input)
	if err != nil {
		return fmt.Errorf("Creating SES template failed: %s", err.Error())
	}
	d.SetId(templateName)

	return resourceAwsSesTemplateRead(d, meta)
}

func resourceAwsSesTemplateRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn
	input := ses.GetTemplateInput{
		TemplateName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading SES template: %#v", input)
	gto, err := conn.GetTemplate(&input)
	if err != nil {
		if isAWSErr(err, "TemplateDoesNotExist", "") {
			log.Printf("[WARN] SES template %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Reading SES template '%s' failed: %s", *input.TemplateName, err.Error())
	}

	d.Set("html", gto.Template.HtmlPart)
	d.Set("name", gto.Template.TemplateName)
	d.Set("subject", gto.Template.SubjectPart)
	d.Set("text", gto.Template.TextPart)

	return nil
}

func resourceAwsSesTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	templateName := d.Id()

	template := ses.Template{
		TemplateName: aws.String(templateName),
	}

	if v, ok := d.GetOk("html"); ok {
		template.HtmlPart = aws.String(v.(string))
	}

	if v, ok := d.GetOk("subject"); ok {
		template.SubjectPart = aws.String(v.(string))
	}

	if v, ok := d.GetOk("text"); ok {
		template.TextPart = aws.String(v.(string))
	}

	input := ses.UpdateTemplateInput{
		Template: &template,
	}

	log.Printf("[DEBUG] Update SES template: %#v", input)
	_, err := conn.UpdateTemplate(&input)
	if err != nil {
		return fmt.Errorf("Updating SES template '%s' failed: %s", templateName, err.Error())
	}

	return resourceAwsSesTemplateRead(d, meta)
}

func resourceAwsSesTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn
	input := ses.DeleteTemplateInput{
		TemplateName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Delete SES template: %#v", input)
	_, err := conn.DeleteTemplate(&input)
	if err != nil {
		return fmt.Errorf("Deleting SES template '%s' failed: %s", *input.TemplateName, err.Error())
	}
	return nil
}
