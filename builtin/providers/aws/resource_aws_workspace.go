package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directoryservice"
	"github.com/aws/aws-sdk-go/service/workspaces"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWorkspace() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWorkspaceCreate,
		Read:   resourceAwsWorkspaceRead,
		Update: resourceAwsWorkspaceUpdate,
		Delete: resourceAwsWorkspaceDelete,

		Schema: map[string]*schema.Schema{
			"bundle_name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"bundle_id": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
			},

			"directory_id": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"root_volume_encryption": &schema.Schema{
				Type:     schema.TypeBool,
				ForceNew: true,
				Optional: true,
				Default:  false,
			},

			"user_name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"user_volume_encryption": &schema.Schema{
				Type:     schema.TypeBool,
				ForceNew: true,
				Optional: true,
				Default:  false,
			},

			"volume_encryption_key": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsWorkspaceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).workspacesconn

	params := workspaces.WorkspaceRequest{
		UserName: aws.String(d.Get("user_name").(string)),
	}

	if v, ok := d.GetOk("bundle_id"); ok {
		params.BundleId = aws.String(v.(string))
	} else {
		// Only looking up from the default provided AMAZON bundles at this point
		foundBundles, err := findWorkspacesBundle("AMAZON", d.Get("bundle_name").(string), meta)

		if err != nil {
			return fmt.Errorf("Error finding bundle: %q", err)
		}

		params.BundleId = foundBundles[0].BundleId
	}

	if v, ok := d.GetOk("root_volume_encryption"); ok {
		params.RootVolumeEncryptionEnabled = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("user_volume_encryption"); ok {
		params.UserVolumeEncryptionEnabled = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("volume_encryption_key"); ok {
		params.VolumeEncryptionKey = aws.String(v.(string))
	}

	foundDirectory, err := findDirectoryById(d.Get("directory_id").(string), meta)

	if err != nil {
		return fmt.Errorf("[ERROR] Error finding directory: %q", err)
	}

	params.DirectoryId = foundDirectory

	var ws_request []*workspaces.WorkspaceRequest
	ws_request = append(ws_request, &params)

	createOpts := &workspaces.CreateWorkspacesInput{
		Workspaces: ws_request,
	}

	ws_resp, err := conn.CreateWorkspaces(createOpts)

	if err != nil {
		log.Printf("[ERROR] Error during creation of Workspace: %q", err.Error())
		return err
	}

	if (ws_resp.FailedRequests == nil) || (len(ws_resp.FailedRequests) == 0) {
		log.Printf("[ERROR] Workspace FailedRequests is not empty - %s", ws_resp.FailedRequests[0].ErrorMessage)
		return nil
	}

	if (ws_resp.PendingRequests == nil) || (len(ws_resp.PendingRequests) == 0) {
		log.Printf("[ERROR] Error during the creation of Workspace - no PendingRequests avaliable")
		return nil
	}

	ws := ws_resp.PendingRequests[0]
	d.Set("id", *ws.WorkspaceId)

	// Wait for creation
	log.Printf("[DEBUG] Waiting for Workspace (%q) to become available", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Pending", "Rebooting", "Rebuilding"},
		Target:  []string{"Avaliable"},
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeWorkspaces(&workspaces.DescribeWorkspacesInput{
				WorkspaceIds: []*string{aws.String(d.Id())},
			})
			if err != nil {
				log.Printf("[ERROR] Error during creation of WS: %q", err.Error())
				return nil, "", err
			}

			ws := resp.Workspaces[0]
			log.Printf("[DEBUG] Creation of Workspace %q is in following state: %q.",
				d.Id(), *ws.State)
			return ws, *ws.State, nil
		},
		Timeout: 30 * time.Minute,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"[ERROR] Error waiting for Workspace (%s) to become available: %s",
			d.Id(), err)
	}

	// Update if we need to
	return resourceAwsWorkspaceUpdate(d, meta)
}

func resourceAwsWorkspaceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).workspacesconn

	resp, err := conn.DescribeWorkspaces(&workspaces.DescribeWorkspacesInput{
		WorkspaceIds: []*string{aws.String(d.Id())},
	})

	if err != nil {
		return fmt.Errorf("[ERROR] Error finding Workspace: %q", err)
	}

	// If nothing was found, then return no state
	if len(resp.Workspaces) == 0 {
		log.Printf("[INFO]: No workspace was found, removing from state")
		d.SetId("")
		return nil
	}

	workspace := resp.Workspaces[0]

	d.Set("bundle_id", workspace.BundleId)
	d.Set("directory_id", workspace.DirectoryId)
	d.Set("user_name", workspace.UserName)

	if workspace.RootVolumeEncryptionEnabled != nil {
		d.Set("root_volume_encryption", workspace.RootVolumeEncryptionEnabled)
	}

	if workspace.UserVolumeEncryptionEnabled != nil {
		d.Set("user_volume_encryption", workspace.UserVolumeEncryptionEnabled)
	}

	if workspace.VolumeEncryptionKey != nil {
		d.Set("volume_encryption_key", workspace.VolumeEncryptionKey)
	}

	tags, err := conn.DescribeTags(&workspaces.DescribeTagsInput{
		ResourceId: aws.String(d.Id()),
	})

	d.Set("computer_name", workspace.ComputerName)
	d.Set("ip_address", workspace.IpAddress)
	d.Set("subnetId", workspace.SubnetId)

	d.Set("tags", tagsToMapWorkspace(tags.TagList))

	return nil
}

func resourceAwsWorkspaceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).workspacesconn

	if err := setTagsWorkspace(conn, d); err != nil {
		return err
	}

	d.SetPartial("tags")

	return resourceAwsWorkspaceRead(d, meta)
}

func resourceAwsWorkspaceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).workspacesconn

	_, err := conn.TerminateWorkspaces(&workspaces.TerminateWorkspacesInput{
		TerminateWorkspaceRequests: []*workspaces.TerminateRequest{
			{
				WorkspaceId: aws.String(d.Id()),
			},
		},
	})

	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func findWorkspacesBundle(owner string, name string, meta interface{}) ([]*workspaces.WorkspaceBundle, error) {
	conn := meta.(*AWSClient).workspacesconn

	resp, err := conn.DescribeWorkspaceBundles(&workspaces.DescribeWorkspaceBundlesInput{
		Owner: aws.String(owner),
	})

	if err != nil {
		log.Printf("[ERROR] Error finding Workspace Bundle: %s", err)
		return nil, err
	}

	found := []*workspaces.WorkspaceBundle{}

	for _, s := range resp.Bundles {
		if strings.EqualFold(*s.Owner, owner) {
			found = append(found, s)
		}
	}

	return found, nil
}

func findDirectoryById(id string, meta interface{}) (*string, error) {

	dsconn := meta.(*AWSClient).dsconn
	ds_resp, err := dsconn.DescribeDirectories(&directoryservice.DescribeDirectoriesInput{
		DirectoryIds: []*string{aws.String(id)},
	})

	if err != nil {
		log.Printf("[ERROR] Error finding directory")
		return nil, err
	}

	if ds_resp.DirectoryDescriptions == nil || len(ds_resp.DirectoryDescriptions) == 0 {
		log.Printf("[ERROR] Could not find directory: %s", id)
		return nil, nil
	} else {
		log.Printf("[INFO] Found directory: %s", id)
		return &id, nil
	}
}
