package main

import (
	"github.com/hashicorp/terraform/helper/schema"
	contentful "github.com/tolgaakyuz/contentful-go"
)

func resourceContentfulContentType() *schema.Resource {
	return &schema.Resource{
		Create: resourceContentTypeCreate,
		Read:   resourceContentTypeRead,
		Update: resourceContentTypeUpdate,
		Delete: resourceContentTypeDelete,

		Schema: map[string]*schema.Schema{
			"space_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"version": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"display_field": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"field": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						//@TODO Add ValidateFunc to validate field type
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"link_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"items": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"validations": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"link_type": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"required": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"localized": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"disabled": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"omitted": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"validations": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func resourceContentTypeCreate(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Contentful)
	spaceID := d.Get("space_id").(string)

	ct := &contentful.ContentType{
		Name:         d.Get("name").(string),
		DisplayField: d.Get("display_field").(string),
		Fields:       []*contentful.Field{},
	}

	if description, ok := d.GetOk("description"); ok {
		ct.Description = description.(string)
	}

	for _, rawField := range d.Get("field").(*schema.Set).List() {
		field := rawField.(map[string]interface{})

		contentfulField := &contentful.Field{
			ID:        field["id"].(string),
			Name:      field["name"].(string),
			Type:      field["type"].(string),
			Localized: field["localized"].(bool),
			Required:  field["required"].(bool),
			Disabled:  field["disabled"].(bool),
			Omitted:   field["omitted"].(bool),
		}

		if linkType, ok := field["link_type"].(string); ok {
			contentfulField.LinkType = linkType
		}

		if validations, ok := field["validations"].([]interface{}); ok {
			parsedValidations, err := contentful.ParseValidations(validations)
			if err != nil {
				return err
			}

			contentfulField.Validations = parsedValidations
		}

		if items := processItems(field["items"].(*schema.Set)); items != nil {
			contentfulField.Items = items
		}

		ct.Fields = append(ct.Fields, contentfulField)
	}

	if err = client.ContentTypes.Upsert(spaceID, ct); err != nil {
		return err
	}

	if err = client.ContentTypes.Activate(spaceID, ct); err != nil {
		//@TODO Maybe delete the CT ?
		return err
	}

	if err = setContentTypeProperties(d, ct); err != nil {
		return err
	}

	d.SetId(ct.Sys.ID)

	return nil
}

func resourceContentTypeRead(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Contentful)
	spaceID := d.Get("space_id").(string)

	_, err = client.ContentTypes.Get(spaceID, d.Id())

	return err
}

func resourceContentTypeUpdate(d *schema.ResourceData, m interface{}) (err error) {
	var existingFields []*contentful.Field
	var deletedFields []*contentful.Field

	client := m.(*contentful.Contentful)
	spaceID := d.Get("space_id").(string)

	ct, err := client.ContentTypes.Get(spaceID, d.Id())
	if err != nil {
		return err
	}

	ct.Name = d.Get("name").(string)
	ct.DisplayField = d.Get("display_field").(string)

	if description, ok := d.GetOk("description"); ok {
		ct.Description = description.(string)
	}

	if d.HasChange("field") {
		old, new := d.GetChange("field")

		existingFields, deletedFields = checkFieldChanges(old.(*schema.Set), new.(*schema.Set))

		ct.Fields = existingFields

		if deletedFields != nil {
			ct.Fields = append(ct.Fields, deletedFields...)
		}
	}

	// To remove a field from a content type 4 API calls need to be made.
	// Ommit the removed fields and publish the new version of the content type,
	// followed by the field removal and final publish.
	if err = client.ContentTypes.Upsert(spaceID, ct); err != nil {
		return err
	}

	if err = client.ContentTypes.Activate(spaceID, ct); err != nil {
		//@TODO Maybe delete the CT ?
		return err
	}

	if deletedFields != nil {
		ct.Fields = existingFields

		if err = client.ContentTypes.Upsert(spaceID, ct); err != nil {
			return err
		}

		if err = client.ContentTypes.Activate(spaceID, ct); err != nil {
			//@TODO Maybe delete the CT ?
			return err
		}
	}

	return setContentTypeProperties(d, ct)
}

func resourceContentTypeDelete(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Contentful)
	spaceID := d.Get("space_id").(string)

	ct, err := client.ContentTypes.Get(spaceID, d.Id())
	if err != nil {
		return err
	}

	err = client.ContentTypes.Deactivate(spaceID, ct)
	if err != nil {
		return err
	}

	if err = client.ContentTypes.Delete(spaceID, ct); err != nil {
		return err
	}

	return nil
}

func setContentTypeProperties(d *schema.ResourceData, ct *contentful.ContentType) (err error) {

	if err = d.Set("version", ct.Sys.Version); err != nil {
		return err
	}

	return nil
}

func checkFieldChanges(old, new *schema.Set) ([]*contentful.Field, []*contentful.Field) {
	var contentfulField *contentful.Field
	var existingFields []*contentful.Field
	var deletedFields []*contentful.Field
	var fieldRemoved bool

	for _, f := range old.List() {
		oldField := f.(map[string]interface{})

		fieldRemoved = true
		for _, newField := range new.List() {
			if oldField["id"].(string) == newField.(map[string]interface{})["id"].(string) {
				fieldRemoved = false
				break
			}
		}

		if fieldRemoved {
			deletedFields = append(deletedFields,
				&contentful.Field{
					ID:        oldField["id"].(string),
					Name:      oldField["name"].(string),
					Type:      oldField["type"].(string),
					LinkType:  oldField["link_type"].(string),
					Localized: oldField["localized"].(bool),
					Required:  oldField["required"].(bool),
					Disabled:  oldField["disabled"].(bool),
					Omitted:   true,
				})
		}
	}

	for _, f := range new.List() {
		newField := f.(map[string]interface{})

		contentfulField = &contentful.Field{
			ID:        newField["id"].(string),
			Name:      newField["name"].(string),
			Type:      newField["type"].(string),
			Localized: newField["localized"].(bool),
			Required:  newField["required"].(bool),
			Disabled:  newField["disabled"].(bool),
			Omitted:   newField["omitted"].(bool),
		}

		if linkType, ok := newField["link_type"].(string); ok {
			contentfulField.LinkType = linkType
		}

		if validations, ok := newField["validations"].([]interface{}); ok {
			parsedValidations, _ := contentful.ParseValidations(validations)

			contentfulField.Validations = parsedValidations
		}

		if items := processItems(newField["items"].(*schema.Set)); items != nil {
			contentfulField.Items = items
		}

		existingFields = append(existingFields, contentfulField)
	}

	return existingFields, deletedFields
}

func processItems(fieldItems *schema.Set) *contentful.FieldTypeArrayItem {
	var items *contentful.FieldTypeArrayItem

	for _, i := range fieldItems.List() {
		item := i.(map[string]interface{})

		var validations []contentful.FieldValidation

		if fieldValidations, ok := item["validations"].([]interface{}); ok {
			validations, _ = contentful.ParseValidations(fieldValidations)
		}

		items = &contentful.FieldTypeArrayItem{
			Type:        item["type"].(string),
			Validations: validations,
			LinkType:    item["link_type"].(string),
		}
	}
	return items
}
