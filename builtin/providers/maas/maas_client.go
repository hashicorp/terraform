package maas

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"launchpad.net/gomaasapi"
	"log"
	"net/url"
	"strconv"
	"strings"
)

/*
This is a *low level* function that access a MAAS Server and returns an array of MAASObject
The function takes a pointer to an already active MAASObject and returns a JSONObject array
and an error code.
*/
func maasListAllNodes(maas *gomaasapi.MAASObject) ([]gomaasapi.JSONObject, error) {
	nodeListing := maas.GetSubObject("nodes")
	log.Printf("[DEBUG] [maasListAllNodes] Fetching list of nodes...\n")
	listNodeObjects, err := nodeListing.CallGet("list", url.Values{})
	if err != nil {
		log.Printf("[ERROR] [maasListAllNodes] Unable to get list of nodes ...\n")
		return nil, err
	}

	listNodes, err := listNodeObjects.GetArray()
	if err != nil {
		log.Printf("[ERROR] [maasListAllNodes] Unable to get the node list array ...\n")
		return nil, err
	}
	return listNodes, err
}

/*
This is a *low level* function that access a MAAS Server and returns a MAASObject
referring to a single MAAS managed node.
The function takes a pointer to an already active MAASObject as well as a system_id and returns a MAASObject array
and an error code.
*/
func maasGetSingleNode(maas *gomaasapi.MAASObject, system_id string) (gomaasapi.MAASObject, error) {
	log.Printf("[DEBUG] [maasGetSingleNode] Getting a node (%s) from MAAS\n", system_id)
	nodeObject, err := maas.GetSubObject("nodes").GetSubObject(system_id).Get()
	if err != nil {
		log.Printf("[ERROR] [maasGetSingleNode] Unable to get node (%s) from MAAS\n", system_id)
		return gomaasapi.MAASObject{}, err
	}
	return nodeObject, nil
}

/*
This is a *low level* function that attempts to acquire a MAAS managed node for future deployment.
*/
func maasAllocateNodes(maas *gomaasapi.MAASObject, params url.Values) (gomaasapi.MAASObject, error) {
	log.Printf("[DEBUG] [maasAllocateNodes] Allocating one or more nodes with following params: %+v", params)

	nodeObject, err := maas.GetSubObject("nodes").CallPost("acquire", params)
	if err != nil {
		log.Printf("[ERROR] [maasAllocateNodes] Unable to acquire a node ... bailing")
		return gomaasapi.MAASObject{}, err
	}
	return nodeObject.GetMAASObject()
}

func maasReleaseNode(maas *gomaasapi.MAASObject, system_id string) error {
	log.Printf("[DEBUG] [maasReleaseNode] Releasing node: %s", system_id)

	_, err := maas.GetSubObject("nodes").GetSubObject(system_id).CallPost("release", url.Values{})
	if err != nil {
		log.Printf("[DEBUG] [maasReleaseNode] Unable to release node (%s)", system_id)
		return err
	}
	return nil
}

