package google

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccComputeImage_resolveImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeImage_basedondisk,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeImageExists(
						"google_compute_image.foobar", &image),
				),
			},
		},
	})
	images := map[string]string{
		"family/debian-8":                                                                                     "projects/debian-cloud/global/images/family/debian-8-jessie",
		"projects/debian-cloud/global/images/debian-8-jessie-v20170110":                                       "projects/debian-cloud/global/images/debian-8-jessie-v20170110",
		"debian-8-jessie":                                                                                     "projects/debian-cloud/global/images/family/debian-8-jessie",
		"debian-8-jessie-v20170110":                                                                           "projects/debian-cloud/global/images/debian-8-jessie-v20170110",
		"https://www.googleapis.com/compute/v1/projects/debian-cloud/global/images/debian-8-jessie-v20170110": "https://www.googleapis.com/compute/v1/projects/debian-cloud/global/images/debian-8-jessie-v20170110",

		// TODO(paddy): we need private images/families here to actually test this
		"global/images/my-private-image":         "global/images/my-private-image",
		"global/images/family/my-private-family": "global/images/family/my-private-family",
		"my-private-image":                       "global/images/my-private-image",
		"my-private-family":                      "global/images/family/my-private-family",
		"my-project/my-private-image":            "projects/my-project/global/images/my-private-image",
		"my-project/my-private-family":           "projects/my-project/global/images/family/my-private-family",
		"insert-URL-here":                        "insert-URL-here",
	}
	config := &Config{
		Credentials: credentials,
		Project:     project,
		Region:      region,
	}

	err := config.loadAndValidate()
	if err != nil {
		t.Fatalf("Error loading config: %s\n", err)
	}
	for input, expectation := range images {
		result, err := resolveImage(config, input)
		if err != nil {
			t.Errorf("Error resolving input %s to image: %+v\n", input, err)
			continue
		}
		if result != expectation {
			t.Errorf("Expected input '%s' to resolve to '%s', it resolved to '%s' instead.\n", input, expectation, result)
			continue
		}
	}
}
