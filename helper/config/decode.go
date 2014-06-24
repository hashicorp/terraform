package config

import (
	"github.com/mitchellh/mapstructure"
)

func Decode(target interface{}, raws ...interface{}) (*mapstructure.Metadata, error) {
	var md mapstructure.Metadata
	decoderConfig := &mapstructure.DecoderConfig{
		Metadata:         &md,
		Result:           target,
		WeaklyTypedInput: true,
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, err
	}

	for _, raw := range raws {
		err := decoder.Decode(raw)
		if err != nil {
			return nil, err
		}
	}

	return &md, nil
}
