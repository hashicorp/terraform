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

	tfe "github.com/hashicorp/go-tfe"
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
