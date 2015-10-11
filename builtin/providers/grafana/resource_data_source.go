package grafana

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"

	gapi "github.com/apparentlymart/go-grafana-api"
)

func ResourceDataSource() *schema.Resource {
	return &schema.Resource{
		Create: CreateDataSource,
		Update: UpdateDataSource,
		Delete: DeleteDataSource,
		Read:   ReadDataSource,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"url": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"is_default": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"basic_auth_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"basic_auth_username": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"basic_auth_password": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"username": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"password": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"database_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"access_mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "proxy",
			},
		},
	}
}

func CreateDataSource(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	dataSource, err := makeDataSource(d)
	if err != nil {
		return err
	}

	id, err := client.NewDataSource(dataSource)
	if err != nil {
		return err
	}

	d.SetId(strconv.FormatInt(id, 10))

	return ReadDataSource(d, meta)
}

func UpdateDataSource(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	dataSource, err := makeDataSource(d)
	if err != nil {
		return err
	}

	return client.UpdateDataSource(dataSource)
}

func ReadDataSource(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	idStr := d.Id()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid id: %#v", idStr)
	}

	dataSource, err := client.DataSource(id)
	if err != nil {
		return err
	}

	d.Set("id", dataSource.Id)
	d.Set("access_mode", dataSource.Access)
	d.Set("basic_auth_enabled", dataSource.BasicAuth)
	d.Set("basic_auth_username", dataSource.BasicAuthUser)
	d.Set("basic_auth_password", dataSource.BasicAuthPassword)
	d.Set("database_name", dataSource.Database)
	d.Set("is_default", dataSource.IsDefault)
	d.Set("name", dataSource.Name)
	d.Set("password", dataSource.Password)
	d.Set("type", dataSource.Type)
	d.Set("url", dataSource.URL)
	d.Set("username", dataSource.User)

	return nil
}

func DeleteDataSource(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	idStr := d.Id()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid id: %#v", idStr)
	}

	return client.DeleteDataSource(id)
}

func makeDataSource(d *schema.ResourceData) (*gapi.DataSource, error) {
	idStr := d.Id()
	var id int64
	var err error
	if idStr != "" {
		id, err = strconv.ParseInt(idStr, 10, 64)
	}

	return &gapi.DataSource{
		Id:                id,
		Name:              d.Get("name").(string),
		Type:              d.Get("type").(string),
		URL:               d.Get("url").(string),
		Access:            d.Get("access_mode").(string),
		Database:          d.Get("database_name").(string),
		User:              d.Get("username").(string),
		Password:          d.Get("password").(string),
		IsDefault:         d.Get("is_default").(bool),
		BasicAuth:         d.Get("basic_auth_enabled").(bool),
		BasicAuthUser:     d.Get("basic_auth_username").(string),
		BasicAuthPassword: d.Get("basic_auth_password").(string),
	}, err
}
