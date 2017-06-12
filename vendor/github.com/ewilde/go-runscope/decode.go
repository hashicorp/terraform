package runscope

import (
	"reflect"
	"time"
	"github.com/mitchellh/mapstructure"
)

func floatToTimeDurationHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.Float64 {
			return data, nil
		}

		if t != reflect.TypeOf(time.Now()) {
			return data, nil
		}

		// Convert it by parsing
		rawValue := data.(float64)
		seconds := int64(rawValue)
		nanoSeconds := int64((rawValue - float64(int64(rawValue)))*1e9)
		return time.Unix(seconds, nanoSeconds), nil
	}
}

func decode(result interface{}, response interface{}) error {

	config := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   result,
		TagName:  "json",
		DecodeHook: floatToTimeDurationHookFunc(),
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		panic(err)
	}

	err = decoder.Decode(response)
	return err
}
