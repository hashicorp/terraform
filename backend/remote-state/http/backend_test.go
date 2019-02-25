package http

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs"
	"github.com/zclconf/go-cty/cty"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestHTTPClientFactory(t *testing.T) {
	// defaults

	conf := map[string]cty.Value{
		"address": cty.StringVal("http://127.0.0.1:8888/foo"),
	}
	b := backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client := b.client

	if client == nil {
		t.Fatal("Unexpected failure, address")
	}
	if client.URL.String() != "http://127.0.0.1:8888/foo" {
		t.Fatalf("Expected address \"%s\", got \"%s\"", conf["address"], client.URL.String())
	}
	if client.UpdateMethod != "POST" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "POST", client.UpdateMethod)
	}
	if client.LockURL != nil || client.LockMethod != "LOCK" {
		t.Fatal("Unexpected lock_address or lock_method")
	}
	if client.UnlockURL != nil || client.UnlockMethod != "UNLOCK" {
		t.Fatal("Unexpected unlock_address or unlock_method")
	}
	if client.Username != "" || client.Password != "" {
		t.Fatal("Unexpected username or password")
	}

	// custom
	conf = map[string]cty.Value{
		"address":        cty.StringVal("http://127.0.0.1:8888/foo"),
		"update_method":  cty.StringVal("BLAH"),
		"lock_address":   cty.StringVal("http://127.0.0.1:8888/bar"),
		"lock_method":    cty.StringVal("BLIP"),
		"unlock_address": cty.StringVal("http://127.0.0.1:8888/baz"),
		"unlock_method":  cty.StringVal("BLOOP"),
		"username":       cty.StringVal("user"),
		"password":       cty.StringVal("pass"),
	}

	b = backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client = b.client

	if client == nil {
		t.Fatal("Unexpected failure, update_method")
	}
	if client.UpdateMethod != "BLAH" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "BLAH", client.UpdateMethod)
	}
	if client.LockURL.String() != conf["lock_address"].AsString() || client.LockMethod != "BLIP" {
		t.Fatalf("Unexpected lock_address \"%s\" vs \"%s\" or lock_method \"%s\" vs \"%s\"", client.LockURL.String(),
			conf["lock_address"].AsString(), client.LockMethod, conf["lock_method"])
	}
	if client.UnlockURL.String() != conf["unlock_address"].AsString() || client.UnlockMethod != "BLOOP" {
		t.Fatalf("Unexpected unlock_address \"%s\" vs \"%s\" or unlock_method \"%s\" vs \"%s\"", client.UnlockURL.String(),
			conf["unlock_address"].AsString(), client.UnlockMethod, conf["unlock_method"])
	}
	if client.Username != "user" || client.Password != "pass" {
		t.Fatalf("Unexpected username \"%s\" vs \"%s\" or password \"%s\" vs \"%s\"", client.Username, conf["username"],
			client.Password, conf["password"])
	}
}
func TestHTTPClientFactoryWithTLSAndECDSAEncryptedKey(t *testing.T) {
	// defaults

	conf := map[string]cty.Value{
		"address": cty.StringVal("https://127.0.0.1:8888/foo"),
		"tls_client_key": cty.StringVal(`-----BEGIN ENCRYPTED PRIVATE KEY-----
MIHsMFcGCSqGSIb3DQEFDTBKMCkGCSqGSIb3DQEFDDAcBAjVvKZtHlmIbAICCAAw
DAYIKoZIhvcNAgkFADAdBglghkgBZQMEASoEEL3jdkBvObn+QELgKVE2cnMEgZAl
wgo3AjtXevJaGgep5GsW2krw9S7dC7xG9dR33Z/a9nBnO1rKm7Htf0+986w/1vmj
4k3M2QiI/VY+tnDFE+46DLLKYtJGRT1aoAH+mwhzaQGwzJnKhbeA23aE0f7KWCAK
+f999+SeHWro7FiRZjHEYVVLGQr/I7K5Wyh24YjN2nR4CU4X+GQU25My/pgSRog=
-----END ENCRYPTED PRIVATE KEY-----`),
		"tls_client_cert": cty.StringVal(`-----BEGIN CERTIFICATE-----
MIIB9jCCAZugAwIBAgIJAOi4ebDp8F1IMAoGCCqGSM49BAMCMFYxCzAJBgNVBAYT
AlVTMQ8wDQYDVQQIDAZEZW5pYWwxFDASBgNVBAcMC1NwcmluZ2ZpZWxkMQwwCgYD
VQQKDANEaXMxEjAQBgNVBAMMCVVTRVJfMTIzNDAgFw0xOTAyMjQwOTMxMzFaGA8y
MTE5MDEzMTA5MzEzMVowVjELMAkGA1UEBhMCVVMxDzANBgNVBAgMBkRlbmlhbDEU
MBIGA1UEBwwLU3ByaW5nZmllbGQxDDAKBgNVBAoMA0RpczESMBAGA1UEAwwJVVNF
Ul8xMjM0MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEipKHaB/oR5gX8KBX6779
RPojC7d4NbOocTlDG5L+mU1RVzQ/98/c5SZYuv2Bq5Up7BziXNBm8EmA9QDMGcq9
E6NQME4wHQYDVR0OBBYEFHdSsLG3NIFhGK9ciGJQaaZQWxFmMB8GA1UdIwQYMBaA
FHdSsLG3NIFhGK9ciGJQaaZQWxFmMAwGA1UdEwQFMAMBAf8wCgYIKoZIzj0EAwID
SQAwRgIhAP1s5Cbm5IyfDsB0HHpfXeDH3EPgs1VxVb5JziJg5z3bAiEA5QXuZNHQ
Iq6lIbHyofyS9nhepycaoT/TDT5BVtrSbGs=
-----END CERTIFICATE-----`),
		"tls_client_key_password": cty.StringVal("password"),
	}
	b := backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client := b.client

	if client == nil {
		t.Fatal("Unexpected failure, address")
	}
	if client.URL.String() != "https://127.0.0.1:8888/foo" {
		t.Fatalf("Expected address \"%s\", got \"%s\"", conf["address"], client.URL.String())
	}
	if client.UpdateMethod != "POST" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "POST", client.UpdateMethod)
	}
	if client.LockURL != nil || client.LockMethod != "LOCK" {
		t.Fatal("Unexpected lock_address or lock_method")
	}
	if client.UnlockURL != nil || client.UnlockMethod != "UNLOCK" {
		t.Fatal("Unexpected unlock_address or unlock_method")
	}
	if client.Username != "" || client.Password != "" {
		t.Fatal("Unexpected username or password")
	}

	// custom
	conf = map[string]cty.Value{
		"address":        cty.StringVal("http://127.0.0.1:8888/foo"),
		"update_method":  cty.StringVal("BLAH"),
		"lock_address":   cty.StringVal("http://127.0.0.1:8888/bar"),
		"lock_method":    cty.StringVal("BLIP"),
		"unlock_address": cty.StringVal("http://127.0.0.1:8888/baz"),
		"unlock_method":  cty.StringVal("BLOOP"),
		"username":       cty.StringVal("user"),
		"password":       cty.StringVal("pass"),
	}

	b = backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client = b.client

	if client == nil {
		t.Fatal("Unexpected failure, update_method")
	}
	if client.UpdateMethod != "BLAH" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "BLAH", client.UpdateMethod)
	}
	if client.LockURL.String() != conf["lock_address"].AsString() || client.LockMethod != "BLIP" {
		t.Fatalf("Unexpected lock_address \"%s\" vs \"%s\" or lock_method \"%s\" vs \"%s\"", client.LockURL.String(),
			conf["lock_address"].AsString(), client.LockMethod, conf["lock_method"])
	}
	if client.UnlockURL.String() != conf["unlock_address"].AsString() || client.UnlockMethod != "BLOOP" {
		t.Fatalf("Unexpected unlock_address \"%s\" vs \"%s\" or unlock_method \"%s\" vs \"%s\"", client.UnlockURL.String(),
			conf["unlock_address"].AsString(), client.UnlockMethod, conf["unlock_method"])
	}
	if client.Username != "user" || client.Password != "pass" {
		t.Fatalf("Unexpected username \"%s\" vs \"%s\" or password \"%s\" vs \"%s\"", client.Username, conf["username"],
			client.Password, conf["password"])
	}
}
func TestHTTPClientFactoryWithTLSAndRSAEncryptedKey(t *testing.T) {
	// defaults

	conf := map[string]cty.Value{
		"address": cty.StringVal("https://127.0.0.1:8888/foo"),
		"tls_client_key": cty.StringVal(`-----BEGIN ENCRYPTED PRIVATE KEY-----
MIIFHzBJBgkqhkiG9w0BBQ0wPDAbBgkqhkiG9w0BBQwwDgQILCEpWEgs7EICAggA
MB0GCWCGSAFlAwQBKgQQuj5iy4VVTkPLkun5Gi7jkASCBNCO7grfZh+kaCcNe91b
r5W8XPp+SkrJ4y4fo/Z8221RQAyOKUSEIskvgPiZq6Jp0dFM4Y4pwlSQB3rr6Rd8
A/SNRdDcFqlwyq9cvhFvqCBiyGqSdH8F5t34zj2LBPtd8GqUUy1d/Ig7PHzUYmZh
C6ABZxwdIKoZciRuzWyDTv1xWoGyciUx8lmxSFmD3k2LojqfjeB50GaIKsRp1jGs
8imPkAZ7bM12FR0mIIg+Y+QU3kzcWNGrc2KCHYrU4T4Mxu7Y+D++pCJt7reHVqiz
2EgIeNjb1J8UPxgspjPrOzO+sLNacZ7G3UOjpU3UD5Mk6x/99AdtXYs39vnqEkwS
YxRAtXesusTX5jc3HVOwnLcjKRZOQPI57VnZs7t6SwmN3wjLQB0eieTb8nxBf/sn
sllVPc/t8lTon7S5RFUfsFwvCvbj4rGtoBJdUQVMFwg4CbMw90CwNW/rLB1m1bpd
GIkDb0gHMsS4fhWpR5SDWUkw1jCitErudWuhH41D8Bai7fHlDg2AnrhK6hHFgsNN
u52NvFmI96ZVpDIylb3i2WUrt2ZbTbw2f23KfxIbTCbi4TkqaRRx6bq+ekOUbZxD
7vtrOHmJeOk6wQ1Mco3bcmb28NHYaocdUHcWeUfgKdATs/MCMO89kMwR7YBshnOP
ydAhzbh6mExm3kJK2xqye7nhgqZ9vPCHWGCkoZEjxj6dvrhTphgQES7Udx7AucD/
3UJF+3bETcF/JdBBktdfims5mqrx+tEMKYhBM5r2PZuv+xgrF7d2UwcWpKu34MFQ
WDeqxG32iCYOwhPPjJYaP0skTPBoQV4RvU/p/6Ot0cbrpjjY8uaFHJT5Se8uiVFd
ElozP3+WpukAC7qDlWiPxSYeTXC1CvVLnx7b34jbESS/JdLghhc3Im1tfOp56E90
5m3j+KzB4SDsLH4AxpQzafCweQ/u8vco+Vd6wUdF9arJoDg3vu0mPkRdKbyfGwsJ
aHUMhUcqYoGRPbnkETsRB3ibXtIUCv7c9anrSlDb90QUSnHIknzRWk8YUdCupKUc
oy5y7kXGsv0lvVB+UTnSYn704eFrE0QJ+YOfY/8kdZQ0Ct+jnMadxzf6Al5+Hi6R
Xt0sfTlvxi4u+5Bqp306PiWyQNKOSUXjyf/BsOlHE35dgiY69YzxbLo/Lcg5Ju6l
nMYbEXpLf+hnApmvpHy1W1mH6lqpkYh7wbxG/3zTHAK35les9DkZM81ne+0YCK4D
X83iU2h4kRUxA8yrcGm3ffGATHyMmkRN9Dh1DnEWH91pYc2qi71rAXGdcfJncmoh
iE8uyjTFPrA8YWuifhc0rfwfJJANzDQK4IbkKrxORujfREKMEk9tJKUgaNPPwpeZ
VEO9qKWY1vBNEvosjA9JYyjlpDnW7YOoPfY5os/BijP1cU3D6ysK/ywVmN8bG6rt
CU7NKimQ6LQcv6c9Ft90UFj0cAOBeSvAqTlvC3qULyyhPWkfQFoUVoBcuteFfZDq
cg4a7DUl9hNd7KY2X+8GJrHOi8eMF57jkRnNNBFPQzN7O7TTPyUm/KnvWewSu/mI
4X5IW/0hyyjFzoeQtdkGSGSXSjVEAu0ULm7ipFMaNsbc03k1UXiOFktenpavKXjU
+UiIJOm2HXEBFGXpVlS5lZMswQ==
-----END ENCRYPTED PRIVATE KEY-----
		
`),
		"tls_client_cert": cty.StringVal(`-----BEGIN CERTIFICATE-----
MIIDgTCCAmmgAwIBAgIJAM3hc9mT29AmMA0GCSqGSIb3DQEBCwUAMFYxCzAJBgNV
BAYTAlVTMQ8wDQYDVQQIDAZEZW5pYWwxFDASBgNVBAcMC1NwcmluZ2ZpZWxkMQww
CgYDVQQKDANEaXMxEjAQBgNVBAMMCVVTRVJfMTIzNDAgFw0xOTAyMjQxMDUwNDZa
GA8yMTE5MDEzMTEwNTA0NlowVjELMAkGA1UEBhMCVVMxDzANBgNVBAgMBkRlbmlh
bDEUMBIGA1UEBwwLU3ByaW5nZmllbGQxDDAKBgNVBAoMA0RpczESMBAGA1UEAwwJ
VVNFUl8xMjM0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAu3lPs4hM
hs4wfKwb6Z9lsL5QTXAhPK5AWSFc6w5JB1TudSWJLrCKgFQ9uPo1AgmvnBSqV4Lg
yah5fNdzMCnL3EU9v7OgtMDsHOQWMSYZYZBKTMOnH+JPEX9Ax1Mf9zYz2ZXLwQHr
TcPeeGDmLx2iLCtu/1z5T/XGrr9GYeP/fDvPOWNqnDqvNwu4qpi2a2mkHyYdpNxl
RSKk+Am1HVQE+V5gNjaTueG4CKAEQWjhpqmWbxDn9/DGpuP1Mzcxizt5u0bRz7oU
WLkZl0YYO2hnuM6fSk4Q/+Il9zDhfzepON2/nZ5vng9znzosEyxyjr8h3Ten2Ekp
MY/PGOizBHsSqQIDAQABo1AwTjAdBgNVHQ4EFgQUlxHPYWL7czXQvvwIGFleI/Hu
1P4wHwYDVR0jBBgwFoAUlxHPYWL7czXQvvwIGFleI/Hu1P4wDAYDVR0TBAUwAwEB
/zANBgkqhkiG9w0BAQsFAAOCAQEAVaFKtwDNex6UOY1h4tXeMSmS3jR8VjKuvRvh
maSVCv9ihaqAaG6fgo/9wG4pOx7V/3JRg9NR9ivkziddhhDcc1FMkQXDjzmfY2Aj
EWKQTTcnIRpz5mGfnfsvfAwsIrx8Wu8YI9V1AHHY7EAAlEa6dkVH7CtXTnA4dN4B
8+hIAEXwCUVXHbf1pzJLldd6KdHhJ/w0vhParWCb02bao/VK7MBAbV5DDj78WHvB
e/lI3S9RWC3s4q8pBy5x6IrD5pZ11bEy/sX8wLIEMKD8YwcTvK4D7LKDDoNCElN+
VTWMMKs/qfW0bp8LzzPlirxT16BNzuHQs3KLO29zgKEmzLI+Eg==
-----END CERTIFICATE-----
`),
		"tls_client_key_password": cty.StringVal("password"),
	}
	b := backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client := b.client

	if client == nil {
		t.Fatal("Unexpected failure, address")
	}
	if client.URL.String() != "https://127.0.0.1:8888/foo" {
		t.Fatalf("Expected address \"%s\", got \"%s\"", conf["address"], client.URL.String())
	}
	if client.UpdateMethod != "POST" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "POST", client.UpdateMethod)
	}
	if client.LockURL != nil || client.LockMethod != "LOCK" {
		t.Fatal("Unexpected lock_address or lock_method")
	}
	if client.UnlockURL != nil || client.UnlockMethod != "UNLOCK" {
		t.Fatal("Unexpected unlock_address or unlock_method")
	}
	if client.Username != "" || client.Password != "" {
		t.Fatal("Unexpected username or password")
	}

	// custom
	conf = map[string]cty.Value{
		"address":        cty.StringVal("http://127.0.0.1:8888/foo"),
		"update_method":  cty.StringVal("BLAH"),
		"lock_address":   cty.StringVal("http://127.0.0.1:8888/bar"),
		"lock_method":    cty.StringVal("BLIP"),
		"unlock_address": cty.StringVal("http://127.0.0.1:8888/baz"),
		"unlock_method":  cty.StringVal("BLOOP"),
		"username":       cty.StringVal("user"),
		"password":       cty.StringVal("pass"),
	}

	b = backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client = b.client

	if client == nil {
		t.Fatal("Unexpected failure, update_method")
	}
	if client.UpdateMethod != "BLAH" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "BLAH", client.UpdateMethod)
	}
	if client.LockURL.String() != conf["lock_address"].AsString() || client.LockMethod != "BLIP" {
		t.Fatalf("Unexpected lock_address \"%s\" vs \"%s\" or lock_method \"%s\" vs \"%s\"", client.LockURL.String(),
			conf["lock_address"].AsString(), client.LockMethod, conf["lock_method"])
	}
	if client.UnlockURL.String() != conf["unlock_address"].AsString() || client.UnlockMethod != "BLOOP" {
		t.Fatalf("Unexpected unlock_address \"%s\" vs \"%s\" or unlock_method \"%s\" vs \"%s\"", client.UnlockURL.String(),
			conf["unlock_address"].AsString(), client.UnlockMethod, conf["unlock_method"])
	}
	if client.Username != "user" || client.Password != "pass" {
		t.Fatalf("Unexpected username \"%s\" vs \"%s\" or password \"%s\" vs \"%s\"", client.Username, conf["username"],
			client.Password, conf["password"])
	}
}

