package archive

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceFile() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceFileRead,

		Schema: map[string]*schema.Schema{
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"source": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"content": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"filename": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
				ConflictsWith: []string{"source_file", "source_dir", "source_content", "source_content_filename"},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["filename"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["content"].(string)))
					return hashcode.String(buf.String())
				},
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
			"output_base64sha256": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "Base64 Encoded SHA256 checksum of output file",
			},
			"output_md5": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "MD5 of output file",
			},
		},
	}
}

func dataSourceFileRead(d *schema.ResourceData, meta interface{}) error {
	outputPath := d.Get("output_path").(string)

	outputDirectory := path.Dir(outputPath)
	if outputDirectory != "" {
		if _, err := os.Stat(outputDirectory); err != nil {
			if err := os.MkdirAll(outputDirectory, 0755); err != nil {
				return err
			}
		}
	}

	if err := archive(d); err != nil {
		return err
	}

	// Generate archived file stats
	fi, err := os.Stat(outputPath)
	if err != nil {
		return err
	}

	sha1, base64sha256, md5, err := genFileShas(outputPath)
	if err != nil {

		return fmt.Errorf("could not generate file checksum sha256: %s", err)
	}
	d.Set("output_sha", sha1)
	d.Set("output_base64sha256", base64sha256)
	d.Set("output_md5", md5)

	d.Set("output_size", fi.Size())
	d.SetId(d.Get("output_sha").(string))

	return nil
}

func archive(d *schema.ResourceData) error {
	archiveType := d.Get("type").(string)
	outputPath := d.Get("output_path").(string)

	archiver := getArchiver(archiveType, outputPath)
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
	} else if v, ok := d.GetOk("source"); ok {
		vL := v.(*schema.Set).List()
		content := make(map[string][]byte)
		for _, v := range vL {
			src := v.(map[string]interface{})
			content[src["filename"].(string)] = []byte(src["content"].(string))
		}
		if err := archiver.ArchiveMultiple(content); err != nil {
			return fmt.Errorf("error archiving content: %s", err)
		}
	} else {
		return fmt.Errorf("one of 'source_dir', 'source_file', 'source_content_filename' must be specified")
	}
	return nil
}

func genFileShas(filename string) (string, string, string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", "", "", fmt.Errorf("could not compute file '%s' checksum: %s", filename, err)
	}
	h := sha1.New()
	h.Write([]byte(data))
	sha1 := hex.EncodeToString(h.Sum(nil))

	h256 := sha256.New()
	h256.Write([]byte(data))
	shaSum := h256.Sum(nil)
	sha256base64 := base64.StdEncoding.EncodeToString(shaSum[:])

	md5 := md5.New()
	md5.Write([]byte(data))
	md5Sum := hex.EncodeToString(md5.Sum(nil))

	return sha1, sha256base64, md5Sum, nil
}
