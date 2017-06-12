package pass

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/hashicorp/terraform/helper/schema"
)

func passPasswordResource() *schema.Resource {
	return &schema.Resource{
		Create: passPasswordResourceWrite,
		Update: passPasswordResourceWrite,
		Delete: passPasswordResourceDelete,
		Read:   passPasswordResourceRead,

		Schema: map[string]*schema.Schema{
			"path": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Full path where the pass password will be written.",
			},

			// Data is passed as JSON so that an arbitrary structure is
			// possible, rather than forcing e.g. all values to be strings.
			"data": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "JSON-encoded secret data to write.",
			},
		},
	}
}

func passPasswordResourceWrite(d *schema.ResourceData, meta interface{}) error {
	path := d.Get("path").(string)

	log.Printf("[DEBUG] Writing pass Password secret to %s", path)
	subProcess := exec.Command("pass", "insert", "-e", path)

	stdin, err := subProcess.StdinPipe()
	if err != nil {
		return fmt.Errorf("Fail to acquire stdin : %v", err)
	}
	defer stdin.Close()

	subProcess.Stdout = os.Stdout
	subProcess.Stderr = os.Stderr

	if err = subProcess.Start(); err != nil {
		return fmt.Errorf("Fail to run command : %v", err)
	}

	io.WriteString(stdin, fmt.Sprintf("%s\n", d.Get("data")))
	subProcess.Wait()

	d.SetId(path)

	return nil
}

func passPasswordResourceDelete(d *schema.ResourceData, meta interface{}) error {
	path := d.Id()

	log.Printf("[DEBUG] Deleting generic Vault from %s", path)
	exec.Command("pass", "rm", path)

	return nil
}

func passPasswordResourceRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[WARN] pass_password does not automatically refresh")
	return nil
}
