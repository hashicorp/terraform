//go:build e2e
// +build e2e

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
)

var terraformVersion string
var terraformBin string
var cliConfigFileEnv string

var tfeClient *tfe.Client
var tfeHostname string
var tfeToken string

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
	setTfeClient()
	teardown := setupBinary()
	setVersion()
	ensureVersionExists()

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

	cmd := exec.Command("go", "build", "-o", tmpTerraformBinaryDir)
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

func setVersion() {
	log.Println("Retrieving version")
	cmd := exec.Command(terraformBin, "version", "-json")
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(fmt.Sprintf("Could not output terraform version: %v", err))
	}
	var data map[string]interface{}
	if err := json.Unmarshal(out, &data); err != nil {
		log.Fatal(fmt.Sprintf("Could not unmarshal version output: %v", err))
	}

	out, err = exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		log.Fatal(fmt.Sprintf("Could not execute go build command: %v", err))
	}

	hash := string(out)[0:8]

	fullVersion := data["terraform_version"].(string)
	version := strings.Split(fullVersion, "-")[0]
	terraformVersion = fmt.Sprintf("%s-%s", version, hash)
}

func ensureVersionExists() {
	opts := tfe.AdminTerraformVersionsListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: 1,
			PageSize:   100,
		},
	}
	hasVersion := false

findTfVersion:
	for {
		tfVersionList, err := tfeClient.Admin.TerraformVersions.List(context.Background(), opts)
		if err != nil {
			log.Fatalf("Could not retrieve list of terraform versions: %v", err)
		}
		for _, item := range tfVersionList.Items {
			if item.Version == terraformVersion {
				hasVersion = true
				break findTfVersion
			}
		}

		// Exit the loop when we've seen all pages.
		if tfVersionList.CurrentPage >= tfVersionList.TotalPages {
			break
		}

		// Update the page number to get the next page.
		opts.PageNumber = tfVersionList.NextPage
	}

	if !hasVersion {
		log.Fatalf("Terraform Version %s does not exist in the list. Please add it.", terraformVersion)
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
