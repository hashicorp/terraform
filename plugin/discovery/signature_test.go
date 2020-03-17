package discovery

import (
	"encoding/base64"
	"strings"
	"testing"

	"golang.org/x/crypto/openpgp"
)

func TestVerifyProviderSignature(t *testing.T) {
	testAuthorSignatureGood, err := base64.StdEncoding.DecodeString(
		testAuthorSignatureGoodBase64)
	if err != nil {
		t.Fatal(err)
	}

	testHashicorpSignatureGood, err := base64.StdEncoding.DecodeString(
		testHashicorpSignatureGoodBase64)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string

		shasums             []byte
		signature           []byte
		authorKeyArmor      string
		trustSignatureArmor string

		expectedIdentity string
		expectedTier     ProviderTier
		expectedError    string
	}{
		{
			name:                "valid community",
			shasums:             testShaSums,
			signature:           testAuthorSignatureGood,
			authorKeyArmor:      testAuthorKeyArmor,
			trustSignatureArmor: "",
			expectedIdentity:    "37A6AB3BCF2C170A Terraform Testing (plugin/discovery/) <terraform+testing@hashicorp.com>",
			expectedTier:        providerTierCommunity,
			expectedError:       "",
		},
		{
			name:                "valid partner",
			shasums:             testShaSums,
			signature:           testAuthorSignatureGood,
			authorKeyArmor:      testAuthorKeyArmor,
			trustSignatureArmor: testAuthorKeyTrustSignatureArmor,
			expectedIdentity:    "37A6AB3BCF2C170A Terraform Testing (plugin/discovery/) <terraform+testing@hashicorp.com>",
			expectedTier:        providerTierPartner,
			expectedError:       "",
		},
		{
			name:                "valid official",
			shasums:             testShaSums,
			signature:           testHashicorpSignatureGood,
			authorKeyArmor:      HashicorpPublicKey,
			trustSignatureArmor: "",
			expectedIdentity:    "51852D87348FFC4C HashiCorp Security <security@hashicorp.com>",
			expectedTier:        providerTierOfficial,
			expectedError:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity, tier, err := verifyProviderSignature(
				tt.shasums, tt.signature,
				tt.authorKeyArmor, tt.trustSignatureArmor,
			)
			if tt.expectedError == "" {
				if err != nil {
					t.Fatalf("err: %s", err)
				}
			} else {
				t.Fatalf("TODO")
			}

			identity := entityString(entity)
			if identity != tt.expectedIdentity {
				t.Fatalf("wanted identity %q, got %q",
					tt.expectedIdentity, identity)
			}

			if tier != tt.expectedTier {
				t.Fatalf("wanted tier %q, got %q",
					tt.expectedTier, tier)
			}
		})
	}

}

func TestEntityString(t *testing.T) {
	var tests = []struct {
		name     string
		entity   *openpgp.Entity
		expected string
	}{
		{
			"nil",
			nil,
			"",
		},
		{
			"testAuthorKeyArmor",
			testReadArmoredEntity(t, testAuthorKeyArmor),
			"37A6AB3BCF2C170A Terraform Testing (plugin/discovery/) <terraform+testing@hashicorp.com>",
		},
		{
			"HashicorpPublicKey",
			testReadArmoredEntity(t, HashicorpPublicKey),
			"51852D87348FFC4C HashiCorp Security <security@hashicorp.com>",
		},
		{
			"HashicorpPartnersKey",
			testReadArmoredEntity(t, HashicorpPartnersKey),
			"7D72D4268E4660FC HashiCorp Security (Terraform Partner Signing) <security+terraform@hashicorp.com>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := entityString(tt.entity)
			if actual != tt.expected {
				t.Errorf("expected %s, actual %s", tt.expected, actual)
			}
		})
	}

}

func testReadArmoredEntity(t *testing.T, armor string) *openpgp.Entity {
	data := strings.NewReader(armor)

	el, err := openpgp.ReadArmoredKeyRing(data)
	if err != nil {
		t.Fatal(err)
	}

	if count := len(el); count != 1 {
		t.Fatalf("expected 1 entity, got %d", count)
	}

	return el[0]
}

