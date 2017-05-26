package atlas

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/atlas-go/v1"
	"github.com/hashicorp/terraform/helper/schema"
)

var (
	// saneMetaKey is used to sanitize the metadata keys so that
	// they can be accessed as a variable interpolation from TF
	saneMetaKey = regexp.MustCompile("[^a-zA-Z0-9-_]")
)

func resourceArtifact() *schema.Resource {
	return &schema.Resource{
		Create: resourceArtifactRead,
		Read:   resourceArtifactRead,
		Delete: resourceArtifactDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:       schema.TypeString,
				Required:   true,
				ForceNew:   true,
				Deprecated: `atlas_artifact is now deprecated. Use the Atlas Artifact Data Source instead. See https://www.terraform.io/docs/providers/terraform-enterprise/d/artifact.html`,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"build": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"metadata_keys": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"file_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"metadata_full": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},

			"slug": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"version_real": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceArtifactRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*atlas.Client)

	// Parse the slug from the name given of the artifact since the API
	// expects these to be split.
	user, name, err := atlas.ParseSlug(d.Get("name").(string))
	if err != nil {
		return err
	}

	// Filter by version or build if given
	var build, version string
	if v, ok := d.GetOk("version"); ok {
		version = v.(string)
	} else if b, ok := d.GetOk("build"); ok {
		build = b.(string)
	}

	// If we have neither, default to latest version
	if build == "" && version == "" {
		version = "latest"
	}

	// Compile the metadata search params
	md := make(map[string]string)
	for _, v := range d.Get("metadata_keys").(*schema.Set).List() {
		md[v.(string)] = atlas.MetadataAnyValue
	}
	for k, v := range d.Get("metadata").(map[string]interface{}) {
		md[k] = v.(string)
	}

	// Do the search!
	vs, err := client.ArtifactSearch(&atlas.ArtifactSearchOpts{
		User:     user,
		Name:     name,
		Type:     d.Get("type").(string),
		Build:    build,
		Version:  version,
		Metadata: md,
	})
	if err != nil {
		return fmt.Errorf(
			"Error searching for artifact '%s/%s': %s",
			user, name, err)
	}

	if len(vs) == 0 {
		return fmt.Errorf("No matching artifact for '%s/%s'", user, name)
	} else if len(vs) > 1 {
		return fmt.Errorf(
			"Got %d results for '%s/%s', only one is allowed",
			len(vs), user, name)
	}
	v := vs[0]

	d.SetId(v.ID)
	if v.ID == "" {
		d.SetId(fmt.Sprintf("%s %d", v.Tag, v.Version))
	}
	d.Set("version_real", v.Version)
	d.Set("metadata_full", cleanMetadata(v.Metadata))
	d.Set("slug", v.Slug)

	d.Set("file_url", "")
	if u, err := client.ArtifactFileURL(v); err != nil {
		return fmt.Errorf(
			"Error reading file URL: %s", err)
	} else if u != nil {
		d.Set("file_url", u.String())
	}

	return nil
}

func resourceArtifactDelete(d *schema.ResourceData, meta interface{}) error {
	// This just always succeeds since this is a readonly element.
	d.SetId("")
	return nil
}

// cleanMetadata is used to ensure the metadata is accessible as
// a variable by doing a simple re-write.
func cleanMetadata(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		sane := saneMetaKey.ReplaceAllString(k, "-")
		out[sane] = v
	}
	return out
}
