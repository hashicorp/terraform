package google

import (
	"fmt"
	"github.com/22acacia/terraform-gcloud"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDataflow() *schema.Resource {
	return &schema.Resource{
		Create: resourceDataflowCreate,
		Read:   resourceDataflowRead,
		Delete: resourceDataflowDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"classpath": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"class": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"optional_args": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem:	  schema.TypeString,
			},

			"jobids": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"job_states": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataflowCleanOptionalArgs(optional_args map[string]interface{}) map[string]string {
	cleaned_opts := make(map[string]string)
	for k,v := range  optional_args {
		cleaned_opts[k] = v.(string)
	}
	return cleaned_opts
}

func resourceDataflowCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
        err := terraformGcloud.InitGcloud(config.Credentials)
	if err != nil {
		return err
	}

	optional_args := dataflowCleanOptionalArgs(d.Get("optional_args").(map[string]interface{}))
	jobids, err := terraformGcloud.CreateDataflow(d.Get("name").(string), d.Get("classpath").(string), d.Get("class").(string), config.Project, optional_args)
	if err != nil {
		return err
	}

	d.Set("jobids", jobids)
	d.SetId(d.Get("name").(string))

	err = resourceDataflowRead(d, meta)
	if err != nil {
		return err
	}

	return nil
}

func resourceDataflowRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
        err := terraformGcloud.InitGcloud(config.Credentials)
	if err != nil {
		return err
	}


	job_states := make([]string, 0)
	job_cnt := d.Get("jobid.#")
	if job_cnt != nil {
		for i := 0; i < job_cnt.(int); i++ {
			jobidkey:= fmt.Sprintf("jobid.%d", i)
			job_state, err := terraformGcloud.ReadDataflow(d.Get(jobidkey).(string))
			if err != nil {
				return err
			}
			job_states = append(job_states, job_state)
		}
	}

	d.Set("job_states", job_states)

	return nil
}

func resourceDataflowDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
        err := terraformGcloud.InitGcloud(config.Credentials)
	if err != nil {
		return err
	}

	err = resourceDataflowRead(d, meta)
	if err != nil {
		return err
	}

	failedCancel := make([]string, 0)
	job_cnt := d.Get("jobid.#")
	if job_cnt != nil {
		for i := 0; i < job_cnt.(int); i++ {
			jobidkey:= fmt.Sprintf("jobid.%d", i)
			jobstatekey := fmt.Sprintf("jobstate.%d", i)
			failedjob, err := terraformGcloud.CancelDataflow(d.Get(jobidkey).(string), d.Get(jobstatekey).(string))
			if err != nil {
				return err
			}
			if failedjob {
				failedCancel = append(failedCancel, d.Get(jobidkey).(string))
			}
		}
	}

	if len(failedCancel) > 0 {
		return fmt.Errorf("Failed to cancel the following jobs: %v", failedCancel)
	}

	d.SetId("")
	return nil
}
