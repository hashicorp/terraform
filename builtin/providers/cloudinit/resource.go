package cloudinit

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resource() *schema.Resource {
	return &schema.Resource{
		Create: Create,
		Delete: Delete,
		Exists: Exists,
		Read:   Read,

		Schema: map[string]*schema.Schema{
			"part": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"order": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(int)

								if value < 0 {
									errors = append(errors, fmt.Errorf("Order must be zero or greater"))
								}

								return
							},
						},
						"content_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)

								if _, supported := supportedContentTypes[value]; !supported {
									errors = append(errors, fmt.Errorf("Unsupported content type: %s", v))
								}

								return
							},
						},
						"content": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"filename": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"merge_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceCloudInitConfigPartHash,
			},
			"gzip": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},
			"base64_encode": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},
			"rendered": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "rendered cloudinit configuration",
			},
		},
	}
}

func Create(d *schema.ResourceData, meta interface{}) error {
	rendered, err := render(d)
	if err != nil {
		return err
	}

	d.Set("rendered", rendered)
	d.SetId(strconv.Itoa(hashcode.String(rendered)))
	return nil
}

func Delete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

func Exists(d *schema.ResourceData, meta interface{}) (bool, error) {
	rendered, err := render(d)
	if err != nil {
		return false, err
	}

	return strconv.Itoa(hashcode.String(rendered)) == d.Id(), nil
}

func Read(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func render(d *schema.ResourceData) (string, error) {
	gzipOutput := d.Get("gzip").(bool)
	base64Output := d.Get("base64_encode").(bool)

	partsValue, hasParts := d.GetOk("part")
	if !hasParts {
		return "", fmt.Errorf("No parts found in the cloudinit resource declaration")
	}
	partsSet, ok := partsValue.(*schema.Set)
	if !ok {
		return "", fmt.Errorf("Parts must be a set TODO error message")
	}

	cloudInitParts := make(cloudInitParts, partsSet.Len())
	for i, v := range partsSet.List() {
		partMap := v.(map[string]interface{})
		part := cloudInitPart{}
		if v, ok := partMap["order"]; ok {
			part.Order = v.(int)
		}
		if v, ok := partMap["content_type"]; ok {
			part.ContentType = v.(string)
		}
		if v, ok := partMap["content"]; ok {
			part.Content = v.(string)
		}
		if v, ok := partMap["merge_type"]; ok {
			part.MergeType = v.(string)
		}
		if v, ok := partMap["filename"]; ok {
			part.Filename = v.(string)
		}
		cloudInitParts[i] = part
	}
	sort.Sort(cloudInitParts)

	var buffer bytes.Buffer

	var err error
	if gzipOutput {
		gzipWriter := gzip.NewWriter(&buffer)
		err = renderPartsToWriter(cloudInitParts, gzipWriter)
		gzipWriter.Close()
	} else {
		err = renderPartsToWriter(cloudInitParts, &buffer)
	}
	if err != nil {
		return "", err
	}

	output := ""
	if base64Output {
		output = base64.StdEncoding.EncodeToString(buffer.Bytes())
	} else {
		output = buffer.String()
	}

	return output, nil
}

func renderPartsToWriter(parts cloudInitParts, writer io.Writer) error {
	mimeWriter := multipart.NewWriter(writer)
	defer mimeWriter.Close()

	writer.Write([]byte(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\n", mimeWriter.Boundary())))

	for _, part := range parts {
		header := textproto.MIMEHeader{}
		if part.ContentType == "" {
			header.Set("Content-Type", "text/plain")
		} else {
			header.Set("Content-Type", part.ContentType)
		}

		if part.Filename != "" {
			header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, part.Filename))
		}

		if part.MergeType != "" {
			header.Set("X-Merge-Type", part.MergeType)
		}

		partWriter, err := mimeWriter.CreatePart(header)
		if err != nil {
			return err
		}

		_, err = partWriter.Write([]byte(part.Content))
		if err != nil {
			return err
		}
	}

	return nil
}

func resourceCloudInitConfigPartHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["order"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}
	if v, ok := m["content_type"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["content"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["merge_type"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	t := hashcode.String(buf.String())
	fmt.Println("hashcode is", t)
	return t
}

type cloudInitPart struct {
	Order       int
	ContentType string
	MergeType   string
	Filename    string
	Content     string
}

type cloudInitParts []cloudInitPart

func (slice cloudInitParts) Len() int {
	return len(slice)
}

func (slice cloudInitParts) Less(i, j int) bool {
	return slice[i].Order < slice[j].Order
}

func (slice cloudInitParts) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

var supportedContentTypes = map[string]bool{
	"text/x-include-once-url":   true,
	"text/x-include-url":        true,
	"text/cloud-config-archive": true,
	"text/upstart-job":          true,
	"text/cloud-config":         true,
	"text/part-handler":         true,
	"text/x-shellscript":        true,
	"text/cloud-boothook":       true,
}
