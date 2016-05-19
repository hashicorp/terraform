package cobbler

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	cobbler "github.com/jtopjian/cobblerclient"
)

func resourceSnippet() *schema.Resource {
	return &schema.Resource{
		Create: resourceSnippetCreate,
		Read:   resourceSnippetRead,
		Update: resourceSnippetUpdate,
		Delete: resourceSnippetDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"body": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceSnippetCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	snippet := cobbler.Snippet{
		Name: d.Get("name").(string),
		Body: d.Get("body").(string),
	}

	log.Printf("[DEBUG] Cobbler Snippet: Create Options: %#v", snippet)

	if err := config.cobblerClient.CreateSnippet(snippet); err != nil {
		return fmt.Errorf("Cobbler Snippet: Error Creating: %s", err)
	}

	d.SetId(snippet.Name)

	return resourceSnippetRead(d, meta)
}

func resourceSnippetRead(d *schema.ResourceData, meta interface{}) error {
	// Since all attributes are required and not computed,
	// there's no reason to read.
	return nil
}

func resourceSnippetUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	snippet := cobbler.Snippet{
		Name: d.Id(),
		Body: d.Get("body").(string),
	}

	log.Printf("[DEBUG] Cobbler Snippet: Updating Snippet (%s) with options: %+v", d.Id(), snippet)

	if err := config.cobblerClient.CreateSnippet(snippet); err != nil {
		return fmt.Errorf("Cobbler Snippet: Error Updating (%s): %s", d.Id(), err)
	}

	return resourceSnippetRead(d, meta)
}

func resourceSnippetDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	if err := config.cobblerClient.DeleteSnippet(d.Id()); err != nil {
		return fmt.Errorf("Cobbler Snippet: Error Deleting (%s): %s", d.Id(), err)
	}

	return nil
}
