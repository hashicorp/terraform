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

type RateLimit struct {
	Limit     int
	Remaining int
	Period    int
}

func (rl RateLimit) PercentageLeft() int {
	return rl.Remaining * 100 / rl.Limit
}

func (rl RateLimit) WaitTime() time.Duration {
	return (time.Second * time.Duration(rl.Period)) / time.Duration(rl.Limit)
}

func (rl RateLimit) WaitTimeRemaining() time.Duration {
	return (time.Second * time.Duration(rl.Period)) / time.Duration(rl.Remaining)
}

func (a *APIClient) RateLimitStrategyNone() {
	a.RateLimitFunc = defaultRateLimitFunc
}

func (a *APIClient) RateLimitStrategySleep() {
	a.RateLimitFunc = func(rl RateLimit) {
		remaining := rl.WaitTimeRemaining()
		if a.debug {
			log.Println("Rate limiting - Limit %d Remaining %d in period %d: Sleeping %dns", rl.Limit, rl.Remaining, rl.Period, remaining)
		}
		time.Sleep(remaining)
	}
}

type APIClient struct {
	ApiKey        string
	RateLimitFunc func(RateLimit)
	debug         bool
}

var defaultRateLimitFunc = func(rl RateLimit) {}

func New(k string) *APIClient {
	return &APIClient{
		ApiKey:        k,
		RateLimitFunc: defaultRateLimitFunc,
		debug:         false,
	}
}

func (c *APIClient) Debug() {
	c.debug = true
}

func (c APIClient) doHTTP(method string, uri string, rbody []byte) ([]byte, int, error) {
	var body []byte
	r := bytes.NewReader(rbody)
	log.Printf("[DEBUG] %s: %s (%s)", method, uri, string(rbody))
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
	log.Println(resp)
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
		return body, resp.StatusCode, errors.New(fmt.Sprintf("%s: %s", resp.Status, string(body)))
	}
	log.Println(fmt.Sprintf("Response body: %s", string(body)))
	return body, resp.StatusCode, nil

}

func (c APIClient) doHTTPUnmarshal(method string, uri string, rbody []byte, unpack_into interface{}) (int, error) {
	body, status, err := c.doHTTP(method, uri, rbody)
	if err != nil {
		return status, err
	}
	return status, json.Unmarshal(body, unpack_into)
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

func (c APIClient) GetZones() ([]Zone, error) {
	var zl []Zone
	_, err := c.doHTTPUnmarshal("GET", "https://api.nsone.net/v1/zones", nil, &zl)
	return zl, err
}

func (c APIClient) GetZone(zone string) (*Zone, error) {
	z := NewZone(zone)
	status, err := c.doHTTPUnmarshal("GET", fmt.Sprintf("https://api.nsone.net/v1/zones/%s", z.Zone), nil, z)
	if status == 404 {
		z.Id = ""
		z.Zone = ""
		return z, nil
	}
	return z, err
}

func (c APIClient) DeleteZone(zone string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/zones/%s", zone))
}

func (c APIClient) CreateZone(z *Zone) error {
	return c.doHTTPBoth("PUT", fmt.Sprintf("https://api.nsone.net/v1/zones/%s", z.Zone), z)
}

func (c APIClient) UpdateZone(z *Zone) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/zones/%s", z.Zone), z)
}

func (c APIClient) CreateRecord(r *Record) error {
	return c.doHTTPBoth("PUT", fmt.Sprintf("https://api.nsone.net/v1/zones/%s/%s/%s", r.Zone, r.Domain, r.Type), r)
}

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

func (c APIClient) DeleteRecord(zone string, domain string, t string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/zones/%s/%s/%s", zone, domain, t))
}

func (c APIClient) UpdateRecord(r *Record) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/zones/%s/%s/%s", r.Zone, r.Domain, r.Type), r)
}

func (c APIClient) CreateDataSource(ds *DataSource) error {
	return c.doHTTPBoth("PUT", "https://api.nsone.net/v1/data/sources", ds)
}

func (c APIClient) GetDataSource(id string) (*DataSource, error) {
	ds := DataSource{}
	_, err := c.doHTTPUnmarshal("GET", fmt.Sprintf("https://api.nsone.net/v1/data/sources/%s", id), nil, &ds)
	return &ds, err
}

func (c APIClient) DeleteDataSource(id string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/data/sources/%s", id))
}

