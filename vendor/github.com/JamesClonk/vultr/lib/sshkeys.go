package lib

import "net/url"

// SSHKey on Vultr account
type SSHKey struct {
	ID      string `json:"SSHKEYID"`
	Name    string `json:"name"`
	Key     string `json:"ssh_key"`
	Created string `json:"date_created"`
}

func (c *Client) GetSSHKeys() (keys []SSHKey, err error) {
	var keyMap map[string]SSHKey
	if err := c.get(`sshkey/list`, &keyMap); err != nil {
		return nil, err
	}

	for _, key := range keyMap {
		keys = append(keys, key)
	}
	return keys, nil
}

func (c *Client) CreateSSHKey(name, key string) (SSHKey, error) {
	values := url.Values{
		"name":    {name},
		"ssh_key": {key},
	}

	var sshKey SSHKey
	if err := c.post(`sshkey/create`, values, &sshKey); err != nil {
		return SSHKey{}, err
	}
	sshKey.Name = name
	sshKey.Key = key

	return sshKey, nil
}

func (c *Client) UpdateSSHKey(key SSHKey) error {
	values := url.Values{
		"SSHKEYID": {key.ID},
	}
	if key.Name != "" {
		values.Add("name", key.Name)
	}
	if key.Key != "" {
		values.Add("ssh_key", key.Key)
	}

	if err := c.post(`sshkey/update`, values, nil); err != nil {
		return err
	}
	return nil
}

func (c *Client) DeleteSSHKey(id string) error {
	values := url.Values{
		"SSHKEYID": {id},
	}

	if err := c.post(`sshkey/destroy`, values, nil); err != nil {
		return err
	}
	return nil
}
