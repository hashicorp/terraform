package pgpkeys

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/vault/helper/jsonutil"
	"github.com/keybase/go-crypto/openpgp"
)

const (
	kbPrefix = "keybase:"
)

// FetchKeybasePubkeys fetches public keys from Keybase given a set of
// usernames, which are derived from correctly formatted input entries. It
// doesn't use their client code due to both the API and the fact that it is
// considered alpha and probably best not to rely on it.  The keys are returned
// as base64-encoded strings.
func FetchKeybasePubkeys(input []string) (map[string]string, error) {
	client := cleanhttp.DefaultClient()
	if client == nil {
		return nil, fmt.Errorf("unable to create an http client")
	}

	if len(input) == 0 {
		return nil, nil
	}

	usernames := make([]string, 0, len(input))
	for _, v := range input {
		if strings.HasPrefix(v, kbPrefix) {
			usernames = append(usernames, strings.TrimPrefix(v, kbPrefix))
		}
	}

	if len(usernames) == 0 {
		return nil, nil
	}

	ret := make(map[string]string, len(usernames))
	url := fmt.Sprintf("https://keybase.io/_/api/1.0/user/lookup.json?usernames=%s&fields=public_keys", strings.Join(usernames, ","))
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	type publicKeys struct {
		Primary struct {
			Bundle string
		}
	}

	type them struct {
		publicKeys `json:"public_keys"`
	}

	type kbResp struct {
		Status struct {
			Name string
		}
		Them []them
	}

	out := &kbResp{
		Them: []them{},
	}

	if err := jsonutil.DecodeJSONFromReader(resp.Body, out); err != nil {
		return nil, err
	}

	if out.Status.Name != "OK" {
		return nil, fmt.Errorf("got non-OK response: %s", out.Status.Name)
	}

	missingNames := make([]string, 0, len(usernames))
	var keyReader *bytes.Reader
	serializedEntity := bytes.NewBuffer(nil)
	for i, themVal := range out.Them {
		if themVal.Primary.Bundle == "" {
			missingNames = append(missingNames, usernames[i])
			continue
		}
		keyReader = bytes.NewReader([]byte(themVal.Primary.Bundle))
		entityList, err := openpgp.ReadArmoredKeyRing(keyReader)
		if err != nil {
			return nil, err
		}
		if len(entityList) != 1 {
			return nil, fmt.Errorf("primary key could not be parsed for user %s", usernames[i])
		}
		if entityList[0] == nil {
			return nil, fmt.Errorf("primary key was nil for user %s", usernames[i])
		}

		serializedEntity.Reset()
		err = entityList[0].Serialize(serializedEntity)
		if err != nil {
			return nil, fmt.Errorf("error serializing entity for user %s: %s", usernames[i], err)
		}

		// The API returns values in the same ordering requested, so this should properly match
		ret[kbPrefix+usernames[i]] = base64.StdEncoding.EncodeToString(serializedEntity.Bytes())
	}

	if len(missingNames) > 0 {
		return nil, fmt.Errorf("unable to fetch keys for user(s) %s from keybase", strings.Join(missingNames, ","))
	}

	return ret, nil
}