// testAuthorKeyArmor is test key ID 5BFEEC4317E746008621970637A6AB3BCF2C170A.
var testAuthorKeyArmor = `-----BEGIN PGP PUBLIC KEY BLOCK-----

mQENBF5vhgYBCAC40OcC2hEx3yGiLhHMbt7DAVEQ0nZwAWy6oL98niknLumBa1VO
nMYshP+o/FKOFatBl8aXhmDo606P6pD9d4Pg/WNehqT7hGNHcAFlm+8qjQAvE5uX
Z/na/Np7dmWasCiL5hYyHEnKU/XFpc9KyicbkS7n8igP1LEb8xDD1pMLULQsQHA4
258asvtwjoYTZIij1I6bUE178bGFPNCfj+FzQM8nKzPpDVxZ7njN9c2sB9FEdJ1+
S9mZQNK5PbJuEAOpD5Jp9BnGE16jsLUhDmvGHBjFZAXMBkNSloEMHhs2ty9lEzoF
eJmJx7XCGw+ds1SWp4MsHQPWzXxAlrfa4GMlABEBAAG0R1RlcnJhZm9ybSBUZXN0
aW5nIChwbHVnaW4vZGlzY292ZXJ5LykgPHRlcnJhZm9ybSt0ZXN0aW5nQGhhc2hp
Y29ycC5jb20+iQFOBBMBCAA4FiEEW/7sQxfnRgCGIZcGN6arO88sFwoFAl5vhgYC
GwMFCwkIBwIGFQoJCAsCBBYCAwECHgECF4AACgkQN6arO88sFwpWvQf/apaMu4Bm
ea8AGjdl9acQhHBpWsyiHLIfZvN11xxN/f3+YITvPXIe2PMgveqNfXxu6PIeZGDb
0DBvnBQy/vqmA+sCQ8t8+kIWdfZ1EeM2YcXdmAEtriooLvc85JFYjafLIKSj9N7o
V/R/e1BCW/v1/7Je47c+6FSt3HHhwyT5AZ3BCq1zpw6PeCDSQ/gZr3Mvq4CjeLA/
K+8TM3KyOF4qBGDvzGzp/t9umQSS2L0ozd90lxJtf5Q8ozqDaBiDo+f/osXT2EvN
VwPP/xh/gABkXiNrPylFbeD+XPAC4N7NmYK5aPDzRYXXknP8e9PDMykoJKZ+bSdz
F3IZ4q5RDHmmNbkBDQReb4YGAQgAt15e1F8TPQQm1jK8+scypHgfmPHbp7Qsulo1
GTcUd8QmhbR4kayuLDEpJYzq6+IoTM4TPqsdVuq/1Nwey9oyK0wXk/SUR29nRIQh
3GBg7JVg1YsObsfVTvEflYOdjk8T/Udqs4I6HnmSbtzsaohzybutpWXPUkW8OzFI
ATwfVTrrz70Yxs+ly0nSEH2Yf+kg2uYZvv5KsJ3MNENhXnHnlaTy2IfhsxAX0xOG
pa9fXV3NzdEbl0mYaEzMi77qRAyIQ9VrIL5F0yY/LlbpLSl6xk2+BB2v3a1Ey6SJ
w4/le6AM0wlH2hKPCTlkvM0IvUWjlzrPzCkeu027iVc+fqdyiQARAQABiQE2BBgB
CAAgFiEEW/7sQxfnRgCGIZcGN6arO88sFwoFAl5vhgYCGwwACgkQN6arO88sFwqz
nAf/eF4oZG9F8sJX01mVdDm/L7Uthe4xjTdl7jwV4ygNX+pCyWrww3qc3qbd3QKg
CFqIt/TAPE/OxHxCFuxalQefpOqfxjKzvcktxzWmpgxaWsvHaXiS4bKBPz78N/Ke
MUtcjGHyLeSzYPUfjquqDzQxqXidRYhyHGSy9c0NKZ6wCElLZ6KcmCQb4sZxVwfu
ssjwAFbPMp1nr0f5SWCJfhTh7QF7lO2ldJaKMlcBM8aebmqFQ52P7ZWOFcgeerng
G7Zdrci1KEd943HhzDCsUFz4gJwbvUyiAYb2ddndpUBkYwCB/XrHWPOSnGxHgZoo
1gIqed9OV/+s5wKxZPjL0pCStQ==
=mYqJ
-----END PGP PUBLIC KEY BLOCK-----`

