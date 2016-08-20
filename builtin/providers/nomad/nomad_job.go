package nomad

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/nomad/jobspec"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceNomadJob() *schema.Resource {
	return &schema.Resource{
		Create: resourceNomadJobCreate,
		Read:   resourceNomadJobRead,
		Update: resourceNomadJobUpdate,
		Delete: resourceNomadJobDelete,
		Schema: map[string]*schema.Schema{
			"job_definition": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"allocations": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func resourceNomadJobCreate(d *schema.ResourceData, m interface{}) error {
	nomad := m.(*Client).nomad

	r := strings.NewReader(d.Get("job_definition").(string))
	jobStruct, err := jobspec.Parse(r)
	if err != nil {
		log.Printf("[DEBUG] failed to parse job_definition: %q", err)
		return err
	}
	job, err := convertStructJob(jobStruct)
	if err != nil {
		log.Printf("[DEBUG] failed to convert jobs: %q", err)
		return err
	}

	if _, _, err := nomad.Jobs().Register(job, nil); err != nil {
		log.Printf("[DEBUG] failed to register job: %q", err)
		return err
	}

	allocs, _, err := nomad.Jobs().Allocations(job.Name, nil)
	if err != nil {
		log.Printf("[DEBUG] failed to fetch allocations: %q", err)
		return err
	}

	d.SetId(fmt.Sprintf("nomad-job-%s", job.Name))
	d.Set("name", job.Name)
	allocations := []string{}
	for _, alloc := range allocs {
		allocations = append(allocations, alloc.ID)
	}
	d.Set("allocations", allocations)

	return nil
}

func resourceNomadJobRead(d *schema.ResourceData, m interface{}) error {
	// nomad := m.(*Client).nomad
	return nil
}

func resourceNomadJobUpdate(d *schema.ResourceData, m interface{}) error {
	// nomad := m.(*Client).nomad
	return resourceNomadJobRead(d, m)
}

func resourceNomadJobDelete(d *schema.ResourceData, m interface{}) error {
	// nomad := m.(*Client).nomad
	return nil
}
