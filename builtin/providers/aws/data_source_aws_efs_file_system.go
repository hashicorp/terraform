package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsEfsFileSystem() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEfsFileSystemRead,

		Schema: map[string]*schema.Schema{
			"creation_token": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateMaxLength(64),
			},
			"file_system_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"performance_mode": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsEfsFileSystemRead(d *schema.ResourceData, meta interface{}) error {
	efsconn := meta.(*AWSClient).efsconn

	describeEfsOpts := &efs.DescribeFileSystemsInput{}

	if v, ok := d.GetOk("creation_token"); ok {
		describeEfsOpts.CreationToken = aws.String(v.(string))
	}

	if v, ok := d.GetOk("file_system_id"); ok {
		describeEfsOpts.FileSystemId = aws.String(v.(string))
	}

	describeResp, err := efsconn.DescribeFileSystems(describeEfsOpts)
	if err != nil {
		return errwrap.Wrapf("Error retrieving EFS: {{err}}", err)
	}
	if len(describeResp.FileSystems) != 1 {
		return fmt.Errorf("Search returned %d results, please revise so only one is returned", len(describeResp.FileSystems))
	}

	d.SetId(*describeResp.FileSystems[0].FileSystemId)

	tags := make([]*efs.Tag, 0)
	var marker string
	for {
		params := &efs.DescribeTagsInput{
			FileSystemId: aws.String(d.Id()),
		}
		if marker != "" {
			params.Marker = aws.String(marker)
		}

		tagsResp, err := efsconn.DescribeTags(params)
		if err != nil {
			return fmt.Errorf("Error retrieving EC2 tags for EFS file system (%q): %s",
				d.Id(), err.Error())
		}

		for _, tag := range tagsResp.Tags {
			tags = append(tags, tag)
		}

		if tagsResp.NextMarker != nil {
			marker = *tagsResp.NextMarker
		} else {
			break
		}
	}

	err = d.Set("tags", tagsToMapEFS(tags))
	if err != nil {
		return err
	}

	var fs *efs.FileSystemDescription
	for _, f := range describeResp.FileSystems {
		if d.Id() == *f.FileSystemId {
			fs = f
			break
		}
	}
	if fs == nil {
		log.Printf("[WARN] EFS (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("creation_token", fs.CreationToken)
	d.Set("performance_mode", fs.PerformanceMode)
	d.Set("file_system_id", fs.FileSystemId)

	return nil
}
