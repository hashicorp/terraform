package akamai

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/akamai-open/AkamaiOPEN-edgegrid-golang/edgegrid"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceProperty() *schema.Resource {
	return &schema.Resource{
		Create: resourcePropertyCreate,
		Read:   resourcePropertyRead,
		Update: resourcePropertyCreate,
		Delete: resourcePropertyDelete,
		Exists: resourcePropertyExists,
		Importer: &schema.ResourceImporter{
			State: importRecord,
		},
		Schema: akamaiPropertySchema,
	}
}

func resourcePropertyCreate(d *schema.ResourceData, meta interface{}) error {
	papi := meta.(*Config).PapiV0Service

	group, err := getGroup(papi, d)
	if err != nil {
		return err
	}

	contract, err := getContract(papi, group, d)
	if err != nil {
		return err
	}

	cloneFrom, err := getCloneFrom(papi, d)
	if err != nil {
		return err
	}

	productId, err := getProductId(papi, d, contract)
	if err != nil {
		return err
	}

	property, err := createProperty(papi, contract, group, cloneFrom, productId, d)
	if err != nil {
		return err
	}

	hostnameEdgeHostnameMap, err := createHostnames(papi, contract, group, productId, d)
	if err != nil {
		return err
	}

	cpCode, err := createCpCode(papi, contract, group, productId, d)
	if err != nil {
		return err
	}

	err = initRules(property, cpCode, d)
	if err != nil {
		return err
	}

	edgeHostnames, err := setEdgeHostnames(papi, property, hostnameEdgeHostnameMap)
	if err != nil {
		return err
	}

	d.Set("edge_hostname", edgeHostnames)

	activation, err := activateProperty(papi, property, d)
	if err != nil {
		return err
	}

	go activation.PollStatus(property)
	for activation.Status != edgegrid.PapiStatusActive {
		select {
		case statusChanged := <-activation.StatusChange:
			if statusChanged == false {
				break
			}
			continue
		case <-time.After(time.Minute * 30):
			break
		}
	}
	log.Printf("[DEBUG] Setting ID: %s\n", property.PropertyID)
	d.SetId(property.PropertyID)

	log.Println("[DEBUG] Done")
	return resourcePropertyRead(d, meta)
}

func getCloneFrom(papi *edgegrid.PapiV0Service, d *schema.ResourceData) (*edgegrid.PapiClonePropertyFrom, error) {
	log.Println("[DEBUG] Setting up clone from")

	cF, ok := d.GetOk("clone_from")

	if !ok {
		return nil, nil
	}

	set := cF.(*schema.Set)
	cloneFrom := set.List()[0].(map[string]interface{})

	propertyId := cloneFrom["property_id"].(string)

	property := edgegrid.NewPapiProperty(edgegrid.NewPapiProperties(papi))
	property.PropertyID = propertyId
	err := property.GetProperty()
	if err != nil {
		return nil, err
	}

	version := cloneFrom["version"].(int)

	if cloneFrom["version"].(int) == 0 {
		v, err := property.GetLatestVersion("")
		if err != nil {
			return nil, err
		}
		version = v.PropertyVersion
	}

	clone := edgegrid.NewPapiClonePropertyFrom()
	clone.PropertyID = propertyId
	clone.Version = version

	if cloneFrom["etag"].(string) != "" {
		clone.CloneFromVersionEtag = cloneFrom["etag"].(string)
	}

	if cloneFrom["copy_hostnames"].(bool) != false {
		clone.CopyHostnames = true
	}

	log.Println("[DEBUG] Clone from complete")

	return clone, nil
}

func getProductId(papi *edgegrid.PapiV0Service, d *schema.ResourceData, contract *edgegrid.PapiContract) (string, error) {
	log.Println("[DEBUG] Fetching product id")

	productId, ok := d.GetOk("product_id")
	if ok {
		if strings.HasPrefix(productId.(string), "prd_") {
			return productId.(string), nil
		}

		return "prd_" + productId.(string), nil
	}

	products, err := papi.GetProducts(contract)
	if err != nil {
		return "", err
	}

	var (
		hasSPM bool
		hasDSD bool
	)

	for _, product := range products.Products.Items {
		if product.ProductID == "prd_SPM" {
			hasSPM = true
		}

		if product.ProductID == "prd_Dynamic_Site_Del" {
			hasDSD = true
		}
	}

	if hasSPM {
		log.Printf("[DEBUG] Found product id: %s\n", productId)

		return "prd_SPM", nil
	}

	if hasDSD {
		log.Printf("[DEBUG] Found product id: %s\n", productId)

		return "prd_Dynamic_Site_Del", nil
	}

	return "", fmt.Errorf("unable to determine product")
}

