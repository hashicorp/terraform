package nomad

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/jobspec"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceJob() *schema.Resource {
	return &schema.Resource{
		Create: resourceJobRegister,
		Update: resourceJobRegister,
		Delete: resourceJobDeregister,
		Read:   resourceJobRead,
		Exists: resourceJobExists,

		Schema: map[string]*schema.Schema{
			"jobspec": {
				Description:      "Job specification. If you want to point to a file use the file() function.",
				Required:         true,
				Type:             schema.TypeString,
				DiffSuppressFunc: jobspecDiffSuppress,
			},

			"deregister_on_destroy": {
				Description: "If true, the job will be deregistered on destroy.",
				Optional:    true,
				Default:     true,
				Type:        schema.TypeBool,
			},

			"deregister_on_id_change": {
				Description: "If true, the job will be deregistered when the job ID changes.",
				Optional:    true,
				Default:     true,
				Type:        schema.TypeBool,
			},
		},
	}
}

func resourceJobRegister(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	// Get the jobspec itself
	jobspecRaw := d.Get("jobspec").(string)

	// Parse it
	jobspecStruct, err := jobspec.Parse(strings.NewReader(jobspecRaw))
	if err != nil {
		return fmt.Errorf("error parsing jobspec: %s", err)
	}

	// Initialize and validate
	jobspecStruct.Canonicalize()
	if err := jobspecStruct.Validate(); err != nil {
		return fmt.Errorf("Error validating job: %v", err)
	}

	// If we have an ID and its not equal to this jobspec, then we
	// have to deregister the old job before we register the new job.
	prevId := d.Id()
	if !d.Get("deregister_on_id_change").(bool) {
		// If we aren't deregistering on ID change, just pretend we
		// don't have a prior ID.
		prevId = ""
	}
	if prevId != "" && prevId != jobspecStruct.ID {
		log.Printf(
			"[INFO] Deregistering %q before registering %q",
			prevId, jobspecStruct.ID)

		log.Printf("[DEBUG] Deregistering job: %q", prevId)
		_, _, err := client.Jobs().Deregister(prevId, nil)
		if err != nil {
			return fmt.Errorf(
				"error deregistering previous job %q "+
					"before registering new job %q: %s",
				prevId, jobspecStruct.ID, err)
		}

		// Success! Clear our state.
		d.SetId("")
	}

	// Convert it so that we can use it with the API
	jobspecAPI, err := convertStructJob(jobspecStruct)
	if err != nil {
		return fmt.Errorf("error converting jobspec: %s", err)
	}

	// Register the job
	_, _, err = client.Jobs().Register(jobspecAPI, nil)
	if err != nil {
		return fmt.Errorf("error applying jobspec: %s", err)
	}

	d.SetId(jobspecAPI.ID)

	return nil
}

func resourceJobDeregister(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	// If deregistration is disabled, then do nothing
	if !d.Get("deregister_on_destroy").(bool) {
		log.Printf(
			"[WARN] Job %q will not deregister since 'deregister_on_destroy'"+
				" is false", d.Id())
		return nil
	}

	id := d.Id()
	log.Printf("[DEBUG] Deregistering job: %q", id)
	_, _, err := client.Jobs().Deregister(id, nil)
	if err != nil {
		return fmt.Errorf("error deregistering job: %s", err)
	}

	return nil
}

func resourceJobExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*api.Client)

	id := d.Id()
	log.Printf("[DEBUG] Checking if job exists: %q", id)
	_, _, err := client.Jobs().Info(id, nil)
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}

		return true, fmt.Errorf("error checking for job: %#v", err)
	}

	return true, nil
}

func resourceJobRead(d *schema.ResourceData, meta interface{}) error {
	// We don't do anything at the moment. Exists is used to
	// remove non-existent jobs but read doesn't have to do anything.
	return nil
}

// convertStructJob is used to take a *structs.Job and convert it to an *api.Job.
//
// This is unfortunate but it is how Nomad itself does it (this is copied
// line for line from Nomad). We'll mimic them exactly to get this done.
func convertStructJob(in *structs.Job) (*api.Job, error) {
	gob.Register([]map[string]interface{}{})
	gob.Register([]interface{}{})
	var apiJob *api.Job
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(in); err != nil {
		return nil, err
	}
	if err := gob.NewDecoder(buf).Decode(&apiJob); err != nil {
		return nil, err
	}
	return apiJob, nil
}

// jobspecDiffSuppress is the DiffSuppressFunc used by the schema to
// check if two jobspecs are equal.
func jobspecDiffSuppress(k, old, new string, d *schema.ResourceData) bool {
	// Parse the old job
	oldJob, err := jobspec.Parse(strings.NewReader(old))
	if err != nil {
		return false
	}

	// Parse the new job
	newJob, err := jobspec.Parse(strings.NewReader(new))
	if err != nil {
		return false
	}

	// Init
	oldJob.Canonicalize()
	newJob.Canonicalize()

	// Check for jobspec equality
	return reflect.DeepEqual(oldJob, newJob)
}
