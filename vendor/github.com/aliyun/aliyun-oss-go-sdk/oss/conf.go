package oss

import (
	"time"
)

// HTTPTimeout defines HTTP timeout.
type HTTPTimeout struct {
	ConnectTimeout   time.Duration
	ReadWriteTimeout time.Duration
	HeaderTimeout    time.Duration
	LongTimeout      time.Duration
	IdleConnTimeout  time.Duration
}

type HTTPMaxConns struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
}

// Config defines oss configuration
type Config struct {
	Endpoint        string       // OSS endpoint
	AccessKeyID     string       // AccessId
	AccessKeySecret string       // AccessKey
	RetryTimes      uint         // Retry count by default it's 5.
	UserAgent       string       // SDK name/version/system information
	IsDebug         bool         // Enable debug mode. Default is false.
	Timeout         uint         // Timeout in seconds. By default it's 60.
	SecurityToken   string       // STS Token
	IsCname         bool         // If cname is in the endpoint.
	HTTPTimeout     HTTPTimeout  // HTTP timeout
	HTTPMaxConns    HTTPMaxConns // Http max connections
	IsUseProxy      bool         // Flag of using proxy.
	ProxyHost       string       // Flag of using proxy host.
	IsAuthProxy     bool         // Flag of needing authentication.
	ProxyUser       string       // Proxy user
	ProxyPassword   string       // Proxy password
	IsEnableMD5     bool         // Flag of enabling MD5 for upload.
	MD5Threshold    int64        // Memory footprint threshold for each MD5 computation (16MB is the default), in byte. When the data is more than that, temp file is used.
	IsEnableCRC     bool         // Flag of enabling CRC for upload.
}

// getDefaultOssConfig gets the default configuration.
func getDefaultOssConfig() *Config {
	config := Config{}

	config.Endpoint = ""
	config.AccessKeyID = ""
	config.AccessKeySecret = ""
	config.RetryTimes = 5
	config.IsDebug = false
	config.UserAgent = userAgent()
	config.Timeout = 60 // Seconds
	config.SecurityToken = ""
	config.IsCname = false

	config.HTTPTimeout.ConnectTimeout = time.Second * 30   // 30s
	config.HTTPTimeout.ReadWriteTimeout = time.Second * 60 // 60s
	config.HTTPTimeout.HeaderTimeout = time.Second * 60    // 60s
	config.HTTPTimeout.LongTimeout = time.Second * 300     // 300s
	config.HTTPTimeout.IdleConnTimeout = time.Second * 50  // 50s
	config.HTTPMaxConns.MaxIdleConns = 100
	config.HTTPMaxConns.MaxIdleConnsPerHost = 100

	config.IsUseProxy = false
	config.ProxyHost = ""
	config.IsAuthProxy = false
	config.ProxyUser = ""
	config.ProxyPassword = ""

	config.MD5Threshold = 16 * 1024 * 1024 // 16MB
	config.IsEnableMD5 = false
	config.IsEnableCRC = true

	return &config
}
