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
			"source_file": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source_content", "source_dir"},
			},
			"source_dir": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source_content", "source_file"},
			},
			"source_content": &schema.Schema{
				Type:          schema.TypeSet,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source_file", "source_dir"},
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
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["filename"].(string)))
					return hashcode.String(buf.String())
				},
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
			"output_md5": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "MD5 checksum of output file",
			},
			"output_base64sha256": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "Base64 Encoded SHA256 checksum of output file",
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

	sha1, base64sha256, err := genFileShas(outputPath)
	if err != nil {

		return fmt.Errorf("could not generate file checksum sha256: %s", err)
	}
	d.Set("output_sha", sha1)
	d.Set("output_base64sha256", base64sha256)

	md5, err := genFileMd5(outputPath)
	if err != nil {
		return fmt.Errorf("could not generate file checksum md5: %s", err)
	}
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
	} else if c, ok := d.GetOk("source_content"); ok {
		cL := c.(*schema.Set).List()
		contents := make(map[string][]byte)
		for _, c := range cL {
			sc := c.(map[string]interface{})
			contents[sc["filename"].(string)] = []byte(sc["content"].(string))
		}
		if err := archiver.ArchiveMultiple(contents); err != nil {
			return fmt.Errorf("error archiving content: %s", err)
		}
	} else {
		return fmt.Errorf("one of 'source_dir', 'source_file', 'source_content' must be specified")
	}
	return nil
}

func genFileShas(filename string) (string, string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", "", fmt.Errorf("could not compute file '%s' checksum: %s", filename, err)
	}
	h := sha1.New()
	h.Write([]byte(data))
	sha1 := hex.EncodeToString(h.Sum(nil))

	h256 := sha256.New()
	h256.Write([]byte(data))
	shaSum := h256.Sum(nil)
	sha256base64 := base64.StdEncoding.EncodeToString(shaSum[:])

	return sha1, sha256base64, nil
}

func genFileMd5(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("could not compute file '%s' checksum: %s", filename, err)
	}
	h := md5.New()
	h.Write([]byte(data))
	md5 := hex.EncodeToString(h.Sum(nil))

	return md5, nil
}
