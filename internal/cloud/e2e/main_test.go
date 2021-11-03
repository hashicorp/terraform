//go:build e2e
// +build e2e

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
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if !accTest() {
		// if TF_ACC is not set, we want to skip all these tests.
		return
	}
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
	hostname := os.Getenv("TFE_HOSTNAME")
	token := os.Getenv("TFE_TOKEN")
	if hostname == "" {
		log.Fatal("hostname cannot be empty")
	}
	if token == "" {
		log.Fatal("token cannot be empty")
	}
	tfeHostname = hostname
	tfeToken = token

	cfg := &tfe.Config{
		Address: fmt.Sprintf("https://%s", hostname),
		Token:   token,
	}

	// Create a new TFE client.
	client, err := tfe.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}
	tfeClient = client
}

func setupBinary() func() {
	log.Println("Setting up terraform binary")
	tmpTerraformBinaryDir, err := ioutil.TempDir("", "terraform-test")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(tmpTerraformBinaryDir)
	currentDir, err := os.Getwd()
	defer os.Chdir(currentDir)
	if err != nil {
		log.Fatal(err)
	}
	// Getting top level dir
	dirPaths := strings.Split(currentDir, "/")
	log.Println(currentDir)
	topLevel := len(dirPaths) - 3
	topDir := strings.Join(dirPaths[0:topLevel], "/")

	if err := os.Chdir(topDir); err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(
		"go",
		"build",
		"-o", tmpTerraformBinaryDir,
		"-ldflags", fmt.Sprintf("-X \"github.com/hashicorp/terraform/version.Prerelease=%s\"", tfversion.Prerelease),
	)
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
	_, err = f.WriteString(creds)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
}

func credentialBlock() string {
	return fmt.Sprintf(`
credentials "%s" {
  token = "%s"
}`, tfeHostname, tfeToken)
}
