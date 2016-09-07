package nsone

import (
	"github.com/hashicorp/terraform/helper/schema"

	nsone "gopkg.in/ns1/ns1-go.v2/rest"
)

func dataSourceResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"sourcetype": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
		Create: DataSourceCreate,
		Read:   DataSourceRead,
		Update: DataSourceUpdate,
		Delete: DataSourceDelete,
	}
}

func dataSourceToResourceData(d *schema.ResourceData, ds *nsone.DataSource) {
	d.SetId(ds.Id)
	d.Set("name", ds.Name)
	d.Set("sourcetype", ds.SourceType)
}

// DataSourceCreate creates an ns1 datasource
func DataSourceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	ds := nsone.NewDataSource(d.Get("name").(string), d.Get("sourcetype").(string))
	if err := client.CreateDataSource(ds); err != nil {
		return err
	}
	dataSourceToResourceData(d, ds)
	return nil
}

// DataSourceRead fetches info for the given datasource from ns1
func DataSourceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	ds, err := client.GetDataSource(d.Id())
	if err != nil {
		return err
	}
	dataSourceToResourceData(d, ds)
	return nil
}

// DataSourceDelete deteltes the given datasource from ns1
func DataSourceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	err := client.DeleteDataSource(d.Id())
	d.SetId("")
	return err
}

// DataSourceUpdate updates the datasource with given parameters
func DataSourceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	ds := nsone.NewDataSource(d.Get("name").(string), d.Get("sourcetype").(string))
	ds.Id = d.Id()
	if err := client.UpdateDataSource(ds); err != nil {
		return err
	}
	dataSourceToResourceData(d, ds)
	return nil
}