func getGroup(papi *edgegrid.PapiV0Service, d *schema.ResourceData) (*edgegrid.PapiGroup, error) {
	log.Println("[DEBUG] Fetching groups")
	groupId := d.Get("group").(string)

	groups := edgegrid.NewPapiGroups(papi)
	groups.GetGroups()
	group, err := groups.FindGroup(groupId)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve group \"%s\"", groupId)
	}
	log.Printf("[DEBUG] Group found: %s\n", group.GroupID)

	return group, nil
}

func getContract(papi *edgegrid.PapiV0Service, group *edgegrid.PapiGroup, d *schema.ResourceData) (*edgegrid.PapiContract, error) {
	log.Println("[DEBUG] Fetching contract")

	contractId, contractOk := d.GetOk("contract")
	contract := edgegrid.NewPapiContract(edgegrid.NewPapiContracts(papi))
	if contractOk {
		if strings.HasPrefix(contractId.(string), "ctr_") {
			contract.ContractID = contractId.(string)
		} else {
			contract.ContractID = "ctr_" + contractId.(string)
		}
		go contract.GetContract()
	} else {
		contract.ContractID = group.ContractIDs[0]
		//go contract.GetContract()
	}

	// Contract is specified in the config, but we were unable to fetch it.
	// if it wasn't specified in the config, we got it from the group so it's assumed valid.
	if contractOk && <-contract.Complete == false {
		return nil, fmt.Errorf("unable to retrieve contract \"%s\"", contractId.(string))
	}

	log.Printf("[DEBUG] Contract found: %s\n", contract.ContractID)

	return contract, nil
}

func createProperty(papi *edgegrid.PapiV0Service, contract *edgegrid.PapiContract, group *edgegrid.PapiGroup, cloneFrom *edgegrid.PapiClonePropertyFrom, productId string, d *schema.ResourceData) (*edgegrid.PapiProperty, error) {
	log.Println("[DEBUG] Creating property")

	name, nameOk := d.GetOk("name")
	hostnames := d.Get("hostname").(*schema.Set).List()

	property, err := papi.NewProperty(contract, group)
	property.ProductID = productId
	if cloneFrom != nil {
		property.CloneFrom = cloneFrom
	}

	property.PropertyName = name.(string)
	if !nameOk {
		property.PropertyName = hostnames[0].(string)
	}
	err = property.Save()
	if err != nil {
		return nil, err
	}

	log.Printf("Property created: %s\n", property.PropertyID)
	return property, nil
}

