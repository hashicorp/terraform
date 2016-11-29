package logentries

type ApiResponse struct {
	Response       string `json:"response"`
	ResponseReason string `json:"reason"`
	Worker         string `json:"worker"`
	Id             string `json:"id"`
}

type ApiObject struct {
	Object string `json:"object"`
}

type LogType struct {
	Title       string `json:"title"`
	Description string `json:"desc"`
	Key         string `json:"key"`
	Shortcut    string `json:"shortcut"`
	ApiObject
}

type Log struct {
	Name      string `json:"name"`
	Created   int64  `json:"created"`
	Key       string `json:"key"`
	Token     string `json:"token"`
	Follow    string `json:"follow"`
	Retention int64  `json:"retention"`
	Source    string `json:"type"`
	Type      string `json:"logtype"`
	Filename  string `json:"filename"`
	ApiObject
}

type LogSet struct {
	Distver  string `json:"distver"`
	C        int64  `json:"c"`
	Name     string `json:"name"`
	Distname string `json:"distname"`
	Location string `json:"hostname"`
	Key      string `json:"key"`
	Logs     []Log
	ApiObject
}

type User struct {
	UserKey string        `json:"user_key"`
	LogSets []LogSet      `json:"hosts"`
	Apps    []interface{} `json:"apps"`
	Logs    []interface{} `json:"logs"`
}

type Client struct {
	Log     *LogClient
	LogSet  *LogSetClient
	User    *UserClient
	LogType *LogTypeClient
}

func NewClient(account_key string) *Client {
	client := &Client{}
	client.Log = NewLogClient(account_key)
	client.LogSet = NewLogSetClient(account_key)
	client.User = NewUserClient(account_key)
	client.LogType = NewLogTypeClient(account_key)
	return client
}
