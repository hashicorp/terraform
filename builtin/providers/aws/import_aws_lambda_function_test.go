package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSLambdaFunction_importLocalFile(t *testing.T) {
	resourceName := "aws_lambda_function.lambda_function_test"

	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSLambdaConfigBasic(rName, rSt),
			},

			resource.TestStep{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"filename", "publish"},
			},
		},
	})
}

func TestAccAWSLambdaFunction_importLocalFile_VPC(t *testing.T) {
	resourceName := "aws_lambda_function.lambda_function_test"

	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSLambdaConfigWithVPC(rName, rSt),
			},

			resource.TestStep{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"filename", "publish"},
			},
		},
	})
}

func TestAccAWSLambdaFunction_importS3(t *testing.T) {
	resourceName := "aws_lambda_function.lambda_function_s3test"

	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSLambdaConfigS3(rName, rSt),
			},

			resource.TestStep{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"s3_bucket", "s3_key", "publish"},
			},
		},
	})
}
