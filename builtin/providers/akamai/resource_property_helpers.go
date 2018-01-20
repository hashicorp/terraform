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
	"github.com/hashicorp/terraform/helper/schema"
)

func getProperty(d *schema.ResourceData) (*papi.Property, error) {
	log.Println("[DEBUG] Fetching property")
	propertyId := d.Id()
	property := papi.NewProperty(papi.NewProperties())
	property.PropertyID = propertyId
	e := property.GetProperty()
	return property, e
}

func getGroup(d *schema.ResourceData) (*papi.Group, error) {
	log.Println("[DEBUG] Fetching groups")
	groupId := d.Get("group_id").(string)

	groups := papi.NewGroups()
	e := groups.GetGroups()
	if e != nil {
		return nil, e
	}

	group, e := groups.FindGroup(groupId)
	if e != nil {
		return nil, e
	}

	log.Printf("[DEBUG] Group found: %s\n", group.GroupID)
	return group, nil
}

func getContract(d *schema.ResourceData) (*papi.Contract, error) {
	log.Println("[DEBUG] Fetching contract")
	contractId := d.Get("contract_id").(string)

	contracts := papi.NewContracts()
	e := contracts.GetContracts()
	if e != nil {
		return nil, e
	}

	contract, e := contracts.FindContract(contractId)
	if e != nil {
		return nil, e
	}

	log.Printf("[DEBUG] Contract found: %s\n", contract.ContractID)
	return contract, nil
}

func getProduct(d *schema.ResourceData, contract *papi.Contract) (*papi.Product, error) {
	log.Println("[DEBUG] Fetching product")
	productId := d.Get("product_id").(string)

	products := papi.NewProducts()
	e := products.GetProducts(contract)
	if e != nil {
		return nil, e
	}

	product, e := products.FindProduct(productId)
	if e != nil {
		return nil, e
	}

	log.Printf("[DEBUG] Product found: %s\n", product.ProductID)
	return product, nil
}

