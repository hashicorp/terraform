package aws

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSLambdaFunction_basic(t *testing.T) {
	var conf lambda.GetFunctionOutput

	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigBasic(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_updateRuntime(t *testing.T) {
	var conf lambda.GetFunctionOutput

	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigBasic(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "runtime", "nodejs4.3"),
				),
			},
			{
				Config: testAccAWSLambdaConfigBasicUpdateRuntime(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "runtime", "nodejs4.3-edge"),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_expectFilenameAndS3Attributes(t *testing.T) {
	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAWSLambdaConfigWithoutFilenameAndS3Attributes(rName, rSt),
				ExpectError: regexp.MustCompile(`filename or s3_\* attributes must be set`),
			},
		},
	})
}

func TestAccAWSLambdaFunction_envVariables(t *testing.T) {
	var conf lambda.GetFunctionOutput

	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigBasic(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					resource.TestCheckNoResourceAttr("aws_lambda_function.lambda_function_test", "environment"),
				),
			},
			{
				Config: testAccAWSLambdaConfigEnvVariables(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "environment.0.variables.foo", "bar"),
				),
			},
			{
				Config: testAccAWSLambdaConfigEnvVariablesModified(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "environment.0.variables.foo", "baz"),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "environment.0.variables.foo1", "bar1"),
				),
			},
			{
				Config: testAccAWSLambdaConfigEnvVariablesModifiedWithoutEnvironment(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					resource.TestCheckNoResourceAttr("aws_lambda_function.lambda_function_test", "environment"),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_encryptedEnvVariables(t *testing.T) {
	var conf lambda.GetFunctionOutput

	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)
	keyRegex := regexp.MustCompile("^arn:aws:kms:")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigEncryptedEnvVariables(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "environment.0.variables.foo", "bar"),
					resource.TestMatchResourceAttr("aws_lambda_function.lambda_function_test", "kms_key_arn", keyRegex),
				),
			},
			{
				Config: testAccAWSLambdaConfigEncryptedEnvVariablesModified(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "environment.0.variables.foo", "bar"),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "kms_key_arn", ""),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_versioned(t *testing.T) {
	var conf lambda.GetFunctionOutput

	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigVersioned(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					resource.TestMatchResourceAttr("aws_lambda_function.lambda_function_test", "version",
						regexp.MustCompile("^[0-9]+$")),
					resource.TestMatchResourceAttr("aws_lambda_function.lambda_function_test", "qualified_arn",
						regexp.MustCompile(":"+rName+":[0-9]+$")),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_DeadLetterConfig(t *testing.T) {
	var conf lambda.GetFunctionOutput

	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigWithDeadLetterConfig(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					func(s *terraform.State) error {
						if !strings.HasSuffix(*conf.Configuration.DeadLetterConfig.TargetArn, ":"+rName) {
							return fmt.Errorf(
								"Expected DeadLetterConfig.TargetArn %s to have suffix %s", *conf.Configuration.DeadLetterConfig.TargetArn, ":"+rName,
							)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_nilDeadLetterConfig(t *testing.T) {
	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigWithNilDeadLetterConfig(rName, rSt),
				ExpectError: regexp.MustCompile(
					fmt.Sprintf("Nil dead_letter_config supplied for function: %s", rName)),
			},
		},
	})
}

func TestAccAWSLambdaFunction_tracingConfig(t *testing.T) {
	var conf lambda.GetFunctionOutput

	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigWithTracingConfig(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "tracing_config.0.mode", "Active"),
				),
			},
			{
				Config: testAccAWSLambdaConfigWithTracingConfigUpdated(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "tracing_config.0.mode", "PassThrough"),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_VPC(t *testing.T) {
	var conf lambda.GetFunctionOutput

	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigWithVPC(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					testAccCheckAWSLambdaFunctionVersion(&conf, "$LATEST"),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "vpc_config.#", "1"),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "vpc_config.0.subnet_ids.#", "1"),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "vpc_config.0.security_group_ids.#", "1"),
					resource.TestMatchResourceAttr("aws_lambda_function.lambda_function_test", "vpc_config.0.vpc_id", regexp.MustCompile("^vpc-")),
				),
			},
		},
	})
}