/*
Convenience function to convert a MAASObject to a NodeInfo instance
The function takes a fully initialized MAASObject and returns a NodeInfo instance
with an error
*/
func toNodeInfo(nodeObject *gomaasapi.MAASObject) (*NodeInfo, error) {
	log.Println("[DEBUG] [toNodeInfo] Attempting to convert node information from MAASObject to NodeInfo")

	nodeMap := nodeObject.GetMap()

	system_id, err := nodeMap["system_id"].GetString()
	if err != nil {
		log.Printf("[ERROR] [toNodeInfo] Unable to get node (%s)\n", system_id)
		return nil, err
	}

	hostname, err := nodeMap["hostname"].GetString()
	if err != nil {
		log.Printf("[ERROR] [toNodeInfo] Unable to get the node (%s) hostname\n", system_id)
		return nil, err
	}

	url := nodeObject.URL().String()
	if len(url) == 0 {
		return nil, errors.New("[ERROR] [toNodeInfo] Empty URL for node")
	}

	power_state, err := nodeMap["power_state"].GetString()
	if err != nil {
		log.Printf("[ERROR] [toNodeInfo] Unable to get the power_state for node: %s\n", system_id)
		return nil, err
	}

	cpu_count_float, err := nodeMap["cpu_count"].GetFloat64()
	if err != nil {
		log.Printf("[ERROR] [toNodeInfo] Unable to get the cpu_count for node: %s\n", system_id)
		log.Printf("[ERROR] [toNodeInfo] Error: %s\n", err)
		log.Printf("[ERROR] [toNodeInfo] cpu_count_float: %v\n", cpu_count_float)
		log.Println("[ERROR] [toNodeInfo] Defaulting cpu_count to 0")
		cpu_count_float = 0
	}
	cpu_count := uint16(cpu_count_float)

	architecture, err := nodeMap["architecture"].GetString()
	if err != nil {
		log.Printf("[ERROR] [toNodeInfo] Unable to get the node (%s) architecture\n", system_id)
		return nil, err
	}

	distro_series, err := nodeMap["distro_series"].GetString()
	if err != nil {
		log.Printf("[ERROR] [toNodeInfo] Unable to get the distro_series for node: %s\n", system_id)
		return nil, err
	}

	memory_float, err := nodeMap["memory"].GetFloat64()
	if err != nil {
		log.Printf("[ERROR] [toNodeInfo] Unable to get the memory for node: %s\n", system_id)
		log.Printf("[ERROR] [toNodeInfo] Error: %s\n", err)
		log.Printf("[ERROR] [toNodeInfo] memory_float: %v\n", memory_float)
		log.Printf("[ERROR] [toNodeInfo] Defaulting memory to 0")
		memory_float = 0
	}
	memory := uint64(memory_float)

	osystem, err := nodeMap["osystem"].GetString()
	if err != nil {
		log.Printf("[ERROR] [toNodeInfo] Unable to get the OS for node: %s\n", system_id)
		return nil, err
	}

	status_float, err := nodeMap["status"].GetFloat64()
	if err != nil {
		log.Printf("[ERROR] [toNodeInfo] Unable to get the status for node: %s\n", system_id)
		return nil, err
	}
	status := uint16(status_float)

	substatus_float, err := nodeMap["substatus"].GetFloat64()
	if err != nil {
		log.Printf("[ERROR] [toNodeInfo] Unable to get the node (%s) substatus\n", system_id)
		return nil, err
	}
	substatus := uint16(substatus_float)

	tag_names, err := nodeMap["tag_names"].GetArray()
	if err != nil {
		log.Printf("[ERROR] [toNodeInfo] Unable to get the tags for node: %s\n", system_id)
		return nil, err
	}

	tag_array := make([]string, 0, 1)
	for _, tag_object := range tag_names {
		tag_name, err := tag_object.GetString()
		if err != nil {
			log.Printf("[ERROR] [toNodeInfo] Unable to parse tag information (%v) for node (%s)", tag_object, system_id)
			return nil, err
		}
		tag_array = append(tag_array, tag_name)
	}

	prettyJSON, err := json.MarshalIndent(nodeObject, "", "    ")
	if err != nil {
		log.Printf("[ERROR] [toNodeInfo] Unable to convert node (%s) information to JSON\n", system_id)
		return nil, err
	}

	log.Printf("[DEBUG] [toNodeInfo] Node (%s) JSON:\n%s\n", system_id, prettyJSON)

	var raw_data map[string]interface{}
	if err := json.Unmarshal(prettyJSON, &raw_data); err != nil {
		log.Printf("[ERROR] [toNodeInfo] Unable to Unmarshal JSON data for node: %s\n", system_id)
		return nil, err
	}

	return &NodeInfo{system_id: system_id,
		hostname:      hostname,
		url:           url,
		power_state:   power_state,
		cpu_count:     uint16(cpu_count),
		architecture:  architecture,
		distro_series: distro_series,
		memory:        memory,
		osystem:       osystem,
		status:        uint16(status),
		substatus:     uint16(substatus),
		tag_names:     tag_array,
		data:          raw_data}, nil
}

/*
Convenience function used by resourceMAASInstanceCreate as a refresh function
to determine the current status of a particular MAAS managed node.
The function takes a fully intitialized MAASObject and a system_id.
It returns StateRefreshFunc resource ( which itself returns a copy of the
node in question, a status string and an error if needed or nil )
*/
func getNodeStatus(maas *gomaasapi.MAASObject, system_id string) resource.StateRefreshFunc {
	log.Printf("[DEBUG] [getNodeStatus] Getting stat of node: %s", system_id)
	return func() (interface{}, string, error) {
		nodeObject, err := getSingleNode(maas, system_id)
		if err != nil {
			log.Printf("[ERROR] [getNodeStatus] Unable to get node: %s\n", system_id)
			return nil, "", err
		}

		nodeStatus := strconv.FormatUint(uint64(nodeObject.status), 10)
		nodeSubStatus := strconv.FormatUint(uint64(nodeObject.substatus), 10)

		var statusRetVal bytes.Buffer
		statusRetVal.WriteString(nodeStatus)
		statusRetVal.WriteString(":")
		statusRetVal.WriteString(nodeSubStatus)

		return nodeObject, statusRetVal.String(), nil
	}
}

