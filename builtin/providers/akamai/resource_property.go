package akamai

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
	"github.com/hashicorp/terraform/builtin/providers/akamai/helper"
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
	group, err := getGroup(d)
	if err != nil {
		return err
	}

	contract, err := getContract(group, d)
	if err != nil {
		return err
	}

	cloneFrom, err := getCloneFrom(d)
	if err != nil {
		return err
	}

	productId, err := getProductId(d, contract)
	if err != nil {
		return err
	}

	property, err := createProperty(contract, group, cloneFrom, productId, d)
	if err != nil {
		return err
	}

	hostnameEdgeHostnameMap, err := createHostnames(contract, group, productId, d)
	if err != nil {
		return err
	}

	cpCode, err := createCpCode(contract, group, productId, d)
	if err != nil {
		return err
	}

	rules, err := initRules(property, cpCode, d)
	if err != nil {
		return err
	}

	if rulesSet, ok := d.GetOk("rule"); ok {
		err := updateRules(rules, "/", rulesSet.(*schema.Set))
		if err != nil {
			return err
		}

		err = rules.Save()
		if err != nil {
			return err
		}
	}

	edgeHostnames, err := setEdgeHostnames(property, hostnameEdgeHostnameMap)
	if err != nil {
		return err
	}

	d.Set("edge_hostname", edgeHostnames)

	activation, err := activateProperty(property, d)
	if err != nil {
		return err
	}

	go activation.PollStatus(property)
	for activation.Status != papi.StatusActive {
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

func getCloneFrom(d *schema.ResourceData) (*papi.ClonePropertyFrom, error) {
	log.Println("[DEBUG] Setting up clone from")

	cF, ok := d.GetOk("clone_from")

	if !ok {
		return nil, nil
	}

	set := cF.(*schema.Set)
	cloneFrom := set.List()[0].(map[string]interface{})

	propertyId := cloneFrom["property_id"].(string)

	property := papi.NewProperty(papi.NewProperties())
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

	clone := papi.NewClonePropertyFrom()
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

func getProductId(d *schema.ResourceData, contract *papi.Contract) (string, error) {
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

func getGroup(d *schema.ResourceData) (*papi.Group, error) {
	log.Println("[DEBUG] Fetching groups")
	groupId := d.Get("group").(string)

	groups := papi.NewGroups()
	groups.GetGroups()
	group, err := groups.FindGroup(groupId)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve group \"%s\"", groupId)
	}
	log.Printf("[DEBUG] Group found: %s\n", group.GroupID)

	return group, nil
}

func getContract(group *papi.Group, d *schema.ResourceData) (*papi.Contract, error) {
	log.Println("[DEBUG] Fetching contract")

	contractId, contractOk := d.GetOk("contract")
	contract := papi.NewContract(papi.NewContracts())
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

func createProperty(contract *papi.Contract, group *papi.Group, cloneFrom *papi.ClonePropertyFrom, productId string, d *schema.ResourceData) (*papi.Property, error) {
	log.Println("[DEBUG] Creating property")

	name, nameOk := d.GetOk("name")
	hostnames := d.Get("hostname").(*schema.Set).List()

	property, err := group.NewProperty(contract)
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

func createHostnames(contract *papi.Contract, group *papi.Group, productId string, d *schema.ResourceData) (map[string]*papi.EdgeHostname, error) {
	hostnames := d.Get("hostname").(*schema.Set).List()

	ipv6, ipv6Ok := d.GetOk("ipv6")

	log.Println("[DEBUG] Figuring out hostnames")
	edgeHostnames := papi.NewEdgeHostnames()
	edgeHostnames.GetEdgeHostnames(contract, group, "")

	hostnameEdgeHostnameMap := map[string]*papi.EdgeHostname{}

	// Contract/Group has _some_ Edge Hostnames, try to map 1:1 (e.g. example.com -> example.com.edgesuite.net)
	// If some mapping exists, map non-existent ones to the first 1:1 we find, otherwise if none exist map to the
	// first Edge Hostname found in the contract/group
	if len(edgeHostnames.EdgeHostnames.Items) > 0 {
		log.Println("[DEBUG] Hostnames retrieved, trying to map")
		edgeHostnamesMap := map[string]*papi.EdgeHostname{}

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
		newEdgeHostname := papi.NewEdgeHostname(edgeHostnames)
		newEdgeHostname.ProductID = productId
		newEdgeHostname.IPVersionBehavior = "IPV4"
		if ipv6Ok && ipv6.(bool) {
			newEdgeHostname.IPVersionBehavior = "IPV6_COMPLIANCE"
		}

		newEdgeHostname.DomainPrefix = hostnames[0].(string)
		newEdgeHostname.DomainSuffix = "edgesuite.net"
		newEdgeHostname.Save("")

		go newEdgeHostname.PollStatus("")

		for newEdgeHostname.Status != papi.StatusActive {
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

func createCpCode(contract *papi.Contract, group *papi.Group, productId string, d *schema.ResourceData) (*papi.CpCode, error) {
	log.Println("[DEBUG] Setting up CPCode")
	cpCodes := papi.NewCpCodes(contract, group)
	cpCode := papi.NewCpCode(cpCodes)
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
			cpCode = papi.NewCpCode(cpCodes)
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

func initRules(property *papi.Property, cpCode *papi.CpCode, d *schema.ResourceData) (*papi.Rules, error) {
	log.Println("[DEBUG] Getting rules")
	rules, err := property.GetRules()
	if err != nil {
		return nil, err
	}

	log.Println("[DEBUG] Setting CPCode")
	rules.AddBehaviorOptions("/cpCode", papi.OptionValue{
		"value": papi.OptionValue{
			"id": cpCode.ID(),
		},
	})

	log.Println("[DEBUG] Setting origin")
	if origin, ok := d.GetOk("origin"); ok {
		originConfig := origin.(*schema.Set).List()[0].(map[string]interface{})

		forwardHostname := originConfig["forward_hostname"].(string)
		var originValues papi.OptionValue
		if forwardHostname == "ORIGIN_HOSTNAME" || forwardHostname == "REQUEST_HOST_HEADER" {
			log.Println("[DEBUG] Setting non-custom forward hostname")
			originValues = papi.OptionValue{
				"originType":        "CUSTOMER",
				"hostname":          originConfig["hostname"].(string),
				"httpPort":          originConfig["port"].(int),
				"forwardHostHeader": forwardHostname,
			}
		} else {
			log.Println("[DEBUG] Setting custom forward hostname")
			originValues = papi.OptionValue{
				"originType":              "CUSTOMER",
				"hostname":                originConfig["hostname"].(string),
				"httpPort":                originConfig["port"].(string),
				"forwardHostHeader":       "CUSTOM",
				"customForwardHostHeader": forwardHostname,
			}
		}
		rules.AddBehaviorOptions("/origin", originValues)
	}

	log.Println("[DEBUG] Setting SureRoute")
	rules.AddBehaviorOptions("/Performance/sureRoute", papi.OptionValue{
		"testObjectUrl":   "/akamai/sureroute-testobject.html",
		"enableCustomKey": false,
	})

	log.Println("[DEBUG] Fixing Image compression settings")
	rules.AddBehaviorOptions("/Performance/JPEG Images/adaptiveImageCompression", papi.OptionValue{
		"tier1MobileCompressionMethod": "BYPASS",
		"tier2MobileCompressionMethod": "COMPRESS",
		"tier2MobileCompressionValue":  60,
	})
	log.Println("[DEBUG] Saving rules")

	err = rules.Save()
	if err != nil {
		errorJson, err := json.MarshalIndent(rules.Errors, "", "    ")
		if err != nil {
			log.Printf("[DEBUG] Rules.Errors: %s\n", string(errorJson))
		}
		return rules, err
	}
	log.Println("[DEBUG] Rules saved")

	return rules, nil
}

func updateRules(rules *papi.Rules, path string, rulesSet *schema.Set) error {
	log.Printf("[DEBUG] Updating rules at path: %s\n", path)
	if rulesSet.Len() == 0 {
		return nil
	}

	for _, ruleElem := range helper.ListSet(rulesSet) {
		rule := papi.NewRule(rules)

		rule.Name = "default"
		if ruleElem.Contains("name") {
			rule.Name = ruleElem.GetString("name")
		}

		if ruleElem.Contains("comment") {
			rule.Comments = ruleElem.GetString("comment")
		}

		if ruleElem.Contains("criteria") {
			criteriaElem := ruleElem.Get("criteria").(*schema.Set)

			for _, v := range helper.ListSet(criteriaElem) {
				criteria := papi.NewCriteria(rule)
				criteria.Name = v.GetString("name")

				options, err := getOptions(v)
				if err != nil {
					return err
				}
				criteria.Options = &options
				rule.AddCriteria(criteria)
			}
		}

		if ruleElem.Contains("behavior") {
			behaviorElem := ruleElem.Get("behavior").(*schema.Set)

			for _, v := range helper.ListSet(behaviorElem) {
				behavior := papi.NewBehavior(rule)
				behavior.Name = v.GetString("name")

				options, err := getOptions(v)
				if err != nil {
					return err
				}
				behavior.Options = &options
				rule.AddBehavior(behavior)
			}
		}

		log.Printf("[DEBUG] Saving rule: %s\n\n", path+"/")
		log.Printf("[DEBUG] Rule: %#v\n\n", rule)
		err := rules.AddChildRule(path, rule)
		if err != nil {
			return err
		}

		if ruleElem.Contains("rule") && ruleElem.GetSet("rule").Len() > 0 {
			err := updateRules(rules, path+"/"+rule.Name, ruleElem.GetSet("rule"))
			if err != nil {
				errorJson, err := json.MarshalIndent(rules.Errors, "", "    ")
				if err != nil {
					log.Printf("[DEBUG] Update Rules.Errors: %s\n", string(errorJson))
				}
				return err
			}
		} else {
			log.Println("[DEBUG] Child rules not found")
		}
	}

	return nil
}

func getOptions(v helper.Elem) (papi.OptionValue, error) {
	option := papi.OptionValue{}
	for _, optionElem := range helper.ListSet(v.GetSet("option")) {
		if v == nil || optionElem.GetString("name") == "" {
			continue
		}
		name := optionElem.GetString("name")

		optionType := optionElem.GetString("type")

		if optionType == "auto" {
			switch {
			case optionElem.Contains("values") && optionElem.GetSet("values").Len() > 0:
				optionType = "set"
			case optionElem.Contains("value") && optionElem.GetString("value") != "":
				optionType = "string"
			case optionElem.Contains("flag"):
				optionType = "bool"
			default:
				return nil, fmt.Errorf("Invalid value for option \"%s\"", name)
			}
		}

		switch {
		case optionType == "string":
			value := optionElem.GetString("value")
			option[name] = value
		case optionType == "array" || optionType == "set":
			var values []interface{}
			for _, value := range optionElem.GetSet("values").List() {
				if v, ok := value.(string); ok {
					values = append(values, numberify(v))
				}
			}
			option[name] = values
		case optionType == "int" || optionType == "integer":
			value, err := strconv.Atoi(optionElem.GetString("value"))
			if err != nil {
				return nil, err
			}

			option[name] = value
		case optionType == "float" || optionType == "double":
			value, err := strconv.ParseFloat(optionElem.GetString("value"), 64)
			if err != nil {
				return nil, err
			}

			option[name] = value
		case optionType == "number":
			option[name] = numberify(optionElem.GetString("value"))
		case optionType == "bool" || optionType == "boolean" || optionType == "flag":
			option[name] = optionElem.GetBool("value")
		}
	}

	return option, nil
}

func numberify(v string) interface{} {
	i, err := strconv.Atoi(v)
	if err == nil {
		return i
	}

	f, err := strconv.ParseFloat(v, 64)
	if err == nil {
		return f
	}

	return v
}

func setEdgeHostnames(property *papi.Property, hostnameEdgeHostnameMap map[string]*papi.EdgeHostname) (map[string]string, error) {
	log.Println("[DEBUG] Setting Edge Hostnames")
	version := papi.NewVersion(papi.NewVersions())
	version.PropertyVersion = 1
	propertyHostnames, err := property.GetHostnames(version)
	if err != nil {
		return nil, err
	}

	var ehn map[string]string = make(map[string]string)
	propertyHostnames.Hostnames.Items = []*papi.Hostname{}
	for from, to := range hostnameEdgeHostnameMap {
		hostname := propertyHostnames.NewHostname()
		hostname.CnameType = papi.CnameTypeEdgeHostname
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

func activateProperty(property *papi.Property, d *schema.ResourceData) (*papi.Activation, error) {
	log.Println("[DEBUG] Creating new activation")
	activation := papi.NewActivation(papi.NewActivations())
	activation.PropertyVersion = property.LatestVersion
	activation.Network = papi.NetworkValue(strings.ToUpper(d.Get("network").(string)))
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