// testAuthorKeyTrustSignatureArmor is a trust signature of the data in
// testAuthorKeyArmor signed with HashicorpPartnersKey.
var testAuthorKeyTrustSignatureArmor = `-----BEGIN PGP SIGNATURE-----

iQIzBAABCAAdFiEEUYkGV8Ws20uCMIZWfXLUJo5GYPwFAl5w9+YACgkQfXLUJo5G
YPwjRBAAvy9jo3vvetb4qx/z2qhbRH2JbZN9byKuqlIggPzDhhaIsVJVZ9L6H6bE
AMgPe/NaH58wfiqMYenulYxj9tZwJORT/OK0Y9ZFXXZk6kWPMNv7TEppyB0wKgqq
ORKf07KjDcVQslDG9ARgnvDq2GA4UTHxhT0chKHdIKeDLmTm0VSkfNeOhQIkW7vB
S/WT9y78319QJek8OKwJo0Jv0O93rvZZI0JFjXGtP15XNBfObMtPXn3l8qoLzhsv
pJJG/u+BsVZ+y1JDQQlHaD1P2TLW/nGymFq12k693IOCmNyaIOa01Wa9B/j3a3RY
v4SdkULvJKbttNMNBgIMJ74wZp5EUhEFs68sllrIrmthH8bW2fbcHEQ1g/MJCe3+
43c9aoW8yNQmuEe7yre9lgqcJOIOxlb5XEJhH0Lh+8OBi5aHA/5wXGU5WrhWqHCR
npXBsNqy2sKUuVkEzvn3Hd6aoKncVLrgNR8xA3VP86jJhawvO+M+YYMr1wOVHc/I
PYq9hlyUR8qJ/0RpnaIE1iLbPYfEpGTg7oHORpbQVoZAUwMN/Sdox7sMkqCOb1RJ
Cmy9J5o7iiNOoshvps5cxcbsM7LNfbf0vDhWpckAvsQehrS1mfVuFHkIiotVQhH1
QXPfvB2cVF/SxMqqHWpnT+8c8klfS03kXSb0BdknrQ4DNPq1H5A=
=3A1s
-----END PGP SIGNATURE-----`

// testShaSums is a byteslice that represents the SHA256SUMS file downloaded
// for a release.
var testShaSums = []byte("example shasums data")

// testAuthorSignatureGoodBase64 is a signature of testShaSums signed with
// testAuthorKeyArmor, which represents the SHA256SUMS.sig file downloaded for
// a release.
var testAuthorSignatureGoodBase64 = `iQEzBAABCAAdFiEEW/7sQxfnRgCGIZcGN6arO88s` +
	`FwoFAl5vh7gACgkQN6arO88sFwrAlQf6Al77qzjxNIj+NQNJfBGYUE5jHIgcuWOs1IPRTYUI` +
	`rHQIUU2RVrdHoAefKTKNzGde653JK/pYTflSV+6ini3/aZZnXlF6t001w3wswmakdwTr0hXx` +
	`Ez/hHYio72Gpn7+T/L+nl6dKkjeGqd/Kor5x2TY9uYB737ESmAe5T8ZlPaGMFHh0mYlNTeRq` +
	`4qIKqL6DwddBF4Ju2svn2MeNMGfE358H31mxAl2k4PPrwBTR1sFUCUOzAXVA/g9Ov5Y9ni2G` +
	`rkTahBtV9yuUUd1D+oRTTTdP0bj3A+3xxXmKTBhRuvurydPTicKuWzeILIJkcwp7Kl5UbI2N` +
	`n1ayZdaCIw/r4w==`

// testHashicorpSignatureGoodBase64 is a signature of testShaSums signed with
// HashicorpPublicKey, which represents the SHA256SUMS.sig file downloaded for
// an official release.
var testHashicorpSignatureGoodBase64 = `iQFLBAABCAA1FiEEkabn+F0FxlYwvvGJUYUth` +
	`zSP/EwFAl5w784XHHNlY3VyaXR5QGhhc2hpY29ycC5jb20ACgkQUYUthzSP/EyB8QgAv9ijp` +
	`kTcoFwDAs+1iEUrcW18h/2cU+bvFtdqNDiffzk7+YJ9ioxeWisPta/Z6hEyhdss2+5L1MNbo` +
	`oUBLABI+Aebfxa/uYFT2kX6r/eySmlY9kqNVpjXdemOQutS4NNZxdJL7CEbh2qIKCVuyo0ul` +
	`YrTdDH35vwVyLXImWiZLnrXcT/fXLpQGx/N8PDy6WmCeju5Y5RD7TuntB71eCaCZi7wFe1tR` +
	`qSoe9tD9A7ONB0rGuCY7BxqUj0S81hhz960YbNR9Q81WoNvF7b5SmcLJ1qJx1yvBLyqya6Su` +
	`DKjU/YYCh7bwHIYzpk1/nK/7SaTHpisekqojVsfDth4TA+jGA==`