func TestHTTPClientFactoryWithTLSAndRSAKey(t *testing.T) {
	// defaults

	conf := map[string]cty.Value{
		"address": cty.StringVal("https://127.0.0.1:8888/foo"),
		"tls_client_key": cty.StringVal(`
-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDSH+zGKY+LUEde
Ft2nJ4I3kI+Tl+zD2sMZuhYQ77cddGXeuAMf4RLQxqjgwsuCas8NCoTrMuB5uK05
GyIYXBegKDcAdJBcB/IE9dQZWJcTXIetkpVnddg86/l/npZj7E3r24N3EMBlkd2S
c//+9yHzkivdtY60SKNUi8Na1DsQqnhTM6Rqg0VIbxAv1PR8f4K03i9mMWRdnkDe
sxA8cZcfDsnsXwOz07CxoxWMnxyGrNrVvCR/ySQSFtLJHb+qPiWF6bbp9oNDRYpk
JyYH1YZwTr+QG4SLwgtk7gbq1KrUBXWievKLJTlkhUxBMswwoM2bxWTDIwZLG+ro
ZHheqjWVAgMBAAECggEAMBOz5idOO67zlVigAIXuqm3+G+QP/UQJjdJhCCEBAdFH
Ga16sYma93/s1fhb/gwYMcCtZu8uI0uY/s7xfydbFH7/DrCc8yGyQ2ZH0EDP2FM8
i/9VBeYVwuKvJH8Ro+1Gaue/7bc8fkDgtIisExdSglt4g/LtotxX2plb6mVS2l3s
atysGoI5vTTuaxUO6snXOjLjTFM8NJC2gyh0l8SIyGFu9NCvAuoqC+tOWwHF/bRg
TviMlhnDEeLtrdWtIzZ4ZkuPJR5SBrCWX9TAfjaD2L/lAxxBvREYMWVFgGgemat/
kLEEZ3SKzLjHNvPvNUFobGhtctWuwkcVWtNh37rXIQKBgQD3s83vkNGTPfC4LjNs
EK6njRqEnazyskAxoHUg7b4HUtTcp4My497asqtskeHe0fH2npiuQnZ4HegmOlvX
TTOSG0m89VKjPhz7W88LboXV4CASXL4+dXSFZQKwlOMg+XLtyG7pxUHjBMlfDiCo
4ADHNBiOL+xMPKzepAWb/NSzyQKBgQDZKd6haz9XMQxtJEec4F96cJNyQMhzhEGM
IS9x0mDak327hBERO7JxMwg3L9UBe832CHKzxjhgmjpkosb6h8NW6oO/jnz/tM4x
kkUC7+tR/rI0wUyhfKvpfTA1eheXuFuSG+b38tuH7jXKRie7Uu+5aWtIZFkQ9JJk
22oIBGbhbQKBgQDOLRWe8HXhD0+MnrginQgjYqnN9Mh+Aqy4Ig0caYcg5WtUdwIX
m+BlPQ6/AfZ1116FnqELezrM5GfVWgIUBaiFVr1b0P8F7a+F8Xc21roDudg4MIYR
ywY/+kHw5Rzg14E4NvtLDeu3oMZUnpfEuR8ssEo4H9+Z3W8uqmwY2KvbMQKBgQC1
RiAS+mVLMSRATtKAf0Lz/9j0vGMXGkVk5aanCofSrN99kcZ1bjGMEJ9BAep6bJAG
WhL1Qfd5nAQ2UTJrmrxSZzxGwHhTMugTtRdqVj9GmKbFJr4C5wDRzLBbU2kyOrAl
jKkGPHFITG4WRO2Rjq+RRBBLw4gdgSpailU+D/6ZGQKBgCxHeunRv+Qmy6rP6dfT
MpteqR1R4WceH7juZvbNhDnltuzPaqiFTP+zuVk4C3krxVEfXxu5QhOZuHMU5z+3
VpPIvxsMAB6/FyJqnE7GvwkstK3Lek4JkMkBJBaa28cIDNKA283eOtjoMOfaPmd4
l+E9MKxNkH8EFhyaLhuTg5+l
-----END PRIVATE KEY-----`),
		"tls_client_cert": cty.StringVal(`-----BEGIN CERTIFICATE-----
MIIDgTCCAmmgAwIBAgIJAIpW1g+5Kw+VMA0GCSqGSIb3DQEBCwUAMFYxCzAJBgNV
BAYTAlVTMQ8wDQYDVQQIDAZEZW5pYWwxFDASBgNVBAcMC1NwcmluZ2ZpZWxkMQww
CgYDVQQKDANEaXMxEjAQBgNVBAMMCVVTRVJfMTIzNDAgFw0xOTAyMjQxMDUzMTFa
GA8yMTE5MDEzMTEwNTMxMVowVjELMAkGA1UEBhMCVVMxDzANBgNVBAgMBkRlbmlh
bDEUMBIGA1UEBwwLU3ByaW5nZmllbGQxDDAKBgNVBAoMA0RpczESMBAGA1UEAwwJ
VVNFUl8xMjM0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0h/sximP
i1BHXhbdpyeCN5CPk5fsw9rDGboWEO+3HXRl3rgDH+ES0Mao4MLLgmrPDQqE6zLg
ebitORsiGFwXoCg3AHSQXAfyBPXUGViXE1yHrZKVZ3XYPOv5f56WY+xN69uDdxDA
ZZHdknP//vch85Ir3bWOtEijVIvDWtQ7EKp4UzOkaoNFSG8QL9T0fH+CtN4vZjFk
XZ5A3rMQPHGXHw7J7F8Ds9OwsaMVjJ8chqza1bwkf8kkEhbSyR2/qj4lhem26faD
Q0WKZCcmB9WGcE6/kBuEi8ILZO4G6tSq1AV1onryiyU5ZIVMQTLMMKDNm8VkwyMG
Sxvq6GR4Xqo1lQIDAQABo1AwTjAdBgNVHQ4EFgQU6J3EEyfztVkQ13TnocKVjW7O
DQkwHwYDVR0jBBgwFoAU6J3EEyfztVkQ13TnocKVjW7ODQkwDAYDVR0TBAUwAwEB
/zANBgkqhkiG9w0BAQsFAAOCAQEADOTaO+0hWHlgqmSs/AHUwiEtjIIIDhUAd+HO
TgtInj2MUOPWvmn9Y7zFwaXNdwtYnr45TP1ZInXDAbQznw27KeMW2xFLXMiPkfj7
dH5ywhNljJsFXuFelu3esKzI9H/EQhoNtGBJHIUXsfm+nueDT6KaN80KcSwWajXd
X3MnX7TRx4NtQjs7fKU0wo+4lvjkl0ED/pglQO8VnWXSN8g5+jbBQv7jzkBkX3iv
ibNnB9ynSnA/EVucL/Q0e7k8MxcFFOKBrJ2MQ3Ne2PzRPGzEfjngweJF36nhbF4T
nClq3nTR5gAz15YtuEyqj8MwNFGu4Z8w6AYNdR7Pwfzd8Z0lzQ==
-----END CERTIFICATE-----`),
		"tls_client_key_password": cty.StringVal(""),
	}
	b := backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client := b.client

	if client == nil {
		t.Fatal("Unexpected failure, address")
	}
	if client.URL.String() != "https://127.0.0.1:8888/foo" {
		t.Fatalf("Expected address \"%s\", got \"%s\"", conf["address"], client.URL.String())
	}
	if client.UpdateMethod != "POST" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "POST", client.UpdateMethod)
	}
	if client.LockURL != nil || client.LockMethod != "LOCK" {
		t.Fatal("Unexpected lock_address or lock_method")
	}
	if client.UnlockURL != nil || client.UnlockMethod != "UNLOCK" {
		t.Fatal("Unexpected unlock_address or unlock_method")
	}
	if client.Username != "" || client.Password != "" {
		t.Fatal("Unexpected username or password")
	}

	// custom
	conf = map[string]cty.Value{
		"address":        cty.StringVal("http://127.0.0.1:8888/foo"),
		"update_method":  cty.StringVal("BLAH"),
		"lock_address":   cty.StringVal("http://127.0.0.1:8888/bar"),
		"lock_method":    cty.StringVal("BLIP"),
		"unlock_address": cty.StringVal("http://127.0.0.1:8888/baz"),
		"unlock_method":  cty.StringVal("BLOOP"),
		"username":       cty.StringVal("user"),
		"password":       cty.StringVal("pass"),
	}

	b = backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client = b.client

	if client == nil {
		t.Fatal("Unexpected failure, update_method")
	}
	if client.UpdateMethod != "BLAH" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "BLAH", client.UpdateMethod)
	}
	if client.LockURL.String() != conf["lock_address"].AsString() || client.LockMethod != "BLIP" {
		t.Fatalf("Unexpected lock_address \"%s\" vs \"%s\" or lock_method \"%s\" vs \"%s\"", client.LockURL.String(),
			conf["lock_address"].AsString(), client.LockMethod, conf["lock_method"])
	}
	if client.UnlockURL.String() != conf["unlock_address"].AsString() || client.UnlockMethod != "BLOOP" {
		t.Fatalf("Unexpected unlock_address \"%s\" vs \"%s\" or unlock_method \"%s\" vs \"%s\"", client.UnlockURL.String(),
			conf["unlock_address"].AsString(), client.UnlockMethod, conf["unlock_method"])
	}
	if client.Username != "user" || client.Password != "pass" {
		t.Fatalf("Unexpected username \"%s\" vs \"%s\" or password \"%s\" vs \"%s\"", client.Username, conf["username"],
			client.Password, conf["password"])
	}
}

