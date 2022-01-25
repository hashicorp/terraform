package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	expect "github.com/Netflix/go-expect"
	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/e2e"
	tfversion "github.com/hashicorp/terraform/version"
)

var terraformBin string
var cliConfigFileEnv string

var tfeClient *tfe.Client
var tfeHostname string
var tfeToken string
var verboseMode bool

func TestMain(m *testing.M) {
	teardown := setup()
	code := m.Run()
	teardown()

	os.Exit(code)
}

func accTest() bool {
	// TF_ACC is set when we want to run acceptance tests, meaning it relies on
	// network access.
	return os.Getenv("TF_ACC") != ""
}

func hasHostname() bool {
	return os.Getenv("TFE_HOSTNAME") != ""
}

func hasToken() bool {
	return os.Getenv("TFE_TOKEN") != ""
}

func hasRequiredEnvVars() bool {
	return accTest() && hasHostname() && hasToken()
}

func skipIfMissingEnvVar(t *testing.T) {
	if !hasRequiredEnvVars() {
		t.Skip("Skipping test, required environment variables missing. Use `TF_ACC`, `TFE_HOSTNAME`, `TFE_TOKEN`")
	}
}

func setup() func() {
	tfOutput := flag.Bool("tfoutput", false, "This flag produces the terraform output from tests.")
	flag.Parse()
	verboseMode = *tfOutput

	setTfeClient()
	teardown := setupBinary()

	return func() {
		teardown()
	}
}
func testRunner(t *testing.T, cases testCases, orgCount int, tfEnvFlags ...string) {
	for name, tc := range cases {
		tc := tc // rebind tc into this lexical scope
		t.Run(name, func(subtest *testing.T) {
			subtest.Parallel()

			orgNames := []string{}
			for i := 0; i < orgCount; i++ {
				organization, cleanup := createOrganization(t)
				t.Cleanup(cleanup)
				orgNames = append(orgNames, organization.Name)
			}

			exp, err := expect.NewConsole(defaultOpts()...)
			if err != nil {
				subtest.Fatal(err)
			}
			defer exp.Close()

			tmpDir, err := ioutil.TempDir("", "terraform-test")
			if err != nil {
				subtest.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			tf := e2e.NewBinary(terraformBin, tmpDir)
			tfEnvFlags = append(tfEnvFlags, "TF_LOG=INFO")
			tfEnvFlags = append(tfEnvFlags, cliConfigFileEnv)
			for _, env := range tfEnvFlags {
				tf.AddEnv(env)
			}
			defer tf.Close()

			var orgName string
			for index, op := range tc.operations {
				if orgCount == 1 {
					orgName = orgNames[0]
				} else {
					orgName = orgNames[index]
				}
				op.prep(t, orgName, tf.WorkDir())
				for _, tfCmd := range op.commands {
					cmd := tf.Cmd(tfCmd.command...)
					cmd.Stdin = exp.Tty()
					cmd.Stdout = exp.Tty()
					cmd.Stderr = exp.Tty()

					err = cmd.Start()
					if err != nil {
						subtest.Fatal(err)
					}

					if tfCmd.expectedCmdOutput != "" {
						got, err := exp.ExpectString(tfCmd.expectedCmdOutput)
						if err != nil {
							subtest.Fatalf("error while waiting for output\nwant: %s\nerror: %s\noutput\n%s", tfCmd.expectedCmdOutput, err, got)
						}
					}

					lenInput := len(tfCmd.userInput)
					lenInputOutput := len(tfCmd.postInputOutput)
					if lenInput > 0 {
						for i := 0; i < lenInput; i++ {
							input := tfCmd.userInput[i]
							exp.SendLine(input)
							// use the index to find the corresponding
							// output that matches the input.
							if lenInputOutput-1 >= i {
								output := tfCmd.postInputOutput[i]
								_, err := exp.ExpectString(output)
								if err != nil {
									subtest.Fatal(err)
								}
							}
						}
					}

					err = cmd.Wait()
					if err != nil && !tfCmd.expectError {
						subtest.Fatal(err)
					}
				}
			}

			if tc.validations != nil {
				tc.validations(t, orgName)
			}
		})
	}
}

func setTfeClient() {
	tfeHostname = os.Getenv("TFE_HOSTNAME")
	tfeToken = os.Getenv("TFE_TOKEN")

	cfg := &tfe.Config{
		Address: fmt.Sprintf("https://%s", tfeHostname),
		Token:   tfeToken,
	}

	if tfeHostname != "" && tfeToken != "" {
		// Create a new TFE client.
		client, err := tfe.NewClient(cfg)
		if err != nil {
			fmt.Printf("Could not create new tfe client: %v\n", err)
			os.Exit(1)
		}
		tfeClient = client
	}
}

func setupBinary() func() {
	log.Println("Setting up terraform binary")
	tmpTerraformBinaryDir, err := ioutil.TempDir("", "terraform-test")
	if err != nil {
		fmt.Printf("Could not create temp directory: %v\n", err)
		os.Exit(1)
	}
	log.Println(tmpTerraformBinaryDir)
	currentDir, err := os.Getwd()
	defer os.Chdir(currentDir)
	if err != nil {
		fmt.Printf("Could not change directories: %v\n", err)
		os.Exit(1)
	}
	// Getting top level dir
	dirPaths := strings.Split(currentDir, "/")
	log.Println(currentDir)
	topLevel := len(dirPaths) - 3
	topDir := strings.Join(dirPaths[0:topLevel], "/")

	if err := os.Chdir(topDir); err != nil {
		fmt.Printf("Could not change directories: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command(
		"go",
		"build",
		"-o", tmpTerraformBinaryDir,
		"-ldflags", fmt.Sprintf("-X \"github.com/hashicorp/terraform/version.Prerelease=%s\"", tfversion.Prerelease),
	)
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Could not run exec command: %v\n", err)
		os.Exit(1)
	}

	credFile := fmt.Sprintf("%s/dev.tfrc", tmpTerraformBinaryDir)
	writeCredRC(credFile)

	terraformBin = fmt.Sprintf("%s/terraform", tmpTerraformBinaryDir)
	cliConfigFileEnv = fmt.Sprintf("TF_CLI_CONFIG_FILE=%s", credFile)

	return func() {
		os.RemoveAll(tmpTerraformBinaryDir)
	}
}

func writeCredRC(file string) {
	creds := credentialBlock()
	f, err := os.Create(file)
	if err != nil {
		fmt.Printf("Could not create file: %v\n", err)
		os.Exit(1)
	}
	_, err = f.WriteString(creds)
	if err != nil {
		fmt.Printf("Could not write credentials: %v\n", err)
		os.Exit(1)
	}
	f.Close()
}

func credentialBlock() string {
	return fmt.Sprintf(`
credentials "%s" {
  token = "%s"
}`, tfeHostname, tfeToken)
}
