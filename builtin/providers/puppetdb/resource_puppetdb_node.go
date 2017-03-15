package puppetdb

import (
	"encoding/json"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePuppetDBNode() *schema.Resource {
	return &schema.Resource{
		Create: resourcePuppetDBNodeCreate,
		Read:   resourcePuppetDBNodeRead,
		Delete: resourcePuppetDBNodeDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"certname": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"deactivated": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"expired": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"cached_catalog_status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"catalog_environment": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"facts_environment": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"report_environment": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"catalog_timestamp": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"facts_timestamp": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"report_timestamp": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"latest_report_corrective_change": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"latest_report_hash": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"latest_report_noop": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"latest_report_noop_pending": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"latest_report_status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourcePuppetDBNodeCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating PuppetDB node: %s", d.Id())

	certname := d.Get("certname").(string)

	client := meta.(*PuppetDBClient)
	_, err := client.Query("query/v4/nodes/"+certname, "GET", "")
	if err != nil {
		return err
	}

	d.SetId(certname)
	return resourcePuppetDBNodeRead(d, meta)
}

func resourcePuppetDBNodeRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Refreshing PuppetDB Node: %s", d.Id())

	certname := d.Get("certname").(string)
	client := meta.(*PuppetDBClient)
	pdbResp, err := client.Query("query/v4/nodes/"+certname, "GET", "")
	if err != nil {
		return err
	}

	d.Set("deactivated", pdbResp.Deactivated)
	d.Set("expired", pdbResp.Expired)
	d.Set("cached_catalog_status", pdbResp.CachedCatalogStatus)
	d.Set("catalog_environment", pdbResp.CatalogEnvironment)
	d.Set("facts_environment", pdbResp.FactsEnvironment)
	d.Set("report_environment", pdbResp.ReportEnvironment)
	d.Set("catalog_timestamp", pdbResp.CatalogTimestamp)
	d.Set("facts_timestamp", pdbResp.FactsTimestamp)
	d.Set("report_timestamp", pdbResp.ReportTimestamp)
	d.Set("latest_report_corrective_change", pdbResp.LatestReportCorrectiveChange)
	d.Set("latest_report_hash", pdbResp.LatestReportHash)
	d.Set("latest_report_noop", pdbResp.LatestReportNoop)
	d.Set("latest_report_noop_pending", pdbResp.LatestReportNoopPending)
	d.Set("latest_report_status", pdbResp.LatestReportStatus)

	return nil
}

func resourcePuppetDBNodeDelete(d *schema.ResourceData, meta interface{}) (err error) {
	log.Printf("[INFO] Deactivating PuppetDB Node: %s", d.Id())

	certname := d.Get("certname").(string)
	client := meta.(*PuppetDBClient)

	payload := commandsPayload{
		Command: "deactivate node",
		Version: 3,
		Payload: map[string]string{"certname": certname},
	}

	stringPayload, err := json.Marshal(&payload)
	if err != nil {
		return
	}
	_, err = client.Query("cmd/v1", "POST", string(stringPayload))
	if err != nil {
		return
	}

	d.SetId("")
	return nil
}
