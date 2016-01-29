package chef

import "fmt"

type ACLService struct {
	client *Client
}

// ACL represents the native Go version of the deserialized ACL type
type ACL map[string]ACLitems

// ACLitems
type ACLitems struct {
	Groups ACLitem `json:"groups"`
	Actors ACLitem `json:"actors"`
}

// ACLitem
type ACLitem []string

func NewACL(acltype string, actors, groups ACLitem) (acl *ACL) {
	acl = &ACL{
		acltype: ACLitems{
			Actors: actors,
			Groups: groups,
		},
	}
	return
}

// Get gets an ACL from the Chef server.
//
// Chef API docs: lol
func (a *ACLService) Get(subkind string, name string) (acl ACL, err error) {
	url := fmt.Sprintf("%s/%s/_acl", subkind, name)
	err = a.client.magicRequestDecoder("GET", url, nil, &acl)
	return
}

// Put updates an ACL on the Chef server.
//
// Chef API docs: rofl
func (a *ACLService) Put(subkind, name string, acltype string, item *ACL) (err error) {
	url := fmt.Sprintf("%s/%s/_acl/%s", subkind, name, acltype)
	body, err := JSONReader(item)
	if err != nil {
		return
	}

	err = a.client.magicRequestDecoder("PUT", url, body, nil)
	return
}
