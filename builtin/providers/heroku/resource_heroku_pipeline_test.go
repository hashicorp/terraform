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

func TestAccHerokuPipeline_Basic(t *testing.T) {
	var pipeline heroku.Pipeline
	pipelineName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	pipelineName2 := fmt.Sprintf("%s-2", pipelineName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuPipelineDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuPipelineConfig_basic(pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuPipelineExists("heroku_pipeline.foobar", &pipeline),
					resource.TestCheckResourceAttr(
						"heroku_pipeline.foobar", "name", pipelineName),
				),
			},
			{
				Config: testAccCheckHerokuPipelineConfig_basic(pipelineName2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"heroku_pipeline.foobar", "name", pipelineName2),
				),
			},
		},
	})
}

func testAccCheckHerokuPipelineConfig_basic(pipelineName string) string {
	return fmt.Sprintf(`
resource "heroku_pipeline" "foobar" {
  name = "%s"
}
`, pipelineName)
}

func testAccCheckHerokuPipelineExists(n string, pipeline *heroku.Pipeline) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No pipeline name set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundPipeline, err := client.PipelineInfo(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if foundPipeline.ID != rs.Primary.ID {
			return fmt.Errorf("Pipeline not found")
		}

		*pipeline = *foundPipeline

		return nil
	}
}

func testAccCheckHerokuPipelineDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*heroku.Service)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_pipeline" {
			continue
		}

		_, err := client.PipelineInfo(context.TODO(), rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Pipeline still exists")
		}
	}

	return nil
}