// See https://github.com/hashicorp/terraform/issues/5767
// and https://github.com/hashicorp/terraform/issues/10272
func TestAccAWSLambdaFunction_VPC_withInvocation(t *testing.T) {
	var conf lambda.GetFunctionOutput

	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigWithVPC(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccAwsInvokeLambdaFunction(&conf),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_s3(t *testing.T) {
	var conf lambda.GetFunctionOutput
	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigS3(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_s3test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					testAccCheckAWSLambdaFunctionVersion(&conf, "$LATEST"),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_localUpdate(t *testing.T) {
	var conf lambda.GetFunctionOutput

	path, zipFile, err := createTempFile("lambda_localUpdate")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	rInt := acctest.RandInt()
	rName := fmt.Sprintf("tf_acc_lambda_local_%d", rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testAccCreateZipFromFiles(map[string]string{"test-fixtures/lambda_func.js": "lambda.js"}, zipFile)
				},
				Config: genAWSLambdaFunctionConfig_local(path, rInt, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_local", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, rName),
					testAccCheckAwsLambdaSourceCodeHash(&conf, "8DPiX+G1l2LQ8hjBkwRchQFf1TSCEvPrYGRKlM9UoyY="),
				),
			},
			{
				PreConfig: func() {
					testAccCreateZipFromFiles(map[string]string{"test-fixtures/lambda_func_modified.js": "lambda.js"}, zipFile)
				},
				Config: genAWSLambdaFunctionConfig_local(path, rInt, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_local", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, rName),
					testAccCheckAwsLambdaSourceCodeHash(&conf, "0tdaP9H9hsk9c2CycSwOG/sa/x5JyAmSYunA/ce99Pg="),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_localUpdate_nameOnly(t *testing.T) {
	var conf lambda.GetFunctionOutput

	rName := fmt.Sprintf("tf_test_iam_%d", acctest.RandInt())

	path, zipFile, err := createTempFile("lambda_localUpdate")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	updatedPath, updatedZipFile, err := createTempFile("lambda_localUpdate_name_change")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(updatedPath)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testAccCreateZipFromFiles(map[string]string{"test-fixtures/lambda_func.js": "lambda.js"}, zipFile)
				},
				Config: genAWSLambdaFunctionConfig_local_name_only(path, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_local", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, rName),
					testAccCheckAwsLambdaSourceCodeHash(&conf, "8DPiX+G1l2LQ8hjBkwRchQFf1TSCEvPrYGRKlM9UoyY="),
				),
			},
			{
				PreConfig: func() {
					testAccCreateZipFromFiles(map[string]string{"test-fixtures/lambda_func_modified.js": "lambda.js"}, updatedZipFile)
				},
				Config: genAWSLambdaFunctionConfig_local_name_only(updatedPath, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_local", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, rName),
					testAccCheckAwsLambdaSourceCodeHash(&conf, "0tdaP9H9hsk9c2CycSwOG/sa/x5JyAmSYunA/ce99Pg="),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_s3Update_basic(t *testing.T) {
	var conf lambda.GetFunctionOutput

	path, zipFile, err := createTempFile("lambda_s3Update")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	bucketName := fmt.Sprintf("tf-acc-lambda-s3-deployments-%d", randomInteger)
	key := "lambda-func.zip"

	rInt := acctest.RandInt()

	rName := fmt.Sprintf("tf_acc_lambda_%d", rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					// Upload 1st version
					testAccCreateZipFromFiles(map[string]string{"test-fixtures/lambda_func.js": "lambda.js"}, zipFile)
				},
				Config: genAWSLambdaFunctionConfig_s3(bucketName, key, path, rInt, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_s3", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, rName),
					testAccCheckAwsLambdaSourceCodeHash(&conf, "8DPiX+G1l2LQ8hjBkwRchQFf1TSCEvPrYGRKlM9UoyY="),
				),
			},
			{
				ExpectNonEmptyPlan: true,
				PreConfig: func() {
					// Upload 2nd version
					testAccCreateZipFromFiles(map[string]string{"test-fixtures/lambda_func_modified.js": "lambda.js"}, zipFile)
				},
				Config: genAWSLambdaFunctionConfig_s3(bucketName, key, path, rInt, rName),
			},
			// Extra step because of missing ComputedWhen
			// See https://github.com/hashicorp/terraform/pull/4846 & https://github.com/hashicorp/terraform/pull/5330
			{
				Config: genAWSLambdaFunctionConfig_s3(bucketName, key, path, rInt, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_s3", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, rName),
					testAccCheckAwsLambdaSourceCodeHash(&conf, "0tdaP9H9hsk9c2CycSwOG/sa/x5JyAmSYunA/ce99Pg="),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_s3Update_unversioned(t *testing.T) {
	var conf lambda.GetFunctionOutput

	rName := fmt.Sprintf("tf_iam_lambda_%d", acctest.RandInt())

	path, zipFile, err := createTempFile("lambda_s3Update")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	bucketName := fmt.Sprintf("tf-acc-lambda-s3-deployments-%d", randomInteger)
	key := "lambda-func.zip"
	key2 := "lambda-func-modified.zip"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					// Upload 1st version
					testAccCreateZipFromFiles(map[string]string{"test-fixtures/lambda_func.js": "lambda.js"}, zipFile)
				},
				Config: testAccAWSLambdaFunctionConfig_s3_unversioned_tpl(rName, bucketName, key, path),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_s3", "tf_acc_lambda_name_s3_unversioned", &conf),
					testAccCheckAwsLambdaFunctionName(&conf, "tf_acc_lambda_name_s3_unversioned"),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, "tf_acc_lambda_name_s3_unversioned"),
					testAccCheckAwsLambdaSourceCodeHash(&conf, "8DPiX+G1l2LQ8hjBkwRchQFf1TSCEvPrYGRKlM9UoyY="),
				),
			},
			{
				PreConfig: func() {
					// Upload 2nd version
					testAccCreateZipFromFiles(map[string]string{"test-fixtures/lambda_func_modified.js": "lambda.js"}, zipFile)
				},
				Config: testAccAWSLambdaFunctionConfig_s3_unversioned_tpl(rName, bucketName, key2, path),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_s3", "tf_acc_lambda_name_s3_unversioned", &conf),
					testAccCheckAwsLambdaFunctionName(&conf, "tf_acc_lambda_name_s3_unversioned"),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, "tf_acc_lambda_name_s3_unversioned"),
					testAccCheckAwsLambdaSourceCodeHash(&conf, "0tdaP9H9hsk9c2CycSwOG/sa/x5JyAmSYunA/ce99Pg="),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_runtimeValidation_noRuntime(t *testing.T) {
	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAWSLambdaConfigNoRuntime(rName, rSt),
				ExpectError: regexp.MustCompile(`\\"runtime\\": required field is not set`),
			},
		},
	})
}

func TestAccAWSLambdaFunction_runtimeValidation_nodeJs(t *testing.T) {
	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAWSLambdaConfigNodeJsRuntime(rName, rSt),
				ExpectError: regexp.MustCompile(fmt.Sprintf("%s has reached end of life since October 2016 and has been deprecated in favor of %s", lambda.RuntimeNodejs, lambda.RuntimeNodejs43)),
			},
		},
	})
}

