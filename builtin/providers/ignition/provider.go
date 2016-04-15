package ignition

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/coreos/ignition/config/types"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

type cache struct {
	users map[string]*types.User
}

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"ignition_config": resourceConfig(),
			"ignition_user":   resourceUser(),
		},
		ConfigureFunc: func(*schema.ResourceData) (interface{}, error) {
			return &cache{
				users: make(map[string]*types.User, 0),
			}, nil
		},
	}
}

func (c *cache) addUser(u *types.User) string {
	id := id(u)
	c.users[id] = u

	return id
}

func id(input interface{}) string {
	b, _ := json.Marshal(input)
	return hash(string(b))
}

func hash(s string) string {
	sha := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sha[:])
}
