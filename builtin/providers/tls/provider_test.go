package tls

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

var testProviders = map[string]terraform.ResourceProvider{
	"tls": Provider(),
}

var testPrivateKey = `
-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQDPLaq43D9C596ko9yQipWUf2FbRhFs18D3wBDBqXLIoP7W3rm5
S292/JiNPa+mX76IYFF416zTBGG9J5w4d4VFrROn8IuMWqHgdXsCUf2szN7EnJcV
BsBzTxxWqz4DjX315vbm/PFOLlKzC0Ngs4h1iDiCD9Hk2MajZuFnJiqj1QIDAQAB
AoGAG6eQ3lQn7Zpd0cQ9sN2O0d+e8zwLH2g9TdTJZ9Bijf1Phwb764vyOQPGqTPO
unqVSEbzGRpQ62nuUf1zkOYDV+gKMNO3mj9Zu+qPNr/nQPHIaGZksPdD34qDUnBl
eRWVGNTyEGQsRPNN0RtFj8ifa4+OWiE30n95PBq2bUGZj4ECQQDZvS5X/4jYxnzw
CscaL4vO9OCVd/Fzdpfak0DQE/KCVmZxzcXu6Q8WuhybCynX84WKHQxuFAo+nBvr
kgtWXX7dAkEA85Vs5ehuDujBKCu3NJYI2R5ie49L9fEMFJVZK9FpkKacoAkET5BZ
UzaZrx4Fg3Zhcv1TssZKSyle+2lYiIydWQJBAMW8/aJi6WdcUsg4MXrBZSlsz6xO
AhOGxv90LS8KfnkJd/2wDyoZs19DY4kWSUjZ2hOEr+4j+u3DHcQAnJUxUW0CQGXP
DrUJcPbKUfF4VBqmmwwkpwT938Hr/iCcS6kE3hqXiN9a5XJb4vnk2FdZNPS9hf2J
5HHUbzj7EbgDT/3CyAECQG0qv6LNQaQMm2lmQKmqpi43Bqj9wvx0xGai1qCOvSeL
rpxCHbX0xSJh0s8j7exRHMF8W16DHjjkc265YdWPXWo=
-----END RSA PRIVATE KEY-----
`