func TestAccAWSLambdaFunction_runtimeValidation_nodeJs43(t *testing.T) {
	var conf lambda.GetFunctionOutput
	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigNodeJs43Runtime(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "runtime", lambda.RuntimeNodejs43),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_runtimeValidation_python27(t *testing.T) {
	var conf lambda.GetFunctionOutput
	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigPython27Runtime(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "runtime", lambda.RuntimePython27),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_runtimeValidation_java8(t *testing.T) {
	var conf lambda.GetFunctionOutput
	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigJava8Runtime(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "runtime", lambda.RuntimeJava8),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_tags(t *testing.T) {
	var conf lambda.GetFunctionOutput

	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigBasic(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					resource.TestCheckNoResourceAttr("aws_lambda_function.lambda_function_test", "tags"),
				),
			},
			{
				Config: testAccAWSLambdaConfigTags(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "tags.%", "2"),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "tags.Key1", "Value One"),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "tags.Description", "Very interesting"),
				),
			},
			{
				Config: testAccAWSLambdaConfigTagsModified(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					testAccCheckAwsLambdaFunctionName(&conf, rName),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":"+rName),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "tags.%", "3"),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "tags.Key1", "Value One Changed"),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "tags.Key2", "Value Two"),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "tags.Key3", "Value Three"),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_runtimeValidation_python36(t *testing.T) {
	var conf lambda.GetFunctionOutput
	rSt := acctest.RandString(5)
	rName := fmt.Sprintf("tf_test_%s", rSt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLambdaConfigPython36Runtime(rName, rSt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", rName, &conf),
					resource.TestCheckResourceAttr("aws_lambda_function.lambda_function_test", "runtime", lambda.RuntimePython36),
				),
			},
		},
	})
}

func testAccCheckLambdaFunctionDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).lambdaconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_lambda_function" {
			continue
		}

		_, err := conn.GetFunction(&lambda.GetFunctionInput{
			FunctionName: aws.String(rs.Primary.ID),
		})

		if err == nil {
			return fmt.Errorf("Lambda Function still exists")
		}

	}

	return nil

}

func testAccCheckAwsLambdaFunctionExists(res, funcName string, function *lambda.GetFunctionOutput) resource.TestCheckFunc {
	// Wait for IAM role
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[res]
		if !ok {
			return fmt.Errorf("Lambda function not found: %s", res)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Lambda function ID not set")
		}

		conn := testAccProvider.Meta().(*AWSClient).lambdaconn

		params := &lambda.GetFunctionInput{
			FunctionName: aws.String(funcName),
		}

		getFunction, err := conn.GetFunction(params)
		if err != nil {
			return err
		}

		*function = *getFunction

		return nil
	}
}

func testAccAwsInvokeLambdaFunction(function *lambda.GetFunctionOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		f := function.Configuration
		conn := testAccProvider.Meta().(*AWSClient).lambdaconn

		// If the function is VPC-enabled this will create ENI automatically
		_, err := conn.Invoke(&lambda.InvokeInput{
			FunctionName: f.FunctionName,
		})

		return err
	}
}

func testAccCheckAwsLambdaFunctionName(function *lambda.GetFunctionOutput, expectedName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := function.Configuration
		if *c.FunctionName != expectedName {
			return fmt.Errorf("Expected function name %s, got %s", expectedName, *c.FunctionName)
		}

		return nil
	}
}

func testAccCheckAWSLambdaFunctionVersion(function *lambda.GetFunctionOutput, expectedVersion string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := function.Configuration
		if *c.Version != expectedVersion {
			return fmt.Errorf("Expected version %s, got %s", expectedVersion, *c.Version)
		}
		return nil
	}
}

func testAccCheckAwsLambdaFunctionArnHasSuffix(function *lambda.GetFunctionOutput, arnSuffix string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := function.Configuration
		if !strings.HasSuffix(*c.FunctionArn, arnSuffix) {
			return fmt.Errorf("Expected function ARN %s to have suffix %s", *c.FunctionArn, arnSuffix)
		}

		return nil
	}
}

func testAccCheckAwsLambdaSourceCodeHash(function *lambda.GetFunctionOutput, expectedHash string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := function.Configuration
		if *c.CodeSha256 != expectedHash {
			return fmt.Errorf("Expected code hash %s, got %s", expectedHash, *c.CodeSha256)
		}

		return nil
	}
}

func testAccCreateZipFromFiles(files map[string]string, zipFile *os.File) error {
	zipFile.Truncate(0)
	zipFile.Seek(0, 0)

	w := zip.NewWriter(zipFile)

	for source, destination := range files {
		f, err := w.Create(destination)
		if err != nil {
			return err
		}

		fileContent, err := ioutil.ReadFile(source)
		if err != nil {
			return err
		}

		_, err = f.Write(fileContent)
		if err != nil {
			return err
		}
	}

	err := w.Close()
	if err != nil {
		return err
	}

	return w.Flush()
}

func createTempFile(prefix string) (string, *os.File, error) {
	f, err := ioutil.TempFile(os.TempDir(), prefix)
	if err != nil {
		return "", nil, err
	}

	pathToFile, err := filepath.Abs(f.Name())
	if err != nil {
		return "", nil, err
	}
	return pathToFile, f, nil
}