/*
Convenience function to get a NodeInfo object for a single MAAS node.
The function takes a fully initialized MAASObject and returns a NodeInfo, error
*/
func getSingleNode(maas *gomaasapi.MAASObject, system_id string) (*NodeInfo, error) {
	log.Printf("[DEBUG] [getSingleNode] getting node (%s) information\n", system_id)
	nodeObject, err := maasGetSingleNode(maas, system_id)
	if err != nil {
		log.Printf("[ERROR] [getSingleNode] Unable to get NodeInfo object for node: %s\n", system_id)
		return nil, err
	}

	return toNodeInfo(&nodeObject)
}

/*
Convenience function to get a NodeInfo slice of all of the nodes.
The function takes a fully initialized MAASObject and returns a slice of
all of the nodes.
*/
func getAllNodes(maas *gomaasapi.MAASObject) ([]NodeInfo, error) {
	log.Println("[DEBUG] [getAllNodes] Getting all of the MAAS managed nodes' information")
	allNodes, err := maasListAllNodes(maas)
	if err != nil {
		log.Println("[ERROR] [getAllNodes] Unable to get MAAS nodes")
		return nil, err
	}

	allNodeInfo := make([]NodeInfo, 0, 10)

	for _, nodeObj := range allNodes {
		maasObject, err := nodeObj.GetMAASObject()
		if err != nil {
			log.Println("[ERROR] [getAllNodes] Unable to get MAASObject object")
			return nil, err
		}

		node, err := toNodeInfo(&maasObject)
		if err != nil {
			log.Println("[ERROR] [getAllNodes] Unable to get NodeInfo object for node")
			return nil, err
		}

		allNodeInfo = append(allNodeInfo, *node)

	}
	return allNodeInfo, err
}

func nodeDo(maas *gomaasapi.MAASObject, system_id string, action string, params url.Values) error {
	log.Printf("[DEBUG] [nodeDo] system_id: %s, action: %s, params: %+v", system_id, action, params)

	nodeObject, err := maasGetSingleNode(maas, system_id)
	if err != nil {
		log.Printf("[ERROR] [nodeDo] Unable to get node (%s) information.\n", system_id)
		return err
	}

	_, err = nodeObject.CallPost(action, params)
	if err != nil {
		log.Printf("[ERROR] [nodeDo] Unable to perform action (%s) on node (%s).  Failed withh error (%s)\n", action, system_id, err)
		return err
	}
	return nil
}

func nodesAllocate(maas *gomaasapi.MAASObject, params url.Values) (*NodeInfo, error) {
	log.Println("[DEBUG] [nodesAllocate] Attempting to allocate one or more MAAS managed nodes")

	maasNodesObject, err := maasAllocateNodes(maas, params)
	if err != nil {
		log.Printf("[ERROR] [nodesAllocate] Unable to allocate node ... bailing")
		return nil, err
	}

	return toNodeInfo(&maasNodesObject)
}

func nodeRelease(maas *gomaasapi.MAASObject, system_id string) error {
	return maasReleaseNode(maas, system_id)
}

func parseConstraints(d *schema.ResourceData) (url.Values, error) {
	log.Printf("[DEBUG] [parseConstraints] Parsing any existing MAAS constraints")
	retVal := url.Values{}

	hostname, set := d.GetOk("hostname")
	if set {
		log.Printf("[DEBUG] [parseConstraints] setting hostname to %+v", hostname)
		retVal["name"] = strings.Fields(hostname.(string))
	}

	architecture, set := d.GetOk("architecture")
	if set {
		log.Printf("[DEBUG] [parseConstraints] Setting architecture to %s", architecture)
		retVal["arch"] = strings.Fields(architecture.(string))
	}

	cpu_count, set := d.GetOk("cpu_count")
	if set {
		retVal["cpu_count"] = strings.Fields(cpu_count.(string))
	}

	memory, set := d.GetOk("memory")
	if set {
		retVal["memory"] = strings.Fields(memory.(string))
	}

	//TODO(negronjl): Need to ensure the value of tag_names is actually a list and not something unexpected
	tags, set := d.GetOk("tag_names")
	if set {
		retVal["tags"] = strings.Fields(tags.(string))
	}

	//TODO(negronjl): Complete the list based on https://maas.ubuntu.com/docs/api.html

	return retVal, nil
}
