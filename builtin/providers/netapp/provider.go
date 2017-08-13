package netapp

import (
	"fmt"
	"log"
	"time"

	"github.com/candidpartners/occm-sdk-go/api/workenv"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

const REQUEST_RESOLUTION_RETRY_COUNT = 60
const REQUEST_RESOLUTION_WAIT_TIME = 2 * time.Second

// Provider represents a resource provider in Terraform
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"email": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NETAPP_EMAIL", nil),
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NETAPP_PASSWORD", nil),
				Sensitive:   true,
			},
			"host": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("NETAPP_HOST", nil),
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"netapp_cloud_workenv": dataSourceCloudWorkingEnvironment(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"netapp_cloud_volume": resourceCloudVolume(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(data *schema.ResourceData) (interface{}, error) {
	config := Config{
		Host:     data.Get("host").(string),
		Email:    data.Get("email").(string),
		Password: data.Get("password").(string),
	}

	apis, err := config.APIs()
	if err != nil {
		return nil, fmt.Errorf("Error creating APIs: %s", err)
	}

	log.Println("[INFO] Initializing NetApp client")

	err = apis.AuthAPI.Login(config.Email, config.Password)
	if err != nil {
		return nil, fmt.Errorf("Error logging in user %s: %s", config.Email, err)
	}

	return apis, nil
}

func GetWorkingEnvironments(apis *APIs) ([]workenv.VsaWorkingEnvironment, error) {
	resp, err := apis.WorkingEnvironmentAPI.GetWorkingEnvironments()
	if err != nil {
		return nil, err
	}

	return resp.VSA, nil
}

func GetWorkingEnvironmentByName(apis *APIs, workEnvName string) (*workenv.VsaWorkingEnvironment, error) {
	workEnvs, err := GetWorkingEnvironments(apis)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Reading working environment %s", workEnvName)

	var found *workenv.VsaWorkingEnvironment

	for _, workenv := range workEnvs {
		if workenv.Name == workEnvName {
			found = &workenv
			break
		}
	}

	if found == nil {
		return nil, fmt.Errorf("Working environment %s not found", workEnvName)
	}

	log.Printf("[DEBUG] Found working environment %s", workEnvName)

	return found, nil
}

func GetWorkingEnvironmentById(apis *APIs, workEnvId string) (*workenv.VsaWorkingEnvironment, error) {
	workEnvs, err := GetWorkingEnvironments(apis)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Reading working environment for ID %s", workEnvId)

	var found *workenv.VsaWorkingEnvironment

	for _, workEnv := range workEnvs {
		if workEnv.PublicId == workEnvId {
			found = &workEnv
			break
		}
	}

	if found == nil {
		return nil, fmt.Errorf("Working environment with ID %s not found", workEnvId)
	}

	return found, nil
}

func WaitForRequest(apis *APIs, requestId string) error {
	log.Printf("[DEBUG] Waiting for completion of request %s", requestId)

	for i := 0; i < REQUEST_RESOLUTION_RETRY_COUNT; i++ {
		summary, err := apis.AuditAPI.GetAuditSummary(requestId)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] Received status for request %s: %s", requestId, summary.Status)

		if summary.Status == "Failed" {
			log.Printf("[DEBUG] Failure detected, breaking wait loop")
			return fmt.Errorf(summary.ErrorMessage)
		}

		if summary.Status == "Success" {
			log.Printf("[DEBUG] Request completion detected, breaking wait loop")
			return nil
		}

		time.Sleep(REQUEST_RESOLUTION_WAIT_TIME)
	}

	return fmt.Errorf("Timed out waiting for request completion")
}
