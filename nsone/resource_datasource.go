package nsone

import (
	"fmt"
	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/schema"
	"regexp"
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
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !regexp.MustCompile(`^(monitor_us)$`).MatchString(value) {
						es = append(es, fmt.Errorf(
							"only monitor_us allowed in %q", k))
					}
					return
				},
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

func DataSourceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	ds := nsone.NewDataSource(d.Get("name").(string), d.Get("sourcetype").(string))
	err := client.CreateDataSource(ds)
	if err != nil {
		return err
	}
	dataSourceToResourceData(d, ds)
	return nil
}

func DataSourceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	ds, err := client.GetDataSource(d.Id())
	if err != nil {
		return err
	}
	dataSourceToResourceData(d, ds)
	return nil
}

func DataSourceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	err := client.DeleteDataSource(d.Id())
	d.SetId("")
	return err
}

func DataSourceUpdate(d *schema.ResourceData, meta interface{}) error {
	panic("Update not implemented")
	return nil
}