func getCloneFrom(d *schema.ResourceData, group *papi.Group, contract *papi.Contract) (*papi.ClonePropertyFrom, error) {
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
	property.Group = group
	property.Contract = contract
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

func createCpCode(contract *papi.Contract, group *papi.Group, product *papi.Product, d *schema.ResourceData) (*papi.CpCode, error) {
	log.Println("[DEBUG] Setting up CPCode")
	cpCodes := papi.NewCpCodes(contract, group)
	cpCode := papi.NewCpCode(cpCodes)
	cpCode.CpcodeID = d.Get("cp_code").(string)
	if !strings.HasPrefix(cpCode.CpcodeID, "cpc_") {
		cpCode.CpcodeID = "cpc_" + cpCode.CpcodeID
	}
	if err := cpCode.GetCpCode(); err != nil {
		cpCode.CpcodeID = ""
		cpCodes.GetCpCodes()
		cpCode, err := cpCodes.FindCpCode(d.Get("cp_code").(string))
		if err != nil {
			return nil, err
		}

		if cpCode == nil {
			log.Println("[DEBUG] CPCode not found, creating a new one")
			cpCode = papi.NewCpCode(cpCodes)
			cpCode.ProductID = product.ProductID
			cpCode.CpcodeName = d.Get("cp_code").(string)
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

func createOrigin(d *schema.ResourceData) (*papi.OptionValue, error) {
	log.Println("[DEBUG] Setting origin")
	if origin, ok := d.GetOk("origin"); ok {
		originConfig := origin.([]interface{})[0].(map[string]interface{})
		forwardHostname := originConfig["forward_hostname"].(string)
		var originValues *papi.OptionValue
		if forwardHostname == "ORIGIN_HOSTNAME" || forwardHostname == "REQUEST_HOST_HEADER" {
			log.Println("[DEBUG] Setting non-custom forward hostname")
			originValues = &papi.OptionValue{
				"originType":         "CUSTOMER",
				"hostname":           originConfig["hostname"].(string),
				"httpPort":           originConfig["port"].(int),
				"forwardHostHeader":  forwardHostname,
				"cacheKeyHostname":   originConfig["cache_key_hostname"].(string),
				"compress":           originConfig["gzip_compression"].(bool),
				"enableTrueClientIp": originConfig["true_client_ip_header"].(bool),
			}
		} else {
			log.Println("[DEBUG] Setting custom forward hostname")
			originValues = &papi.OptionValue{
				"originType":              "CUSTOMER",
				"hostname":                originConfig["hostname"].(string),
				"httpPort":                originConfig["port"].(string),
				"forwardHostHeader":       "CUSTOM",
				"customForwardHostHeader": forwardHostname,
				"cacheKeyHostname":        originConfig["cache_key_hostname"].(string),
				"compress":                originConfig["gzip_compression"].(bool),
				"enableTrueClientIp":      originConfig["true_client_ip_header"].(bool),
			}
		}
		return originValues, nil
	}
	return nil, errors.New("No origin config found")
}

func fixupPerformanceBehaviors(rules *papi.Rules) {
	foundSureRoute := false
	if rule, err := rules.FindRule("/Performance"); err == nil && rule != nil {
		for _, behavior := range rule.Behaviors {
			if behavior.Name == "sureRoute" && behavior.Options["testObjectUrl"] != "" {
				foundSureRoute = true
				break
			}
		}

		if !foundSureRoute {
			return
		}
	}

	log.Println("[DEBUG] Fixing Up SureRoute Behavior")
	r := papi.NewRule()
	r.Name = "Performance"

	b := papi.NewBehavior()
	b.Name = "sureRoute"
	b.Options = papi.OptionValue{
		"testObjectUrl":   "/akamai/sureroute-testobject.html",
		"enableCustomKey": false,
		"enabled":         false,
	}
	r.AddBehavior(b)
	rules.Rule.AddChildRule(r)
}

func updateStandardBehaviors(rules *papi.Rules, cpCode *papi.CpCode, origin *papi.OptionValue) {
	b := papi.NewBehavior()
	b.Name = "cpCode"
	b.Options = papi.OptionValue{
		"value": papi.OptionValue{
			"id": cpCode.ID(),
		},
	}
	rules.Rule.AddBehavior(b)

	b = papi.NewBehavior()
	b.Name = "origin"
	b.Options = *origin
	rules.Rule.AddBehavior(b)
}

func createHostnames(property *papi.Property, product *papi.Product, d *schema.ResourceData) (map[string]*papi.EdgeHostname, error) {
	// If the property has edge hostnames and none is specified in the schema, then don't update them
	edgeHostname, edgeHostnameOk := d.GetOk("edge_hostname")
	if edgeHostnameOk != false {
		hostnames, err := property.GetHostnames(nil)
		if err != nil {
			return nil, err
		}

		if len(hostnames.Hostnames.Items) > 0 {
			return make(map[string]*papi.EdgeHostname), nil
		}
	}

	hostnames := d.Get("hostname").(*schema.Set).List()
	ipv6 := d.Get("ipv6").(bool)

	log.Println("[DEBUG] Figuring out hostnames")
	edgeHostnames := papi.NewEdgeHostnames()
	edgeHostnames.GetEdgeHostnames(property.Contract, property.Group, "")

	hostnameEdgeHostnameMap := map[string]*papi.EdgeHostname{}
	defaultEdgeHostname := edgeHostnames.EdgeHostnames.Items[0]

	if edgeHostnameOk {
		foundEdgeHostname := false
		for _, eHn := range edgeHostnames.EdgeHostnames.Items {
			if eHn.EdgeHostnameDomain == edgeHostname.(string) {
				foundEdgeHostname = true
				defaultEdgeHostname = eHn
			}
		}

		if foundEdgeHostname == false {
			var err error
			defaultEdgeHostname, err = createEdgehostname(edgeHostnames, product, edgeHostname.(string), ipv6)
			if err != nil {
				return nil, err
			}
		}

		for _, hostname := range hostnames {
			if _, ok := hostnameEdgeHostnameMap[hostname.(string)]; !ok {
				hostnameEdgeHostnameMap[hostname.(string)] = defaultEdgeHostname
				return hostnameEdgeHostnameMap, nil
			}
		}
	}

	// Contract/Group has _some_ Edge Hostnames, try to map 1:1 (e.g. example.com -> example.com.edgesuite.net)
	// If some mapping exists, map non-existent ones to the first 1:1 we find, otherwise if none exist map to the
	// first Edge Hostname found in the contract/group
	if len(edgeHostnames.EdgeHostnames.Items) > 0 {
		log.Println("[DEBUG] Hostnames retrieved, trying to map")
		edgeHostnamesMap := map[string]*papi.EdgeHostname{}

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
		newEdgeHostname, err := createEdgehostname(edgeHostnames, product, hostnames[0].(string), ipv6)
		if err != nil {
			return nil, err
		}

		for _, hostname := range hostnames {
			hostnameEdgeHostnameMap[hostname.(string)] = newEdgeHostname
		}

		log.Printf("[DEBUG] Edgehostname created: %s\n", newEdgeHostname.EdgeHostnameDomain)
	}

	return hostnameEdgeHostnameMap, nil
}

func createEdgehostname(edgeHostnames *papi.EdgeHostnames, product *papi.Product, hostname string, ipv6 bool) (*papi.EdgeHostname, error) {
	newEdgeHostname := papi.NewEdgeHostname(edgeHostnames)
	newEdgeHostname.ProductID = product.ProductID
	newEdgeHostname.IPVersionBehavior = "IPV4"
	if ipv6 {
		newEdgeHostname.IPVersionBehavior = "IPV6_COMPLIANCE"
	}

	newEdgeHostname.EdgeHostnameDomain = hostname
	newEdgeHostname.Save("")

	go newEdgeHostname.PollStatus("")

	for newEdgeHostname.Status != papi.StatusActive {
		select {
		case <-newEdgeHostname.StatusChange:
		case <-time.After(time.Minute * 20):
			return nil, fmt.Errorf("No Edge Hostname found and a timeout occurred trying to create \"%s.%s\"", newEdgeHostname.DomainPrefix, newEdgeHostname.DomainSuffix)
		}
	}

	return newEdgeHostname, nil
}

func setEdgeHostnames(property *papi.Property, hostnameEdgeHostnameMap map[string]*papi.EdgeHostname) (map[string]string, error) {
	log.Println("[DEBUG] Setting Edge Hostnames")
	version := papi.NewVersion(papi.NewVersions())
	version.PropertyVersion = property.LatestVersion
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
		ehn[strings.Replace(from, ".", "-", -1)] = to.EdgeHostnameDomain
	}
	log.Println("[DEBUG] Saving edge hostnames")
	err = propertyHostnames.Save()
	log.Println("[DEBUG] Edge hostnames saved")
	if err != nil {
		return nil, err
	}

	return ehn, nil
}

func unmarshalRules(d *schema.ResourceData, rules *papi.Rules) {
	// DEFAULT RULES
	def, ok := d.GetOk("default")
	if ok {
		for _, v := range def.(*schema.Set).List() {
			vv, ok := v.(map[string]interface{})
			if ok {
				dbehavior, ok := vv["behavior"]
				if ok {
					for _, b := range dbehavior.(*schema.Set).List() {
						bb, ok := b.(map[string]interface{})
						if ok {
							beh := papi.NewBehavior()
							beh.Name = bb["name"].(string)
							boptions, ok := bb["option"]
							if ok {
								beh.Options = extractOptions(boptions.(*schema.Set))
							}
							rules.Rule.AddBehavior(beh)
						}
					}
				}

				dcriteria, ok := vv["criteria"]
				if ok {
					for _, b := range dcriteria.(*schema.Set).List() {
						bb, ok := b.(map[string]interface{})
						if ok {
							beh := papi.NewCriteria()
							beh.Name = bb["name"].(string)
							coptions, ok := bb["option"]
							if ok {
								beh.Options = extractOptions(coptions.(*schema.Set))
							}
							rules.Rule.AddCriteria(beh)
						}
					}
				}
			}
		}
	}

	// ALL OTHER RULES
	drules, ok := d.GetOk("rule")
	if ok {
		for _, rule := range extractRules(drules.(*schema.Set)) {
			rules.Rule.AddChildRule(rule)
		}
	}
}

func extractOptions(options *schema.Set) map[string]interface{} {
	optv := make(map[string]interface{})
	for _, o := range options.List() {
		oo, ok := o.(map[string]interface{})
		if ok {
			vals, ok := oo["values"]
			if ok && vals.(*schema.Set).Len() > 0 {
				op := make([]interface{}, 0)
				for _, v := range vals.(*schema.Set).List() {
					op = append(op, numberify(v.(string)))
				}

				optv[oo["key"].(string)] = op
			} else {
				optv[oo["key"].(string)] = numberify(oo["value"].(string))
			}
		}
	}
	return optv
}

func numberify(v string) interface{} {
	f1, err := strconv.ParseFloat(v, 64)
	if err == nil {
		return f1
	}

	f2, err := strconv.ParseInt(v, 10, 64)
	if err == nil {
		return f2
	}

	f3, err := strconv.ParseBool(v)
	if err == nil {
		return f3
	}

	f4, err := strconv.Atoi(v)
	if err == nil {
		return f4
	}

	return v
}

func extractRules(drules *schema.Set) []*papi.Rule {
	var rules []*papi.Rule
	for _, v := range drules.List() {
		rule := papi.NewRule()
		vv, ok := v.(map[string]interface{})
		if ok {
			rule.Name = vv["name"].(string)
			rule.Comments = vv["comment"].(string)
			dbehavior, ok := vv["behavior"]
			if ok {
				for _, b := range dbehavior.(*schema.Set).List() {
					bb, ok := b.(map[string]interface{})
					if ok {
						beh := papi.NewBehavior()
						beh.Name = bb["name"].(string)
						boptions, ok := bb["option"]
						if ok {
							beh.Options = extractOptions(boptions.(*schema.Set))
						}
						rule.AddBehavior(beh)
					}
				}
			}

			dcriteria, ok := vv["criteria"]
			if ok {
				for _, b := range dcriteria.(*schema.Set).List() {
					bb, ok := b.(map[string]interface{})
					if ok {
						beh := papi.NewCriteria()
						beh.Name = bb["name"].(string)
						coptions, ok := bb["option"]
						if ok {
							beh.Options = extractOptions(coptions.(*schema.Set))
						}
						rule.AddCriteria(beh)
					}
				}
			}

			dchildRule, ok := vv["rule"]
			if ok && dchildRule.(*schema.Set).Len() > 0 {
				for _, r := range extractRules(dchildRule.(*schema.Set)) {
					rule.AddChildRule(r)
				}
			}
		}
		rules = append(rules, rule)
	}
	return rules
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
	if err != nil {
		body, _ := json.Marshal(activation)
		log.Printf("[DEBUG] API Request Body: %s\n", string(body))
		return nil, err
	}
	log.Println("[DEBUG] Activation submitted successfully")

	return activation, nil
}

func findProperty(d *schema.ResourceData) *papi.Property {
	results, err := papi.Search(papi.SearchByPropertyName, d.Get("name").(string))
	if err != nil {
		return nil
	}

	if len(results.Versions.Items) == 0 {
		for _, hostname := range d.Get("hostname").(*schema.Set).List() {
			results, err = papi.Search(papi.SearchByHostname, hostname.(string))
			if err == nil && len(results.Versions.Items) != 0 {
				break
			}
		}

		if err != nil || len(results.Versions.Items) == 0 {
			return nil
		}
	}

	property := &papi.Property{
		PropertyID: results.Versions.Items[0].PropertyID,
		Group: &papi.Group{
			GroupID: results.Versions.Items[0].GroupID,
		},
		Contract: &papi.Contract{
			ContractID: results.Versions.Items[0].ContractID,
		},
	}

	err = property.GetProperty()
	if err != nil {
		return nil
	}

	return property
}

func ensureEditableVersion(property *papi.Property) error {
	latestVersion, err := property.GetLatestVersion("")
	if err != nil {
		return err
	}

	if latestVersion.ProductionStatus != papi.StatusInactive || latestVersion.StagingStatus != papi.StatusInactive {
		// The latest version has been activated on either production or staging, so we need to create a new version to apply changes on
		versions, err := property.GetVersions()
		if err != nil {
			return err
		}

		newVersion := versions.NewVersion(latestVersion, false)
		err = newVersion.Save()
		if err != nil {
			return err
		}
	}

	return nil
}
