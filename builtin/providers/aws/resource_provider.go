package aws

type ResourceProvider struct {
}

func (p *ResourceProvider) Configure(map[string]interface{}) ([]string, error) {
	return nil, nil
}
