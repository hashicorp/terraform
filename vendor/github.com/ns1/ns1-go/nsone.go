package nsone

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

// RateLimit stores X-Ratelimit-* headers
type RateLimit struct {
	Limit     int
	Remaining int
	Period    int
}

// PercentageLeft returns the ratio of Remaining to Limit as a percentage
func (rl RateLimit) PercentageLeft() int {
	return rl.Remaining * 100 / rl.Limit
}

// WaitTime returns the time.Duration ratio of Period to Limit
func (rl RateLimit) WaitTime() time.Duration {
	return (time.Second * time.Duration(rl.Period)) / time.Duration(rl.Limit)
}

// WaitTimeRemaining returns the time.Duration ratio of Period to Remaining
func (rl RateLimit) WaitTimeRemaining() time.Duration {
	return (time.Second * time.Duration(rl.Period)) / time.Duration(rl.Remaining)
}

// RateLimitStrategyNone sets RateLimitFunc to an empty func
func (c *APIClient) RateLimitStrategyNone() {
	c.RateLimitFunc = defaultRateLimitFunc
}

// RateLimitStrategySleep sets RateLimitFunc to sleep by WaitTimeRemaining
func (c *APIClient) RateLimitStrategySleep() {
	c.RateLimitFunc = func(rl RateLimit) {
		remaining := rl.WaitTimeRemaining()
		if c.debug {
			log.Printf("Rate limiting - Limit %d Remaining %d in period %d: Sleeping %dns", rl.Limit, rl.Remaining, rl.Period, remaining)
		}
		time.Sleep(remaining)
	}
}

// APIClient stores NS1 client state
type APIClient struct {
	ApiKey        string
	RateLimitFunc func(RateLimit)
	debug         bool
}

var defaultRateLimitFunc = func(rl RateLimit) {}

// New takes an API Key and creates an *APIClient
func New(k string) *APIClient {
	return &APIClient{
		ApiKey:        k,
		RateLimitFunc: defaultRateLimitFunc,
		debug:         false,
	}
}

// Debug enables debug logging
func (c *APIClient) Debug() {
	c.debug = true
}

func (c APIClient) doHTTP(method string, uri string, rbody []byte) ([]byte, int, error) {
	var body []byte
	r := bytes.NewReader(rbody)
	if c.debug {
		log.Printf("[DEBUG] %s: %s (%s)", method, uri, string(rbody))
	}
	req, err := http.NewRequest(method, uri, r)
	if err != nil {
		return body, 510, err
	}
	req.Header.Add("X-NSONE-Key", c.ApiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return body, 510, err
	}
	if c.debug {
		log.Println(resp)
	}
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if len(resp.Header["X-Ratelimit-Limit"]) > 0 {
		var remaining int
		var period int
		limit, err := strconv.Atoi(resp.Header["X-Ratelimit-Limit"][0])
		if err == nil {
			remaining, err = strconv.Atoi(resp.Header["X-Ratelimit-Remaining"][0])
			if err == nil {
				period, err = strconv.Atoi(resp.Header["X-Ratelimit-Period"][0])
			}
		}
		if err == nil {
			c.RateLimitFunc(RateLimit{
				Limit:     limit,
				Remaining: remaining,
				Period:    period,
			})
		}
	}
	if resp.StatusCode != 200 {
		return body, resp.StatusCode, fmt.Errorf("%s: %s", resp.Status, string(body))
	}
	if c.debug {
		log.Println(fmt.Sprintf("Response body: %s", string(body)))
	}
	return body, resp.StatusCode, nil

}

func (c APIClient) doHTTPUnmarshal(method string, uri string, rbody []byte, unpackInto interface{}) (int, error) {
	body, status, err := c.doHTTP(method, uri, rbody)
	if err != nil {
		return status, err
	}
	return status, json.Unmarshal(body, unpackInto)
}

func (c APIClient) doHTTPBoth(method string, uri string, s interface{}) error {
	rbody, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = c.doHTTPUnmarshal(method, uri, rbody, s)
	return err
}

func (c APIClient) doHTTPDelete(uri string) error {
	_, _, err := c.doHTTP("DELETE", uri, nil)
	return err
}

// GetZones returns all active zones and basic zone configuration details for each
func (c APIClient) GetZones() ([]Zone, error) {
	var zl []Zone
	_, err := c.doHTTPUnmarshal("GET", "https://api.nsone.net/v1/zones", nil, &zl)
	return zl, err
}

// GetZone takes a zone and returns a single active zone and its basic configuration details
func (c APIClient) GetZone(zone string) (*Zone, error) {
	z := NewZone(zone)
	_, err := c.doHTTPUnmarshal("GET", fmt.Sprintf("https://api.nsone.net/v1/zones/%s", z.Zone), nil, z)
	return z, err
}

// DeleteZone takes a zone and destroys an existing DNS zone and all records in the zone
func (c APIClient) DeleteZone(zone string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/zones/%s", zone))
}

