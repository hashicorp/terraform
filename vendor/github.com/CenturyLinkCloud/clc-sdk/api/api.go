package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

var debug = os.Getenv("DEBUG") != ""

const baseUriDefault = "https://api.ctl.io/v2"
const userAgentDefault = "CenturyLinkCloud/clc-sdk"

func New(config Config) *Client {
	return &Client{
		config: config,
		client: http.DefaultClient,
	}
}

type HTTP interface {
	Get(url string, resp interface{}) error
	Post(url string, body, resp interface{}) error
	Put(url string, body, resp interface{}) error
	Patch(url string, body, resp interface{}) error
	Delete(url string, resp interface{}) error
	Config() *Config
}

type Client struct {
	config Config
	Token  Token

	client *http.Client
}

func (c *Client) Config() *Config {
	return &c.config
}

func (c *Client) Get(url string, resp interface{}) error {
	return c.DoWithAuth("GET", url, nil, resp)
}

func (c *Client) Post(url string, body, resp interface{}) error {
	return c.DoWithAuth("POST", url, body, resp)
}

func (c *Client) Put(url string, body, resp interface{}) error {
	return c.DoWithAuth("PUT", url, body, resp)
}

func (c *Client) Patch(url string, body, resp interface{}) error {
	return c.DoWithAuth("PATCH", url, body, resp)
}

func (c *Client) Delete(url string, resp interface{}) error {
	return c.DoWithAuth("DELETE", url, nil, resp)
}

func (c *Client) Auth() error {
	url := fmt.Sprintf("%s/authentication/login", c.config.BaseURL)
	body, err := c.serialize(c.config.User)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}

	err = c.Do(req, &c.Token)
	if err == nil && c.config.Alias == "" {
		// set Alias from returned token
		c.config.Alias = c.Token.Alias
	}
	return err
}

func (c *Client) Do(req *http.Request, ret interface{}) error {
	if debug {
		v, _ := httputil.DumpRequest(req, true)
		log.Println(string(v))
	}

	req.Header.Add("User-Agent", c.config.UserAgent)
	req.Header.Add("Api-Client", c.config.UserAgent)
	req.Header.Add("Accept", "application/json")
	if req.Body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		log.Println(err)
		return err
	}
	if debug && resp != nil {
		v, _ := httputil.DumpResponse(resp, true)
		log.Println(string(v))
	}
	if resp.StatusCode >= 400 {
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("http err: [%s] - %s", resp.Status, b)
	}

	if ret == nil {
		return nil
	}

	// FIXME? empty body: check status=204 or content-length=0 before parsing
	return json.NewDecoder(resp.Body).Decode(ret)
}

func (c *Client) DoWithAuth(method, url string, body, ret interface{}) error {
	if !c.Token.Valid() {
		err := c.Auth()
		if err != nil {
			return err
		}
	}

	b, err := c.serialize(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, url, b)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+c.Token.Token)

	return c.Do(req, ret)
}

func (c *Client) serialize(body interface{}) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}

	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(body)
	return b, err
}

type Config struct {
	User      User     `json:"user"`
	Alias     string   `json:"alias"`
	BaseURL   *url.URL `json:"-"`
	UserAgent string   `json:"agent,omitempty"`
}

func (c Config) Valid() bool {
	return c.User.Username != "" && c.User.Password != "" && c.BaseURL != nil
}

func EnvConfig() (Config, error) {
	user := os.Getenv("CLC_USERNAME")
	pass := os.Getenv("CLC_PASSWORD")
	config, err := NewConfig(user, pass)
	if err != nil {
		return config, err
	}

	if !config.Valid() {
		return config, fmt.Errorf("missing environment variables [%s]", config)
	}
	return config, nil
}

// NewConfig takes credentials and returns a Config object that may be further customized.
// Defaults for Alias, BaseURL, and UserAgent will be taken from respective env vars.
func NewConfig(username, password string) (Config, error) {
	alias := os.Getenv("CLC_ALIAS")
	agent := userAgentDefault
	if v := os.Getenv("CLC_USER_AGENT"); v != "" {
		agent = v
	}
	base := baseUriDefault
	if v := os.Getenv("CLC_BASE_URL"); v != "" {
		base = v
	}
	uri, err := url.Parse(base)
	return Config{
		User: User{
			Username: username,
			Password: password,
		},
		Alias:     alias,
		BaseURL:   uri,
		UserAgent: agent,
	}, err
}

func FileConfig(file string) (Config, error) {
	config := Config{}
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(b, &config)

	u, err := url.Parse(baseUriDefault)
	config.BaseURL = u
	return config, err
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Token struct {
	Username string   `json:"userName"`
	Alias    string   `json:"accountAlias"`
	Location string   `json:"locationAlias"`
	Roles    []string `json:"roles"`
	Token    string   `json:"bearerToken"`
}

// TODO: Add some real validation logic
func (t Token) Valid() bool {
	return t.Token != ""
}

type Update struct {
	Op     string      `json:"op"`
	Member string      `json:"member"`
	Value  interface{} `json:"value"`
}
