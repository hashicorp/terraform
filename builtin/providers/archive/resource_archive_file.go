package archive

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArchiveFile() *schema.Resource {
	return &schema.Resource{
		Create: resourceArchiveFileCreate,
		Read:   resourceArchiveFileRead,
		Update: resourceArchiveFileUpdate,
		Delete: resourceArchiveFileDelete,
		Exists: resourceArchiveFileExists,

		Schema: map[string]*schema.Schema{
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"source_content": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source_file", "source_dir"},
			},
			"source_content_filename": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source_file", "source_dir"},
			},
			"source_file": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source_content", "source_content_filename", "source_dir"},
			},
			"source_dir": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source_content", "source_content_filename", "source_file"},
			},
			"output_path": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"output_size": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
				ForceNew: true,
			},
			"output_sha": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "SHA1 checksum of output file",
			},
		},
	}
}

func resourceArchiveFileCreate(d *schema.ResourceData, meta interface{}) error {
	if err := resourceArchiveFileUpdate(d, meta); err != nil {
		return err
	}
	return resourceArchiveFileRead(d, meta)
}

func resourceArchiveFileRead(d *schema.ResourceData, meta interface{}) error {
	output_path := d.Get("output_path").(string)
	fi, err := os.Stat(output_path)
	if os.IsNotExist(err) {
		d.SetId("")
		d.MarkNewResource()
		return nil
	}

	sha, err := genFileSha1(output_path)
	if err != nil {
		return fmt.Errorf("could not generate file checksum sha: %s", err)
	}
	d.Set("output_sha", sha)
	d.Set("output_size", fi.Size())
	d.SetId(d.Get("output_sha").(string))

	return nil
}

func resourceArchiveFileUpdate(d *schema.ResourceData, meta interface{}) error {
	archiveType := d.Get("type").(string)
	output_path := d.Get("output_path").(string)

	outputDirectory := path.Dir(output_path)
	if outputDirectory != "" {
		if _, err := os.Stat(outputDirectory); err != nil {
			if err := os.MkdirAll(outputDirectory, 755); err != nil {
				return err
			}
		}
	}

	archiver := getArchiver(archiveType, output_path)
	if archiver == nil {
		return fmt.Errorf("archive type not supported: %s", archiveType)
	}

	if dir, ok := d.GetOk("source_dir"); ok {
		if err := archiver.ArchiveDir(dir.(string)); err != nil {
			return fmt.Errorf("error archiving directory: %s", err)
		}
	} else if file, ok := d.GetOk("source_file"); ok {
		if err := archiver.ArchiveFile(file.(string)); err != nil {
			return fmt.Errorf("error archiving file: %s", err)
		}
	} else if filename, ok := d.GetOk("source_content_filename"); ok {
		content := d.Get("source_content").(string)
		if err := archiver.ArchiveContent([]byte(content), filename.(string)); err != nil {
			return fmt.Errorf("error archiving content: %s", err)
		}
	} else {
		return fmt.Errorf("one of 'source_dir', 'source_file', 'source_content_filename' must be specified")
	}

	// Generate archived file stats
	fi, err := os.Stat(output_path)
	if err != nil {
		return err
	}

	sha, err := genFileSha1(output_path)
	if err != nil {
		return fmt.Errorf("could not generate file checksum sha: %s", err)
	}
	d.Set("output_sha", sha)
	d.Set("output_size", fi.Size())
	d.SetId(d.Get("output_sha").(string))

	return nil
}

func resourceArchiveFileDelete(d *schema.ResourceData, meta interface{}) error {
	output_path := d.Get("output_path").(string)
	if _, err := os.Stat(output_path); os.IsNotExist(err) {
		return nil
	}

	if err := os.Remove(output_path); err != nil {
		return fmt.Errorf("could not delete zip file %q: %s", output_path, err)
	}

	return nil
}

func resourceArchiveFileExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	output_path := d.Get("output_path").(string)
	_, err := os.Stat(output_path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func genFileSha1(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("could not compute file '%s' checksum: %s", filename, err)
	}
	h := sha1.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil)), nil
}
