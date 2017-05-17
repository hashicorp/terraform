package scaleway

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/scaleway/scaleway-cli/pkg/api"
)

func resourceScalewayVolumeAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceScalewayVolumeAttachmentCreate,
		Read:   resourceScalewayVolumeAttachmentRead,
		Delete: resourceScalewayVolumeAttachmentDelete,
		Schema: map[string]*schema.Schema{
			"server": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"volume": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

var errVolumeAlreadyAttached = fmt.Errorf("Scaleway volume already attached")

func resourceScalewayVolumeAttachmentCreate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	scaleway.ClearCache()

	vol, err := scaleway.GetVolume(d.Get("volume").(string))
	if err != nil {
		return err
	}
	if vol.Server != nil {
		log.Printf("[DEBUG] Scaleway volume %q already attached to %q.", vol.Identifier, vol.Server.Identifier)
		return errVolumeAlreadyAttached
	}

	// guard against server shutdown/ startup race conditiond
	serverID := d.Get("server").(string)
	scalewayMutexKV.Lock(serverID)
	defer scalewayMutexKV.Unlock(serverID)

	server, err := scaleway.GetServer(serverID)
	if err != nil {
		fmt.Printf("Failed getting server: %q", err)
		return err
	}

	var startServerAgain = false
	// volumes can only be modified when the server is powered off
	if server.State != "stopped" {
		startServerAgain = true

		if err := scaleway.PostServerAction(server.Identifier, "poweroff"); err != nil {
			return err
		}
	}
	if err := waitForServerState(scaleway, server.Identifier, "stopped"); err != nil {
		return err
	}

	volumes := make(map[string]api.ScalewayVolume)
	for i, volume := range server.Volumes {
		volumes[i] = volume
	}

	volumes[fmt.Sprintf("%d", len(volumes)+1)] = *vol

	// the API request requires most volume attributes to be unset to succeed
	for k, v := range volumes {
		v.Size = 0
		v.CreationDate = ""
		v.Organization = ""
		v.ModificationDate = ""
		v.VolumeType = ""
		v.Server = nil
		v.ExportURI = ""

		volumes[k] = v
	}

	if err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		scaleway.ClearCache()

		var req = api.ScalewayServerPatchDefinition{
			Volumes: &volumes,
		}
		mu.Lock()
		err := scaleway.PatchServer(serverID, req)
		mu.Unlock()

		if err == nil {
			return nil
		}

		if serr, ok := err.(api.ScalewayAPIError); ok {
			log.Printf("[DEBUG] Error patching server: %q\n", serr.APIMessage)

			if serr.StatusCode == 400 {
				return resource.RetryableError(fmt.Errorf("Waiting for server update to succeed: %q", serr.APIMessage))
			}
		}

		return resource.NonRetryableError(err)
	}); err != nil {
		return err
	}

	if startServerAgain {
		if err := scaleway.PostServerAction(serverID, "poweron"); err != nil {
			return err
		}
		if err := waitForServerState(scaleway, serverID, "running"); err != nil {
			return err
		}
	}

	d.SetId(fmt.Sprintf("scaleway-server:%s/volume/%s", serverID, d.Get("volume").(string)))

	return resourceScalewayVolumeAttachmentRead(d, m)
}

func resourceScalewayVolumeAttachmentRead(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	scaleway.ClearCache()

	server, err := scaleway.GetServer(d.Get("server").(string))
	if err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			log.Printf("[DEBUG] Error reading server: %q\n", serr.APIMessage)

			if serr.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	if _, err := scaleway.GetVolume(d.Get("volume").(string)); err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			log.Printf("[DEBUG] Error reading volume: %q\n", serr.APIMessage)

			if serr.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	for _, volume := range server.Volumes {
		if volume.Identifier == d.Get("volume").(string) {
			return nil
		}
	}

	log.Printf("[DEBUG] Volume %q not attached to server %q\n", d.Get("volume").(string), d.Get("server").(string))
	d.SetId("")
	return nil
}

func resourceScalewayVolumeAttachmentDelete(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	scaleway.ClearCache()

	mu.Lock()
	defer mu.Unlock()

	var startServerAgain = false

	// guard against server shutdown/ startup race conditiond
	serverID := d.Get("server").(string)
	scalewayMutexKV.Lock(serverID)
	defer scalewayMutexKV.Unlock(serverID)

	server, err := scaleway.GetServer(serverID)
	if err != nil {
		return err
	}

	// volumes can only be modified when the server is powered off
	if server.State != "stopped" {
		startServerAgain = true
		if err := scaleway.PostServerAction(server.Identifier, "poweroff"); err != nil {
			return err
		}
	}
	if err := waitForServerState(scaleway, server.Identifier, "stopped"); err != nil {
		return err
	}

	volumes := make(map[string]api.ScalewayVolume)
	for _, volume := range server.Volumes {
		if volume.Identifier != d.Get("volume").(string) {
			volumes[fmt.Sprintf("%d", len(volumes))] = volume
		}
	}

	// the API request requires most volume attributes to be unset to succeed
	for k, v := range volumes {
		v.Size = 0
		v.CreationDate = ""
		v.Organization = ""
		v.ModificationDate = ""
		v.VolumeType = ""
		v.Server = nil
		v.ExportURI = ""

		volumes[k] = v
	}

	if err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		scaleway.ClearCache()

		var req = api.ScalewayServerPatchDefinition{
			Volumes: &volumes,
		}
		mu.Lock()
		err := scaleway.PatchServer(serverID, req)
		mu.Unlock()

		if err == nil {
			return nil
		}

		if serr, ok := err.(api.ScalewayAPIError); ok {
			log.Printf("[DEBUG] Error patching server: %q\n", serr.APIMessage)

			if serr.StatusCode == 400 {
				return resource.RetryableError(fmt.Errorf("Waiting for server update to succeed: %q", serr.APIMessage))
			}
		}

		return resource.NonRetryableError(err)
	}); err != nil {
		return err
	}

	if startServerAgain {
		if err := scaleway.PostServerAction(serverID, "poweron"); err != nil {
			return err
		}
		if err := waitForServerState(scaleway, serverID, "running"); err != nil {
			return err
		}
	}

	d.SetId("")

	return nil
}