func createHostnames(papi *edgegrid.PapiV0Service, contract *edgegrid.PapiContract, group *edgegrid.PapiGroup, productId string, d *schema.ResourceData) (map[string]*edgegrid.PapiEdgeHostname, error) {
	hostnames := d.Get("hostname").(*schema.Set).List()

	ipv6, ipv6Ok := d.GetOk("ipv6")

	log.Println("[DEBUG] Figuring out hostnames")
	edgeHostnames := edgegrid.NewPapiEdgeHostnames(papi)
	edgeHostnames.GetEdgeHostnames(contract, group, "")

	hostnameEdgeHostnameMap := map[string]*edgegrid.PapiEdgeHostname{}

	// Contract/Group has _some_ Edge Hostnames, try to map 1:1 (e.g. example.com -> example.com.edgesuite.net)
	// If some mapping exists, map non-existent ones to the first 1:1 we find, otherwise if none exist map to the
	// first Edge Hostname found in the contract/group
	if len(edgeHostnames.EdgeHostnames.Items) > 0 {
		log.Println("[DEBUG] Hostnames retrieved, trying to map")
		edgeHostnamesMap := map[string]*edgegrid.PapiEdgeHostname{}

		defaultEdgeHostname := edgeHostnames.EdgeHostnames.Items[0]

		for _, edgeHostname := range edgeHostnames.EdgeHostnames.Items {
			edgeHostnamesMap[edgeHostname.EdgeHostnameDomain] = edgeHostname
		}

		// Search for existing hostname, map 1:1
		var overrideDefault bool
		for _, hostname := range hostnames {
			if edgeHostname, ok := edgeHostnamesMap[hostname.(string)+".edgesuite.net"]; ok {
				hostnameEdgeHostnameMap[hostname.(string)] = edgeHostname
				// Override the default with the first one found
				if !overrideDefault {
					defaultEdgeHostname = edgeHostname
					overrideDefault = true
				}
				continue
			}

			/* Support for secure properties
			if (property is secure) {
				if edgeHostname, ok := edgeHostnamesMap[hostname.(string)+".edgekey.net"]; ok {
					hostnameEdgeHostnameMap[hostname.(string)] = edgeHostname
				}
			}
			*/
		}

		// Fill in defaults
		if len(hostnameEdgeHostnameMap) < len(hostnames) {
			log.Printf("[DEBUG] Hostnames being set to default: %d of %d\n", len(hostnameEdgeHostnameMap), len(hostnames))
			for _, hostname := range hostnames {
				if _, ok := hostnameEdgeHostnameMap[hostname.(string)]; !ok {
					hostnameEdgeHostnameMap[hostname.(string)] = defaultEdgeHostname
				}
			}
		}
	}

	// Contract/Group has no Edge Hostnames, create a single based on the first hostname
	// mapping example.com -> example.com.edgegrid.net
	if len(edgeHostnames.EdgeHostnames.Items) == 0 {
		log.Println("[DEBUG] No Edge Hostnames found, creating new one")
		newEdgeHostname := edgegrid.NewPapiEdgeHostname(edgeHostnames)
		newEdgeHostname.ProductID = productId
		newEdgeHostname.IPVersionBehavior = "IPV4"
		if ipv6Ok && ipv6.(bool) {
			newEdgeHostname.IPVersionBehavior = "IPV6_COMPLIANCE"
		}

		newEdgeHostname.DomainPrefix = hostnames[0].(string)
		newEdgeHostname.DomainSuffix = "edgesuite.net"
		newEdgeHostname.Save("")

		go newEdgeHostname.PollStatus("")

		for newEdgeHostname.Status != edgegrid.PapiStatusActive {
			select {
			case <-newEdgeHostname.StatusChange:
			case <-time.After(time.Minute * 20):
				return nil, fmt.Errorf("No Edge Hostname found and a timeout occurred trying to create \"%s.%s\"", newEdgeHostname.DomainPrefix, newEdgeHostname.DomainSuffix)
			}
		}

		for _, hostname := range hostnames {
			hostnameEdgeHostnameMap[hostname.(string)] = newEdgeHostname
		}

		log.Printf("[DEBUG] Edgehostname created: %s\n", newEdgeHostname.EdgeHostnameDomain)
	}

	return hostnameEdgeHostnameMap, nil
}

func createCpCode(papi *edgegrid.PapiV0Service, contract *edgegrid.PapiContract, group *edgegrid.PapiGroup, productId string, d *schema.ResourceData) (*edgegrid.PapiCpCode, error) {
	log.Println("[DEBUG] Setting up CPCode")
	cpCodes := edgegrid.NewPapiCpCodes(papi, contract, group)
	cpCode := edgegrid.NewPapiCpCode(cpCodes)
	cpCode.CpcodeID = d.Get("cpcode").(string)
	if !strings.HasPrefix(cpCode.CpcodeID, "cpc_") {
		cpCode.CpcodeID = "cpc_" + cpCode.CpcodeID
	}
	if err := cpCode.GetCpCode(); err != nil {
		cpCode.CpcodeID = ""
		cpCodes.GetCpCodes()
		cpCode, err := cpCodes.FindCpCode(d.Get("cpcode").(string))
		if err != nil {
			return nil, err
		}

		if cpCode == nil {
			log.Println("[DEBUG] CPCode not found, creating a new one")
			cpCode = edgegrid.NewPapiCpCode(cpCodes)
			cpCode.ProductID = productId
			cpCode.CpcodeName = d.Get("cpcode").(string)
			err := cpCode.Save()
			if err != nil {
				return nil, err
			}
			log.Println("[DEBUG] CPCode created")
		}
	}
	log.Println("[DEBUG] CPCode set up")

	return cpCode, nil
}