func baseAccAWSLambdaConfig(rst string) string {
	return fmt.Sprintf(`
resource "aws_iam_role_policy" "iam_policy_for_lambda" {
    name = "iam_policy_for_lambda_%s"
    role = "${aws_iam_role.iam_for_lambda.id}"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:PutLogEvents"
            ],
            "Resource": "arn:aws:logs:*:*:*"
        },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateNetworkInterface",
				"ec2:DescribeNetworkInterfaces",
				"ec2:DeleteNetworkInterface"
      ],
      "Resource": [
        "*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "SNS:Publish"
      ],
      "Resource": [
        "*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "xray:PutTraceSegments"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
EOF
}

resource "aws_iam_role" "iam_for_lambda" {
    name = "iam_for_lambda_%s"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_vpc" "vpc_for_lambda" {
    cidr_block = "10.0.0.0/16"
		tags {
			Name = "baseAccAWSLambdaConfig"
		}
}

resource "aws_subnet" "subnet_for_lambda" {
    vpc_id = "${aws_vpc.vpc_for_lambda.id}"
    cidr_block = "10.0.1.0/24"

    tags {
        Name = "lambda"
    }
}

resource "aws_security_group" "sg_for_lambda" {
  name = "sg_for_lambda_%s"
  description = "Allow all inbound traffic for lambda test"
  vpc_id = "${aws_vpc.vpc_for_lambda.id}"

  ingress {
      from_port = 0
      to_port = 0
      protocol = "-1"
      cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
      from_port = 0
      to_port = 0
      protocol = "-1"
      cidr_blocks = ["0.0.0.0/0"]
  }
}`, rst, rst, rst)
}

func testAccAWSLambdaConfigBasic(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
}
`, rName)
}

func testAccAWSLambdaConfigBasicUpdateRuntime(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3-edge"
}
`, rName)
}

func testAccAWSLambdaConfigWithoutFilenameAndS3Attributes(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
		runtime = "nodejs4.3"
}
`, rName)
}

func testAccAWSLambdaConfigEnvVariables(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
    environment {
        variables = {
            foo = "bar"
        }
    }
}
`, rName)
}

func testAccAWSLambdaConfigEnvVariablesModified(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
    environment {
        variables = {
            foo = "baz"
            foo1 = "bar1"
        }
    }
}
`, rName)
}

func testAccAWSLambdaConfigEnvVariablesModifiedWithoutEnvironment(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
}
`, rName)
}

func testAccAWSLambdaConfigEncryptedEnvVariables(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_kms_key" "foo" {
    description = "Terraform acc test %s"
    policy = <<POLICY
{
  "Version": "2012-10-17",
  "Id": "kms-tf-1",
  "Statement": [
    {
      "Sid": "Enable IAM User Permissions",
      "Effect": "Allow",
      "Principal": {
        "AWS": "*"
      },
      "Action": "kms:*",
      "Resource": "*"
    }
  ]
}
POLICY
}

resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    kms_key_arn = "${aws_kms_key.foo.arn}"
    runtime = "nodejs4.3"
    environment {
        variables = {
            foo = "bar"
        }
    }
}
`, rName, rName)
}

func testAccAWSLambdaConfigEncryptedEnvVariablesModified(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
    environment {
        variables = {
            foo = "bar"
        }
    }
}
`, rName)
}

func testAccAWSLambdaConfigVersioned(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    publish = true
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
}
`, rName)
}

func testAccAWSLambdaConfigWithTracingConfig(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"

    tracing_config {
        mode = "Active"
    }
}

`, rName)
}

func testAccAWSLambdaConfigWithTracingConfigUpdated(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"

    tracing_config {
        mode = "PassThrough"
    }
}

`, rName)
}

func testAccAWSLambdaConfigWithDeadLetterConfig(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"

    dead_letter_config {
        target_arn = "${aws_sns_topic.lambda_function_test.arn}"
    }
}

resource "aws_sns_topic" "lambda_function_test" {
	name = "%s"
}

`, rName, rName)
}

func testAccAWSLambdaConfigWithNilDeadLetterConfig(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"

    dead_letter_config {
        target_arn = ""
    }
}
`, rName)
}

func testAccAWSLambdaConfigWithVPC(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"

    vpc_config = {
        subnet_ids = ["${aws_subnet.subnet_for_lambda.id}"]
        security_group_ids = ["${aws_security_group.sg_for_lambda.id}"]
    }
}`, rName)
}

func testAccAWSLambdaConfigS3(rName, rSt string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "lambda_bucket" {
  bucket = "tf-test-bucket-%d"
}

resource "aws_s3_bucket_object" "lambda_code" {
  bucket = "${aws_s3_bucket.lambda_bucket.id}"
  key = "lambdatest.zip"
  source = "test-fixtures/lambdatest.zip"
}

resource "aws_iam_role" "iam_for_lambda" {
    name = "iam_for_lambda_%s"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_lambda_function" "lambda_function_s3test" {
    s3_bucket = "${aws_s3_bucket.lambda_bucket.id}"
    s3_key = "${aws_s3_bucket_object.lambda_code.id}"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
}
`, acctest.RandInt(), rSt, rName)
}

func testAccAWSLambdaConfigNoRuntime(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
}
`, rName)
}

