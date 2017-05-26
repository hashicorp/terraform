package google

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
)

func resourceComputeUrlMap() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeUrlMapCreate,
		Read:   resourceComputeUrlMapRead,
		Update: resourceComputeUrlMapUpdate,
		Delete: resourceComputeUrlMapDelete,

		Schema: map[string]*schema.Schema{
			"default_service": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"host_rule": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				// TODO(evandbrown): Enable when lists support validation
				//ValidateFunc: validateHostRules,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"description": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"hosts": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"path_matcher": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"path_matcher": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"default_service": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"description": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"path_rule": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"paths": &schema.Schema{
										Type:     schema.TypeList,
										Required: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},

									"service": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"test": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"description": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"host": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"path": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"service": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func createHostRule(v interface{}) *compute.HostRule {
	_hostRule := v.(map[string]interface{})

	_hosts := _hostRule["hosts"].([]interface{})
	hosts := make([]string, len(_hosts))

	for i, v := range _hosts {
		hosts[i] = v.(string)
	}

	pathMatcher := _hostRule["path_matcher"].(string)

	hostRule := &compute.HostRule{
		Hosts:       hosts,
		PathMatcher: pathMatcher,
	}

	if v, ok := _hostRule["description"]; ok {
		hostRule.Description = v.(string)
	}

	return hostRule
}

func createPathMatcher(v interface{}) *compute.PathMatcher {
	_pathMatcher := v.(map[string]interface{})

	_pathRules := _pathMatcher["path_rule"].([]interface{})
	pathRules := make([]*compute.PathRule, len(_pathRules))

	for ip, vp := range _pathRules {
		_pathRule := vp.(map[string]interface{})

		_paths := _pathRule["paths"].([]interface{})
		paths := make([]string, len(_paths))

		for ipp, vpp := range _paths {
			paths[ipp] = vpp.(string)
		}

		service := _pathRule["service"].(string)

		pathRule := &compute.PathRule{
			Paths:   paths,
			Service: service,
		}

		pathRules[ip] = pathRule
	}

	name := _pathMatcher["name"].(string)
	defaultService := _pathMatcher["default_service"].(string)

	pathMatcher := &compute.PathMatcher{
		PathRules:      pathRules,
		Name:           name,
		DefaultService: defaultService,
	}

	if vp, okp := _pathMatcher["description"]; okp {
		pathMatcher.Description = vp.(string)
	}

	return pathMatcher
}

func createUrlMapTest(v interface{}) *compute.UrlMapTest {
	_test := v.(map[string]interface{})

	host := _test["host"].(string)
	path := _test["path"].(string)
	service := _test["service"].(string)

	test := &compute.UrlMapTest{
		Host:    host,
		Path:    path,
		Service: service,
	}

	if vp, okp := _test["description"]; okp {
		test.Description = vp.(string)
	}

	return test
}

func resourceComputeUrlMapCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	defaultService := d.Get("default_service").(string)

	urlMap := &compute.UrlMap{
		Name:           name,
		DefaultService: defaultService,
	}

	if v, ok := d.GetOk("description"); ok {
		urlMap.Description = v.(string)
	}

	_hostRules := d.Get("host_rule").(*schema.Set)
	urlMap.HostRules = make([]*compute.HostRule, _hostRules.Len())

	for i, v := range _hostRules.List() {
		urlMap.HostRules[i] = createHostRule(v)
	}

	_pathMatchers := d.Get("path_matcher").([]interface{})
	urlMap.PathMatchers = make([]*compute.PathMatcher, len(_pathMatchers))

	for i, v := range _pathMatchers {
		urlMap.PathMatchers[i] = createPathMatcher(v)
	}

	_tests := make([]interface{}, 0)
	if v, ok := d.GetOk("test"); ok {
		_tests = v.([]interface{})
	}
	urlMap.Tests = make([]*compute.UrlMapTest, len(_tests))

	for i, v := range _tests {
		urlMap.Tests[i] = createUrlMapTest(v)
	}

	op, err := config.clientCompute.UrlMaps.Insert(project, urlMap).Do()

	if err != nil {
		return fmt.Errorf("Error, failed to insert Url Map %s: %s", name, err)
	}

	err = computeOperationWaitGlobal(config, op, project, "Insert Url Map")

	if err != nil {
		return fmt.Errorf("Error, failed waitng to insert Url Map %s: %s", name, err)
	}

	return resourceComputeUrlMapRead(d, meta)
}

func resourceComputeUrlMapRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)

	urlMap, err := config.clientCompute.UrlMaps.Get(project, name).Do()

	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("URL Map %q", d.Get("name").(string)))
	}

	d.SetId(name)
	d.Set("self_link", urlMap.SelfLink)
	d.Set("id", strconv.FormatUint(urlMap.Id, 10))
	d.Set("fingerprint", urlMap.Fingerprint)

	hostRuleMap := make(map[string]*compute.HostRule)
	for _, v := range urlMap.HostRules {
		hostRuleMap[v.PathMatcher] = v
	}

	/* Only read host rules into our TF state that we have defined */
	_hostRules := d.Get("host_rule").(*schema.Set).List()
	_newHostRules := make([]interface{}, 0)
	for _, v := range _hostRules {
		_hostRule := v.(map[string]interface{})
		_pathMatcher := _hostRule["path_matcher"].(string)

		/* Delete local entries that are no longer found on the GCE server */
		if hostRule, ok := hostRuleMap[_pathMatcher]; ok {
			_newHostRule := make(map[string]interface{})
			_newHostRule["path_matcher"] = _pathMatcher

			hostsSet := make(map[string]bool)
			for _, host := range hostRule.Hosts {
				hostsSet[host] = true
			}

			/* Only store hosts we are keeping track of */
			_newHosts := make([]interface{}, 0)
			for _, vp := range _hostRule["hosts"].([]interface{}) {
				if _, okp := hostsSet[vp.(string)]; okp {
					_newHosts = append(_newHosts, vp)
				}
			}

			_newHostRule["hosts"] = _newHosts
			_newHostRule["description"] = hostRule.Description

			_newHostRules = append(_newHostRules, _newHostRule)
		}
	}

	d.Set("host_rule", _newHostRules)

	pathMatcherMap := make(map[string]*compute.PathMatcher)
	for _, v := range urlMap.PathMatchers {
		pathMatcherMap[v.Name] = v
	}

	/* Only read path matchers into our TF state that we have defined */
	_pathMatchers := d.Get("path_matcher").([]interface{})
	_newPathMatchers := make([]interface{}, 0)
	for _, v := range _pathMatchers {
		_pathMatcher := v.(map[string]interface{})
		_name := _pathMatcher["name"].(string)

		if pathMatcher, ok := pathMatcherMap[_name]; ok {
			_newPathMatcher := make(map[string]interface{})
			_newPathMatcher["name"] = _name
			_newPathMatcher["default_service"] = pathMatcher.DefaultService
			_newPathMatcher["description"] = pathMatcher.Description

			_newPathRules := make([]interface{}, len(pathMatcher.PathRules))
			for ip, pathRule := range pathMatcher.PathRules {
				_newPathRule := make(map[string]interface{})
				_newPathRule["service"] = pathRule.Service
				_paths := make([]interface{}, len(pathRule.Paths))

				for ipp, vpp := range pathRule.Paths {
					_paths[ipp] = vpp
				}

				_newPathRule["paths"] = _paths

				_newPathRules[ip] = _newPathRule
			}

			_newPathMatcher["path_rule"] = _newPathRules
			_newPathMatchers = append(_newPathMatchers, _newPathMatcher)
		}
	}

	d.Set("path_matcher", _newPathMatchers)

	testMap := make(map[string]*compute.UrlMapTest)
	for _, v := range urlMap.Tests {
		testMap[fmt.Sprintf("%s/%s", v.Host, v.Path)] = v
	}

	_tests := make([]interface{}, 0)
	/* Only read tests into our TF state that we have defined */
	if v, ok := d.GetOk("test"); ok {
		_tests = v.([]interface{})
	}
	_newTests := make([]interface{}, 0)
	for _, v := range _tests {
		_test := v.(map[string]interface{})
		_host := _test["host"].(string)
		_path := _test["path"].(string)

		/* Delete local entries that are no longer found on the GCE server */
		if test, ok := testMap[fmt.Sprintf("%s/%s", _host, _path)]; ok {
			_newTest := make(map[string]interface{})
			_newTest["host"] = _host
			_newTest["path"] = _path
			_newTest["description"] = test.Description
			_newTest["service"] = test.Service

			_newTests = append(_newTests, _newTest)
		}
	}

	d.Set("test", _newTests)

	return nil
}

func resourceComputeUrlMapUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	urlMap, err := config.clientCompute.UrlMaps.Get(project, name).Do()
	if err != nil {
		return fmt.Errorf("Error, failed to get Url Map %s: %s", name, err)
	}

	urlMap.DefaultService = d.Get("default_service").(string)

	if v, ok := d.GetOk("description"); ok {
		urlMap.Description = v.(string)
	}

	if d.HasChange("host_rule") {
		_oldHostRules, _newHostRules := d.GetChange("host_rule")
		_oldHostRulesMap := make(map[string]interface{})
		_newHostRulesMap := make(map[string]interface{})

		for _, v := range _oldHostRules.(*schema.Set).List() {
			_hostRule := v.(map[string]interface{})
			_oldHostRulesMap[_hostRule["path_matcher"].(string)] = v
		}

		for _, v := range _newHostRules.(*schema.Set).List() {
			_hostRule := v.(map[string]interface{})
			_newHostRulesMap[_hostRule["path_matcher"].(string)] = v
		}

		newHostRules := make([]*compute.HostRule, 0)
		/* Decide which host rules to keep */
		for _, v := range urlMap.HostRules {
			/* If it's in the old state, we have ownership over the host rule */
			if vOld, ok := _oldHostRulesMap[v.PathMatcher]; ok {
				if vNew, ok := _newHostRulesMap[v.PathMatcher]; ok {
					/* Adjust for any changes made to this rule */
					_newHostRule := vNew.(map[string]interface{})
					_oldHostRule := vOld.(map[string]interface{})
					_newHostsSet := make(map[string]bool)
					_oldHostsSet := make(map[string]bool)

					hostRule := &compute.HostRule{
						PathMatcher: v.PathMatcher,
					}

					for _, v := range _newHostRule["hosts"].([]interface{}) {
						_newHostsSet[v.(string)] = true
					}

					for _, v := range _oldHostRule["hosts"].([]interface{}) {
						_oldHostsSet[v.(string)] = true
					}

					/* Only add hosts that have been added locally or are new,
					 * not touching those from the GCE server state */
					for _, host := range v.Hosts {
						_, okNew := _newHostsSet[host]
						_, okOld := _oldHostsSet[host]

						/* Drop deleted hosts */
						if okOld && !okNew {
							continue
						}

						hostRule.Hosts = append(hostRule.Hosts, host)

						/* Kep track of the fact that this host was added */
						delete(_newHostsSet, host)
					}

					/* Now add in the brand new entries */
					for host, _ := range _newHostsSet {
						hostRule.Hosts = append(hostRule.Hosts, host)
					}

					if v, ok := _newHostRule["description"]; ok {
						hostRule.Description = v.(string)
					}

					newHostRules = append(newHostRules, hostRule)

					/* Record that we've include this host rule */
					delete(_newHostRulesMap, v.PathMatcher)
				} else {
					/* It's been deleted */
					continue
				}
			} else {
				if vNew, ok := _newHostRulesMap[v.PathMatcher]; ok {
					newHostRules = append(newHostRules, createHostRule(vNew))

					/* Record that we've include this host rule */
					delete(_newHostRulesMap, v.PathMatcher)
				} else {
					/* It wasn't created or modified locally */
					newHostRules = append(newHostRules, v)
				}
			}
		}

		/* Record brand new host rules (ones not deleted above) */
		for _, v := range _newHostRulesMap {
			newHostRules = append(newHostRules, createHostRule(v))
		}

		urlMap.HostRules = newHostRules
	}

	if d.HasChange("path_matcher") {
		_oldPathMatchers, _newPathMatchers := d.GetChange("path_matcher")
		_oldPathMatchersMap := make(map[string]interface{})
		_newPathMatchersMap := make(map[string]interface{})

		for _, v := range _oldPathMatchers.([]interface{}) {
			_pathMatcher := v.(map[string]interface{})
			_oldPathMatchersMap[_pathMatcher["name"].(string)] = v
		}

		for _, v := range _newPathMatchers.([]interface{}) {
			_pathMatcher := v.(map[string]interface{})
			_newPathMatchersMap[_pathMatcher["name"].(string)] = v
		}

		newPathMatchers := make([]*compute.PathMatcher, 0)
		/* Decide which path matchers to keep */
		for _, v := range urlMap.PathMatchers {
			/* If it's in the old state, we have ownership over the host rule */
			_, okOld := _oldPathMatchersMap[v.Name]
			vNew, okNew := _newPathMatchersMap[v.Name]

			/* Drop deleted entries */
			if okOld && !okNew {
				continue
			}

			/* Don't change entries that don't belong to us */
			if !okNew {
				newPathMatchers = append(newPathMatchers, v)
			} else {
				newPathMatchers = append(newPathMatchers, createPathMatcher(vNew))

				delete(_newPathMatchersMap, v.Name)
			}
		}

		/* Record brand new host rules */
		for _, v := range _newPathMatchersMap {
			newPathMatchers = append(newPathMatchers, createPathMatcher(v))
		}

		urlMap.PathMatchers = newPathMatchers
	}

	if d.HasChange("test") {
		_oldTests, _newTests := d.GetChange("test")
		_oldTestsMap := make(map[string]interface{})
		_newTestsMap := make(map[string]interface{})

		for _, v := range _oldTests.([]interface{}) {
			_test := v.(map[string]interface{})
			ident := fmt.Sprintf("%s/%s", _test["host"].(string), _test["path"].(string))
			_oldTestsMap[ident] = v
		}

		for _, v := range _newTests.([]interface{}) {
			_test := v.(map[string]interface{})
			ident := fmt.Sprintf("%s/%s", _test["host"].(string), _test["path"].(string))
			_newTestsMap[ident] = v
		}

		newTests := make([]*compute.UrlMapTest, 0)
		/* Decide which path matchers to keep */
		for _, v := range urlMap.Tests {
			ident := fmt.Sprintf("%s/%s", v.Host, v.Path)
			/* If it's in the old state, we have ownership over the host rule */
			_, okOld := _oldTestsMap[ident]
			vNew, okNew := _newTestsMap[ident]

			/* Drop deleted entries */
			if okOld && !okNew {
				continue
			}

			/* Don't change entries that don't belong to us */
			if !okNew {
				newTests = append(newTests, v)
			} else {
				newTests = append(newTests, createUrlMapTest(vNew))

				delete(_newTestsMap, ident)
			}
		}

		/* Record brand new host rules */
		for _, v := range _newTestsMap {
			newTests = append(newTests, createUrlMapTest(v))
		}

		urlMap.Tests = newTests
	}
	op, err := config.clientCompute.UrlMaps.Update(project, urlMap.Name, urlMap).Do()

	if err != nil {
		return fmt.Errorf("Error, failed to update Url Map %s: %s", name, err)
	}

	err = computeOperationWaitGlobal(config, op, project, "Update Url Map")

	if err != nil {
		return fmt.Errorf("Error, failed waitng to update Url Map %s: %s", name, err)
	}

	return resourceComputeUrlMapRead(d, meta)
}

func resourceComputeUrlMapDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)

	op, err := config.clientCompute.UrlMaps.Delete(project, name).Do()

	if err != nil {
		return fmt.Errorf("Error, failed to delete Url Map %s: %s", name, err)
	}

	err = computeOperationWaitGlobal(config, op, project, "Delete Url Map")

	if err != nil {
		return fmt.Errorf("Error, failed waitng to delete Url Map %s: %s", name, err)
	}

	return nil
}

func validateHostRules(v interface{}, k string) (ws []string, es []error) {
	pathMatchers := make(map[string]bool)
	hostRules := v.([]interface{})
	for _, hri := range hostRules {
		hr := hri.(map[string]interface{})
		pm := hr["path_matcher"].(string)
		if pathMatchers[pm] {
			es = append(es, fmt.Errorf("Multiple host_rule entries with the same path_matcher are not allowed. Please collapse all hosts with the same path_matcher into one host_rule"))
			return
		}
		pathMatchers[pm] = true
	}
	return
}
