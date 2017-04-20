package icsp

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

// FailModeData stage const
type FailModeData int

const (
	FM_ABORT FailModeData = 1 + iota
	FM_IGNORE
)

var failmodelist = [...]string{
	"ABORT",
	"IGNORE",
}

// String helper for stage
func (o FailModeData) String() string { return failmodelist[o-1] }

// Equal helper for stage
func (o FailModeData) Equal(s string) bool { return (strings.ToUpper(s) == strings.ToUpper(o.String())) }

// OSDNicDataV2 network interface
type OSDNicDataV2 struct {
	DHCPv4         bool     `json:"dhcpv4,omitempty"`         // interfaces[]dhcpv4 Boolean
	DNSSearch      []string `json:"dnsSearch,omitempty"`      // interfaces[]dnsSearch array of string
	DNSServers     []string `json:"dnsServers,omitempty"`     // interfaces[]dnsServers array of string
	Enabled        bool     `json:"enabled,omitempty"`        // interfaces[]enabled Boolean
	IPv4Gateway    string   `json:"ipv4gateway,omitempty"`    // interfaces[]ipv4gateway string
	IPv6Autoconfig bool     `json:"ipv6Autoconfig,omitempty"` // interfaces[]ipv6Autoconfig Boolean
	IPv6Gateway    string   `json:"ipv6gateway,omitempty"`    // interfaces[]ipv6gateway string
	MACAddress     string   `json:"macAddress,omitempty"`     // interfaces[]macAddress string
	StaticNetworks []string `json:"staticNetworks,omitempty"` // interfaces[]staticNetworks array of string
	VLanID         int      `json:"vlanid,omitempty"`         // interfaces[]vlanid integer
	WinsServers    []string `json:"winsServers,omitempty"`    // interfaces[]winsServers array of string
}

// OSDPersonalityDataV2 personality data
type OSDPersonalityDataV2 struct {
	Domain            string         `json:"domain,omitempty"`            // domain string
	HostName          string         `json:"hostname,omitempty"`          // hostname string
	Interfaces        []OSDNicDataV2 `json:"interfaces,omitempty"`        // interfaces array of OSDNicDataV2
	VirtualInterfaces []OSDNicDataV2 `json:"virtualInterfaces,omitempty"` // virtualInterfaces array of OSDNicDataV2
	Workgroup         string         `json:"workgroup,omitempty"`         // workgroup string
}

// OSDPersonalizeServerDataV2  server data
type OSDPersonalizeServerDataV2 struct {
	PersonalityData *OSDPersonalityDataV2 `json:"personalityData,omitempty"`
	ServerURI       string                `json:"serverUri,omitempty"`  // serverUri string
	SkipReboot      bool                  `json:"skipReboot,omitempty"` // skipReboot Boolean
}

// DeploymentJobs is used for creating a new os build plan
type DeploymentJobs struct {
	FailMode   string                       `json:"failMode,omitempty"`   // failMode Selects a behavior for handling OS Build Plan failure on a server. By default, when a build plan fails on a server, it will be excluded from running any remaining build plans, and successful servers will continue to run through the series of build plans. This can be changed by setting a different failure mode.
	OsbpUris   []string                     `json:"osbpUris,omitempty"`   // osbpUris An array of OS Build Plan URIs
	ServerData []OSDPersonalizeServerDataV2 `json:"serverData,omitempty"` // server data
	StepNo     int                          `json:"stepNo,omitempty"`     // stepNo The step number in the OS build plan from which to start execution. integer
}

func (dj DeploymentJobs) NewDeploymentJobs(bp []OSDBuildPlan, bpdata *OSDPersonalityDataV2, servers []Server) DeploymentJobs {
	var bpURI []string
	var sd []OSDPersonalizeServerDataV2

	for _, plan := range bp {
		bpURI = append(bpURI, plan.URI.String())
	}

	for _, s := range servers {
		var pd OSDPersonalizeServerDataV2
		if s.URI.IsNil() {
			log.Errorf("Unable to create new server deployment with nil server refrence.")
		}
		pd = OSDPersonalizeServerDataV2{
			ServerURI: s.URI.String(),
		}
		if bpdata != nil {
			pd.PersonalityData = bpdata
		}
		sd = append(sd, pd)
	}

	return DeploymentJobs{
		OsbpUris:   bpURI,
		ServerData: sd,
	}
}

// SubmitDeploymentJobs api call to deployment jobs
func (c *ICSPClient) SubmitDeploymentJobs(dj DeploymentJobs) (jt *JobTask, err error) {
	log.Info("Applying OS Build plan for ICSP")
	var (
		uri  = "/rest/os-deployment-jobs"
		juri ODSUri
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	jt = jt.NewJobTask(c)
	jt.Reset()
	data, err := c.RestAPICall(rest.POST, uri, dj)
	if err != nil {
		jt.IsDone = true
		log.Errorf("Error submitting new build request: %s", err)
		return jt, err
	}

	log.Debugf("Response submit new os build plan job %s", data)
	if err := json.Unmarshal([]byte(data), &juri); err != nil {
		jt.IsDone = true
		jt.JobURI = juri
		log.Errorf("Error with task un-marshal: %s", err)
		return jt, err
	}
	jt.JobURI = juri

	return jt, err
}

// ApplyDeployment plan to server
func (c *ICSPClient) ApplyDeploymentJobs(buildplans []string, bpdata *OSDPersonalityDataV2, s Server) (jt *JobTask, err error) {

	var dj DeploymentJobs
	var bplans []OSDBuildPlan
	var servers []Server

	// lookup buildplan by name
	for _, buildPlan := range buildplans {
		match, _ := regexp.MatchString("^(/rest/)(.+)*", buildPlan)
		if match {
			bp, err := c.GetBuildPlanByUri(utils.NewNstring(buildPlan))

			if err != nil {
				return jt, err
			}
			bplans = append(bplans, bp)
		} else {
			bp, err := c.GetBuildPlanByName(buildPlan)
			if err != nil {
				return jt, err
			}
			bplans = append(bplans, bp)
		}
	}

	servers = append(servers, s)
	dj = dj.NewDeploymentJobs(bplans, bpdata, servers)
	jt, err = c.SubmitDeploymentJobs(dj)
	err = jt.Wait()
	if err != nil {
		return jt, err
	}
	return jt, nil
}
