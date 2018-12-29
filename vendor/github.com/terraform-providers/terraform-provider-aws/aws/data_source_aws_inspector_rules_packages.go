package aws

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/service/inspector"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsInspectorRulesPackages() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsInspectorRulesPackagesRead,

		Schema: map[string]*schema.Schema{
			"arns": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceAwsInspectorRulesPackagesRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	log.Printf("[DEBUG] Reading Rules Packages.")
	d.SetId(time.Now().UTC().String())

	var arns []string

	input := &inspector.ListRulesPackagesInput{}

	err := conn.ListRulesPackagesPages(input, func(page *inspector.ListRulesPackagesOutput, lastPage bool) bool {
		for _, arn := range page.RulesPackageArns {
			arns = append(arns, *arn)
		}
		return !lastPage
	})
	if err != nil {
		return fmt.Errorf("Error fetching Rules Packages: %s", err)
	}

	if len(arns) == 0 {
		return errors.New("No rules packages found.")
	}

	sort.Strings(arns)
	d.Set("arns", arns)

	return nil
}