func testAccAWSLambdaConfigNodeJsRuntime(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
}
`, rName)
}

func testAccAWSLambdaConfigNodeJs43Runtime(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
}
`, rName)
}

func testAccAWSLambdaConfigPython27Runtime(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "python2.7"
}
`, rName)
}

func testAccAWSLambdaConfigJava8Runtime(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "java8"
}
`, rName)
}

func testAccAWSLambdaConfigTags(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
    tags {
		Key1 = "Value One"
		Description = "Very interesting"
    }
}
`, rName)
}

func testAccAWSLambdaConfigTagsModified(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
    tags {
		Key1 = "Value One Changed"
		Key2 = "Value Two"
		Key3 = "Value Three"
    }
}
`, rName)
}

func testAccAWSLambdaConfigPython36Runtime(rName, rSt string) string {
	return fmt.Sprintf(baseAccAWSLambdaConfig(rSt)+`
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "python3.6"
}
`, rName)
}

const testAccAWSLambdaFunctionConfig_local_tpl = `
resource "aws_iam_role" "iam_for_lambda" {
    name = "iam_for_lambda_%d"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}
resource "aws_lambda_function" "lambda_function_local" {
    filename = "%s"
    source_code_hash = "${base64sha256(file("%s"))}"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
}
`

func genAWSLambdaFunctionConfig_local(filePath string, rInt int, rName string) string {
	return fmt.Sprintf(testAccAWSLambdaFunctionConfig_local_tpl, rInt,
		filePath, filePath, rName)
}

func genAWSLambdaFunctionConfig_local_name_only(filePath, rName string) string {
	return testAccAWSLambdaFunctionConfig_local_name_only_tpl(filePath, rName)
}

func testAccAWSLambdaFunctionConfig_local_name_only_tpl(filePath, rName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "iam_for_lambda" {
    name = "%s"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}
resource "aws_lambda_function" "lambda_function_local" {
    filename = "%s"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
}`, rName, filePath, rName)
}

const testAccAWSLambdaFunctionConfig_s3_tpl = `
resource "aws_s3_bucket" "artifacts" {
    bucket = "%s"
    acl = "private"
    force_destroy = true
    versioning {
        enabled = true
    }
}
resource "aws_s3_bucket_object" "o" {
    bucket = "${aws_s3_bucket.artifacts.bucket}"
    key = "%s"
    source = "%s"
    etag = "${md5(file("%s"))}"
}
resource "aws_iam_role" "iam_for_lambda" {
    name = "iam_for_lambda_%d"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}
resource "aws_lambda_function" "lambda_function_s3" {
    s3_bucket = "${aws_s3_bucket_object.o.bucket}"
    s3_key = "${aws_s3_bucket_object.o.key}"
    s3_object_version = "${aws_s3_bucket_object.o.version_id}"
    function_name = "%s"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
}
`

func genAWSLambdaFunctionConfig_s3(bucket, key, path string, rInt int, rName string) string {
	return fmt.Sprintf(testAccAWSLambdaFunctionConfig_s3_tpl,
		bucket, key, path, path, rInt, rName)
}

func testAccAWSLambdaFunctionConfig_s3_unversioned_tpl(rName, bucketName, key, path string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "artifacts" {
    bucket = "%s"
    acl = "private"
    force_destroy = true
}
resource "aws_s3_bucket_object" "o" {
    bucket = "${aws_s3_bucket.artifacts.bucket}"
    key = "%s"
    source = "%s"
    etag = "${md5(file("%s"))}"
}
resource "aws_iam_role" "iam_for_lambda" {
		name = "%s"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}
resource "aws_lambda_function" "lambda_function_s3" {
    s3_bucket = "${aws_s3_bucket_object.o.bucket}"
    s3_key = "${aws_s3_bucket_object.o.key}"
    function_name = "tf_acc_lambda_name_s3_unversioned"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
    runtime = "nodejs4.3"
}`, bucketName, key, path, path, rName)
}