// CreateZone takes a *Zone and creates a new DNS zone
func (c APIClient) CreateZone(z *Zone) error {
	return c.doHTTPBoth("PUT", fmt.Sprintf("https://api.nsone.net/v1/zones/%s", z.Zone), z)
}

// UpdateZone takes a *Zone and modifies basic details of a DNS zone
func (c APIClient) UpdateZone(z *Zone) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/zones/%s", z.Zone), z)
}

// CreateRecord takes a *Record and creates a new DNS record in the specified zone, for the specified domain, of the given record type
func (c APIClient) CreateRecord(r *Record) error {
	return c.doHTTPBoth("PUT", fmt.Sprintf("https://api.nsone.net/v1/zones/%s/%s/%s", r.Zone, r.Domain, r.Type), r)
}

// GetRecord takes a zone, domain and record type t and returns full configuration for a DNS record
func (c APIClient) GetRecord(zone string, domain string, t string) (*Record, error) {
	r := NewRecord(zone, domain, t)
	status, err := c.doHTTPUnmarshal("GET", fmt.Sprintf("https://api.nsone.net/v1/zones/%s/%s/%s", r.Zone, r.Domain, r.Type), nil, r)
	if status == 404 {
		r.Id = ""
		r.Zone = ""
		r.Domain = ""
		r.Type = ""
		return r, nil
	}
	return r, err
}

// DeleteRecord takes a zone, domain and record type t and removes an existing record and all associated answers and configuration details
func (c APIClient) DeleteRecord(zone string, domain string, t string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/zones/%s/%s/%s", zone, domain, t))
}

// UpdateRecord takes a *Record and modifies configuration details for an existing DNS record
func (c APIClient) UpdateRecord(r *Record) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/zones/%s/%s/%s", r.Zone, r.Domain, r.Type), r)
}

// CreateDataSource takes a *DataSource and creates a new data source
func (c APIClient) CreateDataSource(ds *DataSource) error {
	return c.doHTTPBoth("PUT", "https://api.nsone.net/v1/data/sources", ds)
}

// GetDataSource takes an ID returns the details for a single data source
func (c APIClient) GetDataSource(id string) (*DataSource, error) {
	ds := DataSource{}
	_, err := c.doHTTPUnmarshal("GET", fmt.Sprintf("https://api.nsone.net/v1/data/sources/%s", id), nil, &ds)
	return &ds, err
}

// DeleteDataSource takes an ID and removes an existing data source and all connected feeds from the cource
func (c APIClient) DeleteDataSource(id string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/data/sources/%s", id))
}

// UpdateDataSource takes a *DataSource modifies basic details of a data source
func (c APIClient) UpdateDataSource(ds *DataSource) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/data/sources/%s", ds.Id), ds)
}

// CreateDataFeed takes a *DataFeed and connects a new data feed to an existing data source
func (c APIClient) CreateDataFeed(df *DataFeed) error {
	return c.doHTTPBoth("PUT", fmt.Sprintf("https://api.nsone.net/v1/data/feeds/%s", df.SourceId), df)
}

// GetDataFeed takes a data source ID and a data feed ID and returns the details of a single data feed
func (c APIClient) GetDataFeed(dsID string, dfID string) (*DataFeed, error) {
	df := NewDataFeed(dsID)
	status, err := c.doHTTPUnmarshal("GET", fmt.Sprintf("https://api.nsone.net/v1/data/feeds/%s/%s", dsID, dfID), nil, df)
	if status == 404 {
		df.SourceId = ""
		df.Id = ""
		df.Name = ""
		return df, nil
	}
	return df, err
}

// DeleteDataFeed takes a data source ID and a data feed ID and disconnects the feed from the data source and all attached destination metadata tables
func (c APIClient) DeleteDataFeed(dsID string, dfID string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/data/feeds/%s/%s", dsID, dfID))
}

// UpdateDataFeed takes a *DataFeed and modifies and existing data feed
func (c APIClient) UpdateDataFeed(df *DataFeed) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/data/feeds/%s/%s", df.SourceId, df.Id), df)
}

// GetMonitoringJobTypes returns the list of all available monitoring job types
func (c APIClient) GetMonitoringJobTypes() (MonitoringJobTypes, error) {
	var mjt MonitoringJobTypes
	_, err := c.doHTTPUnmarshal("GET", "https://api.nsone.net/v1/monitoring/jobtypes", nil, &mjt)
	return mjt, err
}

// GetMonitoringJobs returns the list of all monitoring jobs for the account
func (c APIClient) GetMonitoringJobs() (MonitoringJobs, error) {
	var mj MonitoringJobs
	_, err := c.doHTTPUnmarshal("GET", "https://api.nsone.net/v1/monitoring/jobs", nil, &mj)
	return mj, err
}

// GetMonitoringJob takes an ID and returns details for a specific monitoring job
func (c APIClient) GetMonitoringJob(id string) (MonitoringJob, error) {
	var mj MonitoringJob
	_, err := c.doHTTPUnmarshal("GET", fmt.Sprintf("https://api.nsone.net/v1/monitoring/jobs/%s", id), nil, &mj)
	return mj, err
}