func (c APIClient) UpdateDataSource(ds *DataSource) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/data/sources/%s", ds.Id), ds)
}
func (c APIClient) CreateDataFeed(df *DataFeed) error {
	return c.doHTTPBoth("PUT", fmt.Sprintf("https://api.nsone.net/v1/data/feeds/%s", df.SourceId), df)
}

func (c APIClient) GetDataFeed(ds_id string, df_id string) (*DataFeed, error) {
	df := NewDataFeed(ds_id)
	status, err := c.doHTTPUnmarshal("GET", fmt.Sprintf("https://api.nsone.net/v1/data/feeds/%s/%s", ds_id, df_id), nil, df)
	if status == 404 {
		df.SourceId = ""
		df.Id = ""
		df.Name = ""
		return df, nil
	}
	return df, err
}

func (c APIClient) DeleteDataFeed(ds_id string, df_id string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/data/feeds/%s/%s", ds_id, df_id))
}

func (c APIClient) UpdateDataFeed(df *DataFeed) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/data/feeds/%s/%s", df.SourceId, df.Id), df)
}

func (c APIClient) GetMonitoringJobTypes() (MonitoringJobTypes, error) {
	var mjt MonitoringJobTypes
	_, err := c.doHTTPUnmarshal("GET", "https://api.nsone.net/v1/monitoring/jobtypes", nil, &mjt)
	return mjt, err
}

func (c APIClient) GetMonitoringJobs() (MonitoringJobs, error) {
	var mj MonitoringJobs
	_, err := c.doHTTPUnmarshal("GET", "https://api.nsone.net/v1/monitoring/jobs", nil, &mj)
	return mj, err
}

func (c APIClient) GetMonitoringJob(id string) (MonitoringJob, error) {
	var mj MonitoringJob
	status, err := c.doHTTPUnmarshal("GET", fmt.Sprintf("https://api.nsone.net/v1/monitoring/jobs/%s", id), nil, &mj)
	if status == 404 {
		mj.Id = ""
		mj.Name = ""
		return mj, nil
	}
	return mj, err
}

func (c APIClient) CreateMonitoringJob(mj *MonitoringJob) error {
	return c.doHTTPBoth("PUT", "https://api.nsone.net/v1/monitoring/jobs", mj)
}

func (c APIClient) DeleteMonitoringJob(id string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/monitoring/jobs/%s", id))
}

func (c APIClient) UpdateMonitoringJob(mj *MonitoringJob) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/monitoring/jobs/%s", mj.Id), mj)
}

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

func (c APIClient) GetUsers() ([]User, error) {
	var users []User
	_, err := c.doHTTPUnmarshal("GET", "https://api.nsone.net/v1/account/users", nil, &users)
	return users, err
}

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

func (c APIClient) CreateUser(u *User) error {
	return c.doHTTPBoth("PUT", fmt.Sprintf("https://api.nsone.net/v1/account/users/%s", u.Username), &u)
}

func (c APIClient) DeleteUser(username string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/account/users/%s", username))
}

func (c APIClient) UpdateUser(user *User) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/account/users/%s", user.Username), user)
}

func (c APIClient) GetApikeys() ([]Apikey, error) {
	var apikeys []Apikey
	_, err := c.doHTTPUnmarshal("GET", "https://api.nsone.net/v1/account/apikeys", nil, &apikeys)
	return apikeys, err
}

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

func (c APIClient) CreateApikey(k *Apikey) error {
	return c.doHTTPBoth("PUT", fmt.Sprintf("https://api.nsone.net/v1/account/apikeys/%s", k.Id), &k)
}

func (c APIClient) DeleteApikey(id string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/account/apikeys/%s", id))
}

func (c APIClient) UpdateApikey(k *Apikey) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/account/apikeys/%s", k.Id), k)
}

func (c APIClient) GetTeams() ([]Team, error) {
	var teams []Team
	_, err := c.doHTTPUnmarshal("GET", "https://api.nsone.net/v1/account/teams", nil, &teams)
	return teams, err
}

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

func (c APIClient) CreateTeam(t *Team) error {
	return c.doHTTPBoth("PUT", "https://api.nsone.net/v1/account/teams", &t)
}

func (c APIClient) DeleteTeam(id string) error {
	return c.doHTTPDelete(fmt.Sprintf("https://api.nsone.net/v1/account/teams/%s", id))
}

func (c APIClient) UpdateTeam(t *Team) error {
	return c.doHTTPBoth("POST", fmt.Sprintf("https://api.nsone.net/v1/account/teams/%s", t.Id), t)
}
