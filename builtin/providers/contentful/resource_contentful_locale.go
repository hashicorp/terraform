package main

import (
	"github.com/hashicorp/terraform/helper/schema"
	contentful "github.com/tolgaakyuz/contentful-go"
)

func resourceContentfulLocale() *schema.Resource {
	return &schema.Resource{
		Create: resourceCreateLocale,
		Read:   resourceReadLocale,
		Update: resourceUpdateLocale,
		Delete: resourceDeleteLocale,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"version": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"space_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"code": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"fallback_code": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "en-US",
			},
			"optional": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"cda": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"cma": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceCreateLocale(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Contentful)
	spaceID := d.Get("space_id").(string)

	locale := &contentful.Locale{
		Name:         d.Get("name").(string),
		Code:         d.Get("code").(string),
		FallbackCode: d.Get("fallback_code").(string),
		Optional:     d.Get("optional").(bool),
		CDA:          d.Get("cda").(bool),
		CMA:          d.Get("cma").(bool),
	}

	err = client.Locales.Upsert(spaceID, locale)
	if err != nil {
		return err
	}

	err = setLocaleProperties(d, locale)
	if err != nil {
		return err
	}

	d.SetId(locale.Sys.ID)

	return nil
}

func resourceReadLocale(d *schema.ResourceData, m interface{}) error {
	client := m.(*contentful.Contentful)
	spaceID := d.Get("space_id").(string)
	localeID := d.Id()

	locale, err := client.Locales.Get(spaceID, localeID)
	if _, ok := err.(*contentful.NotFoundError); ok {
		d.SetId("")
		return nil
	}

	if err != nil {
		return err
	}

	return setLocaleProperties(d, locale)
}

func resourceUpdateLocale(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Contentful)
	spaceID := d.Get("space_id").(string)
	localeID := d.Id()

	locale, err := client.Locales.Get(spaceID, localeID)
	if err != nil {
		return err
	}

	locale.Name = d.Get("name").(string)
	locale.Code = d.Get("code").(string)
	locale.FallbackCode = d.Get("fallback_code").(string)
	locale.Optional = d.Get("optional").(bool)
	locale.CDA = d.Get("cda").(bool)
	locale.CMA = d.Get("cma").(bool)

	err = client.Locales.Upsert(spaceID, locale)
	if err != nil {
		return err
	}

	err = setLocaleProperties(d, locale)
	if err != nil {
		return err
	}

	return nil
}

func resourceDeleteLocale(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Contentful)
	spaceID := d.Get("space_id").(string)
	localeID := d.Id()

	locale, err := client.Locales.Get(spaceID, localeID)
	if err != nil {
		return err
	}

	err = client.Locales.Delete(spaceID, locale)
	if _, ok := err.(*contentful.NotFoundError); ok {
		return nil
	}

	if err != nil {
		return err
	}

	return nil
}

func setLocaleProperties(d *schema.ResourceData, locale *contentful.Locale) error {
	err := d.Set("name", locale.Name)
	if err != nil {
		return err
	}

	err = d.Set("code", locale.Code)
	if err != nil {
		return err
	}

	err = d.Set("fallback_code", locale.FallbackCode)
	if err != nil {
		return err
	}

	err = d.Set("optional", locale.Optional)
	if err != nil {
		return err
	}

	err = d.Set("cda", locale.CDA)
	if err != nil {
		return err
	}

	err = d.Set("cma", locale.CMA)
	if err != nil {
		return err
	}

	return nil
}
