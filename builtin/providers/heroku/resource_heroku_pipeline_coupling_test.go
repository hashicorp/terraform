package heroku

import (
	"context"
	"fmt"
	"testing"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuPipelineCoupling_Basic(t *testing.T) {
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
		},
	})
}

func testAccCheckHerokuPipelineCouplingConfig_basic(appName, pipelineName, stageName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "default" {
  name   = "%s"
  region = "us"
}

resource "heroku_pipeline" "default" {
  name = "%s"
}

resource "heroku_pipeline_coupling" "default" {
  app      = "${heroku_app.default.id}"
  pipeline = "${heroku_pipeline.default.id}"
  stage    = "%s"
}
`, appName, pipelineName, stageName)
}

func testAccCheckHerokuPipelineCouplingExists(n string, pipeline *heroku.PipelineCoupling) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No coupling ID set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundPipelineCoupling, err := client.PipelineCouplingInfo(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if foundPipelineCoupling.ID != rs.Primary.ID {
			return fmt.Errorf("PipelineCoupling not found: %s != %s", foundPipelineCoupling.ID, rs.Primary.ID)
		}

		*pipeline = *foundPipelineCoupling

		return nil
	}
}

func testAccCheckHerokuPipelineCouplingAttributes(coupling *heroku.PipelineCoupling, pipelineResource, stageName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		pipeline, ok := s.RootModule().Resources[pipelineResource]
		if !ok {
			return fmt.Errorf("Pipeline not found: %s", pipelineResource)
		}

		if coupling.Pipeline.ID != pipeline.Primary.ID {
			return fmt.Errorf("Bad pipeline ID: %v != %v", coupling.Pipeline.ID, pipeline.Primary.ID)
		}
		if coupling.Stage != stageName {
			return fmt.Errorf("Bad stage: %s", coupling.Stage)
		}

		return nil
	}
}

func testAccCheckHerokuPipelineCouplingDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*heroku.Service)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_pipeline_coupling" {
			continue
		}

		_, err := client.PipelineCouplingInfo(context.TODO(), rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("PipelineCoupling still exists")
		}
	}

	return nil
}
