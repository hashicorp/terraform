package template

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"strconv"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceCloudinitConfig() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCloudinitConfigRead,

		Schema: map[string]*schema.Schema{
			"part": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"content_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"content": {
							Type:     schema.TypeString,
							Required: true,
						},
						"filename": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"merge_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"gzip": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"base64_encode": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"rendered": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "rendered cloudinit configuration",
			},
		},
	}
}

func dataSourceCloudinitConfigRead(d *schema.ResourceData, meta interface{}) error {
	rendered, err := renderCloudinitConfig(d)
	if err != nil {
		return err
	}

	d.Set("rendered", rendered)
	d.SetId(strconv.Itoa(hashcode.String(rendered)))
	return nil
}

func renderCloudinitConfig(d *schema.ResourceData) (string, error) {
	gzipOutput := d.Get("gzip").(bool)
	base64Output := d.Get("base64_encode").(bool)

	partsValue, hasParts := d.GetOk("part")
	if !hasParts {
		return "", fmt.Errorf("No parts found in the cloudinit resource declaration")
	}

	cloudInitParts := make(cloudInitParts, len(partsValue.([]interface{})))
	for i, v := range partsValue.([]interface{}) {
		p, castOk := v.(map[string]interface{})
		if !castOk {
			return "", fmt.Errorf("Unable to parse parts in cloudinit resource declaration")
		}

		part := cloudInitPart{}
		if p, ok := p["content_type"]; ok {
			part.ContentType = p.(string)
		}
		if p, ok := p["content"]; ok {
			part.Content = p.(string)
		}
		if p, ok := p["merge_type"]; ok {
			part.MergeType = p.(string)
		}
		if p, ok := p["filename"]; ok {
			part.Filename = p.(string)
		}
		cloudInitParts[i] = part
	}

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

	// we need to set the boundary explictly, otherwise the boundary is random
	// and this causes terraform to complain about the resource being different
	if err := mimeWriter.SetBoundary("MIMEBOUNDARY"); err != nil {
		return err
	}

	writer.Write([]byte(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\n", mimeWriter.Boundary())))
	writer.Write([]byte("MIME-Version: 1.0\r\n\r\n"))

	for _, part := range parts {
		header := textproto.MIMEHeader{}
		if part.ContentType == "" {
			header.Set("Content-Type", "text/plain")
		} else {
			header.Set("Content-Type", part.ContentType)
		}

		header.Set("MIME-Version", "1.0")
		header.Set("Content-Transfer-Encoding", "7bit")

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

type cloudInitPart struct {
	ContentType string
	MergeType   string
	Filename    string
	Content     string
}

type cloudInitParts []cloudInitPart