// CreateMonitoringJob takes a *MonitoringJob and creates a new monitoring job
func (c APIClient) CreateMonitoringJob(mj *MonitoringJob) error {
	return c.doHTTPBoth("PUT", "https://api.nsone.net/v1/monitoring/jobs", mj)
}

// DeleteMonitoringJob takes an ID and immediately terminates and deletes and existing monitoring job
func (c APIClient) DeleteMonitoringJob(id string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/monitoring/jobs/%s", id))
}

// UpdateMonitoringJob takes a *MonitoringJob and change the configuration details of an existing monitoring job
func (c APIClient) UpdateMonitoringJob(mj *MonitoringJob) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/monitoring/jobs/%s", mj.Id), mj)
}

// GetQPSStats returns current queries per second (QPS) for the account
func (c APIClient) GetQPSStats() (v float64, err error) {
	var s map[string]float64
	_, err = c.doHTTPUnmarshal("GET", "https://api.nsone.net/v1/stats/qps", nil, &s)
	if err != nil {
		return v, err
	}
	v, found := s["qps"]
	if !found {
		return v, errors.New("Could not find 'qps' key in returned data")
	}
	return v, nil
}

// GetUsers returns a list of all users with access to the account
func (c APIClient) GetUsers() ([]User, error) {
	var users []User
	_, err := c.doHTTPUnmarshal("GET", "https://api.nsone.net/v1/account/users", nil, &users)
	return users, err
}

// GetUser takes a username and returns the details for a single user
func (c APIClient) GetUser(username string) (User, error) {
	var u User
	status, err := c.doHTTPUnmarshal("GET", fmt.Sprintf("https://api.nsone.net/v1/account/users/%s", username), nil, &u)
	if status == 404 {
		u.Username = ""
		u.Name = ""
		return u, nil
	}
	return u, err
}

// CreateUser takes a *User and creates a new user
func (c APIClient) CreateUser(u *User) error {
	return c.doHTTPBoth("PUT", fmt.Sprintf("https://api.nsone.net/v1/account/users/%s", u.Username), &u)
}

// DeleteUser takes a username and deletes a user from the account
func (c APIClient) DeleteUser(username string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/account/users/%s", username))
}

// UpdateUser takes a *User and change contact details, notification settings or access rights for a user
func (c APIClient) UpdateUser(user *User) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/account/users/%s", user.Username), user)
}

// GetApikeys returns a list of all API keys under the account
func (c APIClient) GetApikeys() ([]Apikey, error) {
	var apikeys []Apikey
	_, err := c.doHTTPUnmarshal("GET", "https://api.nsone.net/v1/account/apikeys", nil, &apikeys)
	return apikeys, err
}

// GetApikey takes an ID and returns details, including permissions, for a single API key
func (c APIClient) GetApikey(id string) (Apikey, error) {
	var k Apikey
	status, err := c.doHTTPUnmarshal("GET", fmt.Sprintf("https://api.nsone.net/v1/account/apikeys/%s", id), nil, &k)
	if status == 404 {
		k.Id = ""
		k.Key = ""
		k.Name = ""
		return k, nil
	}
	return k, err
}

// CreateApikey takes an *Apikey and creates a new API key
func (c APIClient) CreateApikey(k *Apikey) error {
	return c.doHTTPBoth("PUT", fmt.Sprintf("https://api.nsone.net/v1/account/apikeys/%s", k.Id), &k)
}

// DeleteApikey takes an ID and deletes and API key
func (c APIClient) DeleteApikey(id string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/account/apikeys/%s", id))
}

// UpdateApikey takes an *Apikey and change name or access rights for an API key
func (c APIClient) UpdateApikey(k *Apikey) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/account/apikeys/%s", k.Id), k)
}

// GetTeams returns a list of all teams under the account
func (c APIClient) GetTeams() ([]Team, error) {
	var teams []Team
	_, err := c.doHTTPUnmarshal("GET", "https://api.nsone.net/v1/account/teams", nil, &teams)
	return teams, err
}

// GetTeam takes an ID and returns details, including permissions, for a single team
func (c APIClient) GetTeam(id string) (Team, error) {
	var t Team
	status, err := c.doHTTPUnmarshal("GET", fmt.Sprintf("https://api.nsone.net/v1/account/teams/%s", id), nil, &t)
	if status == 404 {
		t.Id = ""
		t.Name = ""
		return t, nil
	}
	return t, err
}

// CreateTeam takes a *Team and creates a new team
func (c APIClient) CreateTeam(t *Team) error {
	return c.doHTTPBoth("PUT", "https://api.nsone.net/v1/account/teams", &t)
}

// DeleteTeam takes an ID and deletes a team. Any users of API keys that belong to the team will be removed from the team.
func (c APIClient) DeleteTeam(id string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/account/teams/%s", id))
}

// UpdateTeam takes a *Team and change name or access rights for a team
func (c APIClient) UpdateTeam(t *Team) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/account/teams/%s", t.Id), t)
}