func initRules(property *edgegrid.PapiProperty, cpCode *edgegrid.PapiCpCode, d *schema.ResourceData) error {
	log.Println("[DEBUG] Getting rules")
	rules, err := property.GetRules()
	if err != nil {
		return err
	}

	log.Println("[DEBUG] Setting CPCode")
	rules.AddBehaviorOptions("/cpCode", edgegrid.PapiOptionValue{
		"value": edgegrid.PapiOptionValue{
			"id": cpCode.ID(),
		},
	})

	log.Println("[DEBUG] Setting origin")
	originConfig := d.Get("origin").(*schema.Set).List()[0].(map[string]interface{})

	forwardHostname := originConfig["forward_hostname"].(string)
	var originValues edgegrid.PapiOptionValue
	if forwardHostname == "ORIGIN_HOSTNAME" || forwardHostname == "REQUEST_HOST_HEADER" {
		log.Println("[DEBUG] Setting non-custom forward hostname")
		originValues = edgegrid.PapiOptionValue{
			"originType":        "CUSTOMER",
			"hostname":          originConfig["hostname"].(string),
			"httpPort":          originConfig["port"].(int),
			"forwardHostHeader": forwardHostname,
		}
	} else {
		log.Println("[DEBUG] Setting custom forward hostname")
		originValues = edgegrid.PapiOptionValue{
			"originType":              "CUSTOMER",
			"hostname":                originConfig["hostname"].(string),
			"httpPort":                originConfig["port"].(string),
			"forwardHostHeader":       "CUSTOM",
			"customForwardHostHeader": forwardHostname,
		}
	}
	rules.AddBehaviorOptions("/origin", originValues)

	log.Println("[DEBUG] Setting SureRoute")
	rules.AddBehaviorOptions("/Performance/sureRoute", edgegrid.PapiOptionValue{
		"testObjectUrl":   "/akamai/sureroute-testobject.html",
		"enableCustomKey": false,
	})

	log.Println("[DEBUG] Fixing Image compression settings")
	rules.AddBehaviorOptions("/Performance/JPEG Images/adaptiveImageCompression", edgegrid.PapiOptionValue{
		"tier1MobileCompressionMethod": "BYPASS",
		"tier2MobileCompressionMethod": "COMPRESS",
		"tier2MobileCompressionValue":  60,
	})

	log.Println("[DEBUG] Saving rules")

	err = rules.Save()
	if err != nil {
		return err
	}
	log.Println("[DEBUG] Rules saved")

	return nil
}

func setEdgeHostnames(papi *edgegrid.PapiV0Service, property *edgegrid.PapiProperty, hostnameEdgeHostnameMap map[string]*edgegrid.PapiEdgeHostname) (map[string]string, error) {
	log.Println("[DEBUG] Setting Edge Hostnames")
	version := edgegrid.NewPapiVersion(edgegrid.NewPapiVersions(papi))
	version.PropertyVersion = 1
	propertyHostnames, err := property.GetHostnames(version)
	if err != nil {
		return nil, err
	}

	var ehn map[string]string = make(map[string]string)
	propertyHostnames.Hostnames.Items = []*edgegrid.PapiHostname{}
	for from, to := range hostnameEdgeHostnameMap {
		hostname := propertyHostnames.NewHostname()
		hostname.CnameType = edgegrid.PapiCnameTypeEdgeHostname
		hostname.CnameFrom = from
		hostname.CnameTo = to.EdgeHostnameDomain
		hostname.EdgeHostnameID = to.EdgeHostnameID
		ehn[from] = to.EdgeHostnameDomain
	}
	log.Println("[DEBUG] Saving edge hostnames")
	err = propertyHostnames.Save()
	log.Println("[DEBUG] Edge hostnames saved")
	if err != nil {
		return nil, err
	}

	return ehn, nil
}

func activateProperty(papi *edgegrid.PapiV0Service, property *edgegrid.PapiProperty, d *schema.ResourceData) (*edgegrid.PapiActivation, error) {
	log.Println("[DEBUG] Creating new activation")
	activation := edgegrid.NewPapiActivation(edgegrid.NewPapiActivations(papi))
	activation.PropertyVersion = property.LatestVersion
	activation.Network = edgegrid.PapiNetworkStaging
	for _, email := range d.Get("contact").(*schema.Set).List() {
		activation.NotifyEmails = append(activation.NotifyEmails, email.(string))
	}
	activation.Note = "Using Terraform"
	log.Println("[DEBUG] Activating")
	err := activation.Save(property, true)
	log.Println("[DEBUG] Activation submitted")
	if err != nil {
		return nil, err
	}

	return activation, nil
}

func resourcePropertyRead(d *schema.ResourceData, meta interface{}) error {
	log.Println("[DEBUG] resourcePropertyRead")
	return nil
}

func resourcePropertyDelete(d *schema.ResourceData, meta interface{}) error {
	return errors.New("resourcePropertyDelete")
}

func resourcePropertyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	return false, errors.New("resourcePropertyExists")
}
