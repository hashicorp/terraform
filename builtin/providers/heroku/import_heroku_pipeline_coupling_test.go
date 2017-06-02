package heroku

import (
	"fmt"
	"testing"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccHerokuPipelineCoupling_importBasic(t *testing.T) {
	var coupling heroku.PipelineCoupling

	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	pipelineName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	stageName := "development"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuPipelineCouplingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuPipelineCouplingConfig_basic(appName, pipelineName, stageName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuPipelineCouplingExists("heroku_pipeline_coupling.default", &coupling),
					testAccCheckHerokuPipelineCouplingAttributes(
						&coupling,
						"heroku_pipeline.default",
						stageName,
					),
				),
			},
			{
				ResourceName:            "heroku_pipeline_coupling.default",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config_vars"},
			},
		},
	})
}
