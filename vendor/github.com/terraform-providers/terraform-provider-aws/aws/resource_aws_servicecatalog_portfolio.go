package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/servicecatalog"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsServiceCatalogPortfolio() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsServiceCatalogPortfolioCreate,
		Read:   resourceAwsServiceCatalogPortfolioRead,
		Update: resourceAwsServiceCatalogPortfolioUpdate,
		Delete: resourceAwsServiceCatalogPortfolioDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_time": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateServiceCatalogPortfolioName,
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateServiceCatalogPortfolioDescription,
			},
			"provider_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateServiceCatalogPortfolioProviderName,
			},
			"tags": tagsSchema(),
		},
	}
}
func resourceAwsServiceCatalogPortfolioCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).scconn
	input := servicecatalog.CreatePortfolioInput{
		AcceptLanguage: aws.String("en"),
	}
	name := d.Get("name").(string)
	input.DisplayName = &name
	now := time.Now()
	input.IdempotencyToken = aws.String(fmt.Sprintf("%d", now.UnixNano()))

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("provider_name"); ok {
		input.ProviderName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("tags"); ok {
		tags := []*servicecatalog.Tag{}
		t := v.(map[string]interface{})
		for k, v := range t {
			tag := servicecatalog.Tag{
				Key:   aws.String(k),
				Value: aws.String(v.(string)),
			}
			tags = append(tags, &tag)
		}
		input.Tags = tags
	}

	log.Printf("[DEBUG] Creating Service Catalog Portfolio: %#v", input)
	resp, err := conn.CreatePortfolio(&input)
	if err != nil {
		return fmt.Errorf("Creating Service Catalog Portfolio failed: %s", err.Error())
	}
	d.SetId(*resp.PortfolioDetail.Id)

	return resourceAwsServiceCatalogPortfolioRead(d, meta)
}

func resourceAwsServiceCatalogPortfolioRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).scconn
	input := servicecatalog.DescribePortfolioInput{
		AcceptLanguage: aws.String("en"),
	}
	input.Id = aws.String(d.Id())

	log.Printf("[DEBUG] Reading Service Catalog Portfolio: %#v", input)
	resp, err := conn.DescribePortfolio(&input)
	if err != nil {
		if scErr, ok := err.(awserr.Error); ok && scErr.Code() == "ResourceNotFoundException" {
			log.Printf("[WARN] Service Catalog Portfolio %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Reading ServiceCatalog Portfolio '%s' failed: %s", *input.Id, err.Error())
	}
	portfolioDetail := resp.PortfolioDetail
	if err := d.Set("created_time", portfolioDetail.CreatedTime.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Error setting created_time: %s", err)
	}
	d.Set("arn", portfolioDetail.ARN)
	d.Set("description", portfolioDetail.Description)
	d.Set("name", portfolioDetail.DisplayName)
	d.Set("provider_name", portfolioDetail.ProviderName)
	tags := map[string]string{}
	for _, tag := range resp.Tags {
		tags[*tag.Key] = *tag.Value
	}
	d.Set("tags", tags)
	return nil
}

func resourceAwsServiceCatalogPortfolioUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).scconn
	input := servicecatalog.UpdatePortfolioInput{
		AcceptLanguage: aws.String("en"),
		Id:             aws.String(d.Id()),
	}

	if d.HasChange("name") {
		v, _ := d.GetOk("name")
		input.DisplayName = aws.String(v.(string))
	}

	if d.HasChange("accept_language") {
		v, _ := d.GetOk("accept_language")
		input.AcceptLanguage = aws.String(v.(string))
	}

	if d.HasChange("description") {
		v, _ := d.GetOk("description")
		input.Description = aws.String(v.(string))
	}

	if d.HasChange("provider_name") {
		v, _ := d.GetOk("provider_name")
		input.ProviderName = aws.String(v.(string))
	}

	if d.HasChange("tags") {
		currentTags, requiredTags := d.GetChange("tags")
		log.Printf("[DEBUG] Current Tags: %#v", currentTags)
		log.Printf("[DEBUG] Required Tags: %#v", requiredTags)

		tagsToAdd, tagsToRemove := tagUpdates(requiredTags.(map[string]interface{}), currentTags.(map[string]interface{}))
		log.Printf("[DEBUG] Tags To Add: %#v", tagsToAdd)
		log.Printf("[DEBUG] Tags To Remove: %#v", tagsToRemove)
		input.AddTags = tagsToAdd
		input.RemoveTags = tagsToRemove
	}

	log.Printf("[DEBUG] Update Service Catalog Portfolio: %#v", input)
	_, err := conn.UpdatePortfolio(&input)
	if err != nil {
		return fmt.Errorf("Updating Service Catalog Portfolio '%s' failed: %s", *input.Id, err.Error())
	}
	return resourceAwsServiceCatalogPortfolioRead(d, meta)
}

func tagUpdates(requriedTags, currentTags map[string]interface{}) ([]*servicecatalog.Tag, []*string) {
	var tagsToAdd []*servicecatalog.Tag
	var tagsToRemove []*string

	for rk, rv := range requriedTags {
		addTag := true
		for ck, cv := range currentTags {
			if (rk == ck) && (rv.(string) == cv.(string)) {
				addTag = false
			}
		}
		if addTag {
			tag := &servicecatalog.Tag{Key: aws.String(rk), Value: aws.String(rv.(string))}
			tagsToAdd = append(tagsToAdd, tag)
		}
	}

	for ck, cv := range currentTags {
		removeTag := true
		for rk, rv := range requriedTags {
			if (rk == ck) && (rv.(string) == cv.(string)) {
				removeTag = false
			}
		}
		if removeTag {
			tagsToRemove = append(tagsToRemove, aws.String(ck))
		}
	}

	return tagsToAdd, tagsToRemove
}

func resourceAwsServiceCatalogPortfolioDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).scconn
	input := servicecatalog.DeletePortfolioInput{}
	input.Id = aws.String(d.Id())

	log.Printf("[DEBUG] Delete Service Catalog Portfolio: %#v", input)
	_, err := conn.DeletePortfolio(&input)
	if err != nil {
		return fmt.Errorf("Deleting Service Catalog Portfolio '%s' failed: %s", *input.Id, err.Error())
	}
	return nil
}