func TestHTTPClientFactoryWithTLSAndPKCS1RSAKey(t *testing.T) {
	// defaults

	conf := map[string]cty.Value{
		"address": cty.StringVal("https://127.0.0.1:8888/foo"),
		"tls_client_key": cty.StringVal(`-----BEGIN RSA PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: DES-CBC,4AA323C1023EAA14

ql8NawFwFQMFHI3vTrN6ClByyu4xX6zW4KbP29KKgAS/sADj1p1zYluXsp2Y0/vF
f/yCYIkuFquX9iXS/Ty7wnD+YKx3JRvrmayid2fteOJQ5Qj0pmKwWrjKBgvz9pwv
8vu0g5vwmUu7lJS770Z2aPBxVDw6iTyXO7Qc1oFbVT5NEo+aFUNzxsFnCWLAeVLc
VVYR+I/i7cnILMGDwmWPcqQexXSJPcx7W90GREBNhiioZjtrYowvOKyyCZwF9XAx
znOcMJbCyWil9OMguIEAimWaRnpETxhH3DPx0o5fpLI9k/gJZ/uZiYpGIKjRI7Sh
ZVa+wfykgtlCNLrg3WyS9jBNUvsuUqkLiuhut5fqq72HdQcCO6TicMvZa+z4pR41
MERCNWwuv5+n9YVk9WimVd8sP8jH+I2/PW+lLKtIfQ5e2ODfd3e/BmrTU/coeuRk
WHWYxa7qaPgZ5Gcd7TM2pPhS1BLVv4m0KWe1elUIbpbDHNpGcKkm7KWkZwiiFYMd
nvEFMRQVaCpFMpDey8kW/M7Jpmv+EbC3ZrNsIbMmcncAYa4LesmCjlUzxyTjN5CE
+QJ//X4SFBl/epu0Bqg+QrJNz6IDVc8BHyYduKHph4T+hjE7VldEw8hwYk0JFG/j
fklZ0EpWFj70OTYJFB3dwwyu5Kq5e2LHZPg1DGwq+IQXfF71oJwOKO7Z45vWRP+2
N8mnG3BD1FHEBAEfFjNONXkilmDmZeI+77EZEHD2ziZ421AGwdbv20Qt2cTcRfcN
yYVqrs75OJrJNwxaXPOJLPUCdvT73pOG2zne5PsqBBeykgObmdsBkQvhTXbRvSJm
WVUfA5clNpbBfOWuhEWayH9OfhTfzfIzATrv59CDYpTs0D5v/CbNptQuRAS7084D
TJNcMxatCdvx6wCl0wYBgqxYKW1MG3BqyX0QUmFgO9Cr/ljbx3khZvlk7aZE2TEW
3bSMpcMejnafoIzKWFgNVgq1hcdy8qT1H7+j6yuXqcJfx2wI7jk1+ivgyuD0CqXG
IWnM1PSN+F7njCm/HW4uEg+fuT+n5EO/4B+CdDODSaGLd/PU5NRTWDyXQ1XGk2dW
0AtXuocjnCl0dZTsNfEn0bfhugGh8PMkhSJDcVIDJrNX9T/RYgQY3SUpPzbc4PPf
gZOoM+R9Ul2cSCLv++smGoNwOWWN/piFXYGEtERMQDFE3ZGu3sL5ePOQF2kJoOIf
XTCF1w+u8jdskBpVJm247w4ZhG4NTetp7f63J+YKyf+9ryTAg8gPrA+T9H+mMMO8
T5XmSo/0XIGvFxQJoqN+SxBdVKoIe9atrg7slXIoD0rf9jAtpEYo1WoGA+oKEWFg
rPFeOCf1Eclbo1uElS5e1yjq30TySiMlfRFICcKJ8ggZzFH6WLXjE7a0kDRyPOQu
Q2j326pKmVD08V+cvo2XUkb/CQ2mQJw0yigeC/VQON2IjpCZAEUCzs8N3idHP1uh
oNUyBmzo9RVeODQtWdyybePMOQsSpLMqauuHcWNxE6GYkWe7x4ZmQV/G64OX7+SW
NTZV60U09XwxtbTAwg/tpDuz6ZhD6388J/24t7x2p4UykBd5gMTMj7a2YHf92Xvs
-----END RSA PRIVATE KEY-----
`),
		"tls_client_cert": cty.StringVal(`-----BEGIN CERTIFICATE-----
MIIDgTCCAmmgAwIBAgIJANFDBEonczdkMA0GCSqGSIb3DQEBCwUAMFYxCzAJBgNV
BAYTAlVTMQ8wDQYDVQQIDAZEZW5pYWwxFDASBgNVBAcMC1NwcmluZ2ZpZWxkMQww
CgYDVQQKDANEaXMxEjAQBgNVBAMMCVVTRVJfMTIzNDAgFw0xOTAyMjQxNzMzMTRa
GA8yMTE5MDEzMTE3MzMxNFowVjELMAkGA1UEBhMCVVMxDzANBgNVBAgMBkRlbmlh
bDEUMBIGA1UEBwwLU3ByaW5nZmllbGQxDDAKBgNVBAoMA0RpczESMBAGA1UEAwwJ
VVNFUl8xMjM0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAw1jq3r70
AGsl4gxINHL4Np9V7RCVMbMHK50UDGLNybHvLjHJ1CRMD7IgKEhFs3QcCn6I+2oi
ffFApLDtJzIR0gzkf8/B/EqI9xya5/xb14io7usivN80VDkWIQVb+pgxLUFO2WlE
OhJA+1EWUYBuUEk4nCg2gMiJa+ZV8qxOzhD1MB8ohMorKT4aXYEpKz4LG2QpWhiD
i0dMo+QChgag7se94uGf4gn6e9yV4qaNYPbgqtlBLNsRakNGtFlZ2MsffKzs1SxY
chBsMly00cw3eytMqZSQLaygIY3x9zrsE6YNVGHGTUghxt6Ykgn4/ftDkhP6pQHP
r0nhj4h++/u9zwIDAQABo1AwTjAdBgNVHQ4EFgQUpsgAeKFtCTrMnsQV9DYigWgJ
AmowHwYDVR0jBBgwFoAUpsgAeKFtCTrMnsQV9DYigWgJAmowDAYDVR0TBAUwAwEB
/zANBgkqhkiG9w0BAQsFAAOCAQEAPyo0x6Uzyj6RaEzukWQFVekJh3tkbNvHU5zd
bkC4z6wAo/fvoT8juu79sNgBwxuArr4K3Csudq18tKZ7iB2wMK46fPpCjI8xii3f
x7M9z/sw7eCHn+kusRPZf6M2aq4EULNK7nVjYuSqOso1akMRqFDtS3HLYMYhoydq
eGPdgZ5ZSBs7oBedHYWcIUrYlzokxTvtFoYMSjY2xgT6GO/SoqQq7/az3BdPYRYg
ATj7mumxFOTcRNdU61lPEBqv0C3UHjnfe7zBh2fqEj6XSBbbnC17ACGQ4F5iCxYz
LlLLSKNIYEC0E1fIhGpBix16CkjEFFFNEuDJ/GdNl0F4Lo3Itg==
-----END CERTIFICATE-----
`),
		"tls_client_key_password": cty.StringVal("password"),
	}
	b := backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client := b.client

	if client == nil {
		t.Fatal("Unexpected failure, address")
	}
	if client.URL.String() != "https://127.0.0.1:8888/foo" {
		t.Fatalf("Expected address \"%s\", got \"%s\"", conf["address"], client.URL.String())
	}
	if client.UpdateMethod != "POST" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "POST", client.UpdateMethod)
	}
	if client.LockURL != nil || client.LockMethod != "LOCK" {
		t.Fatal("Unexpected lock_address or lock_method")
	}
	if client.UnlockURL != nil || client.UnlockMethod != "UNLOCK" {
		t.Fatal("Unexpected unlock_address or unlock_method")
	}
	if client.Username != "" || client.Password != "" {
		t.Fatal("Unexpected username or password")
	}

	// custom
	conf = map[string]cty.Value{
		"address":        cty.StringVal("http://127.0.0.1:8888/foo"),
		"update_method":  cty.StringVal("BLAH"),
		"lock_address":   cty.StringVal("http://127.0.0.1:8888/bar"),
		"lock_method":    cty.StringVal("BLIP"),
		"unlock_address": cty.StringVal("http://127.0.0.1:8888/baz"),
		"unlock_method":  cty.StringVal("BLOOP"),
		"username":       cty.StringVal("user"),
		"password":       cty.StringVal("pass"),
	}

	b = backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client = b.client

	if client == nil {
		t.Fatal("Unexpected failure, update_method")
	}
	if client.UpdateMethod != "BLAH" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "BLAH", client.UpdateMethod)
	}
	if client.LockURL.String() != conf["lock_address"].AsString() || client.LockMethod != "BLIP" {
		t.Fatalf("Unexpected lock_address \"%s\" vs \"%s\" or lock_method \"%s\" vs \"%s\"", client.LockURL.String(),
			conf["lock_address"].AsString(), client.LockMethod, conf["lock_method"])
	}
	if client.UnlockURL.String() != conf["unlock_address"].AsString() || client.UnlockMethod != "BLOOP" {
		t.Fatalf("Unexpected unlock_address \"%s\" vs \"%s\" or unlock_method \"%s\" vs \"%s\"", client.UnlockURL.String(),
			conf["unlock_address"].AsString(), client.UnlockMethod, conf["unlock_method"])
	}
	if client.Username != "user" || client.Password != "pass" {
		t.Fatalf("Unexpected username \"%s\" vs \"%s\" or password \"%s\" vs \"%s\"", client.Username, conf["username"],
			client.Password, conf["password"])
	}
}
