package rundeck

// KeyMeta is the metadata associated with a resource in the Rundeck key store.
type KeyMeta struct {
	XMLName string `xml:"resource"`
	Name string `xml:"name,attr,omitempty"`
	Path string `xml:"path,attr,omitempty"`
	ResourceType string `xml:"type,attr,omitempty"`
	URL string `xml:"url,attr,omitempty"`
	ContentType string `xml:"resource-meta>Rundeck-content-type"`
	ContentSize string `xml:"resource-meta>Rundeck-content-size"`
	ContentMask string `xml:"resource-meta>Rundeck-content-mask"`
	KeyType string `xml:"resource-meta>Rundeck-key-type"`
	LastModifiedByUserName string `xml:"resource-meta>Rundeck-auth-modified-username"`
	CreatedByUserName string `xml:"resource-meta>Rundeck-auth-created-username"`
	CreatedTimestamp string `xml:"resource-meta>Rundeck-content-creation-time"`
	LastModifiedTimestamp string `xml:"resource-meta>Rundeck-content-modify-time"`
}

type keyMetaListContents struct {
	Keys []KeyMeta `xml:"contents>resource"`
}

// GetKeyMeta returns the metadata for the key at the given keystore path.
func (c *Client) GetKeyMeta(path string) (*KeyMeta, error) {
	k := &KeyMeta{}
	err := c.get([]string{"storage", "keys", path}, nil, k)
	return k, err
}

// GetKeysInDirMeta returns the metadata for the keys and subdirectories within
// the directory at the given keystore path.
func (c *Client) GetKeysInDirMeta(path string) ([]KeyMeta, error) {
	r := &keyMetaListContents{}
	err := c.get([]string{"storage", "keys", path}, nil, r)
	if err != nil {
		return nil, err
	}
	return r.Keys, nil
}

// GetKeyContent retrieves and returns the content of the key at the given keystore path.
// Private keys are write-only, so they cannot be retrieved via this interface.
func (c *Client) GetKeyContent(path string) (string, error) {
	return c.rawGet([]string{"storage", "keys", path}, nil, "application/pgp-keys")
}

func (c *Client) CreatePublicKey(path string, content string) error {
	return c.createOrReplacePublicKey("POST", path, "application/pgp-keys", content)
}

func (c *Client) ReplacePublicKey(path string, content string) error {
	return c.createOrReplacePublicKey("PUT", path, "application/pgp-keys", content)
}

func (c *Client) CreatePrivateKey(path string, content string) error {
	return c.createOrReplacePublicKey("POST", path, "application/octet-stream", content)
}

func (c *Client) ReplacePrivateKey(path string, content string) error {
	return c.createOrReplacePublicKey("PUT", path, "application/octet-stream", content)
}

func (c *Client) createOrReplacePublicKey(method string, path string, contentType string, content string) error {
	req := &request{
		Method: method,
		PathParts: []string{"storage", "keys", path},
		Headers: map[string]string{
			"Content-Type": contentType,
		},
		BodyBytes: []byte(content),
	}

	_, err := c.rawRequest(req)

	return err
}

func (c *Client) DeleteKey(path string) error {
	return c.delete([]string{"storage", "keys", path})
}
