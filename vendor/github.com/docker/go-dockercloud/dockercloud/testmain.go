package dockercloud

import (
	"fmt"
	"io/ioutil"
	"os"
)

/*
Test variables used as arguments
*/
var (
	fake_location_name    = "ams2"
	fake_name             = "1gb"
	fake_provider         = "digitalocean"
	fake_uuid_action      = "6246c558-976c-4df6-ba60-eb1a344a17af"
	fake_uuid_container   = "dcbe16b4-21a1-474b-a814-131a3626b1de"
	fake_uuid_node        = "89226618-4cbf-44a7-b354-0edd6e251068"
	fake_uuid_nodecluster = "72a7902a-5f70-4771-bcbf-4abb3a4f93fe"
	fake_uuid_service     = "02522970-a79a-46d6-8a64-475bf52e4258"
	fake_uuid_stack       = "09cbcf8d-a727-40d9-b420-c8e18b7fa55b"
	fake_uuid_volume      = "1863e34d-6a7d-4945-aefc-8f27a4ab1a9e"
	fake_uuid_volumegroup = "1863e34d-6a7d-4945-aefc-8f27a4ab1a9e"
	fake_image_name       = "tutum/mysql"
	fake_image_tag        = "latest"
)

func MockupResponse(response_file string) (string, error) {

	file, e := ioutil.ReadFile("json_test_output/" + response_file)
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}

	fake_response := string(file)

	return fake_response, nil
}
