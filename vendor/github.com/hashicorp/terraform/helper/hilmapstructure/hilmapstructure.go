package hilmapstructure

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

var hilMapstructureDecodeHookEmptySlice []interface{}
var hilMapstructureDecodeHookStringSlice []string
var hilMapstructureDecodeHookEmptyMap map[string]interface{}

// WeakDecode behaves in the same way as mapstructure.WeakDecode but has a
// DecodeHook which defeats the backward compatibility mode of mapstructure
// which WeakDecodes []interface{}{} into an empty map[string]interface{}. This
// allows us to use WeakDecode (desirable), but not fail on empty lists.
func WeakDecode(m interface{}, rawVal interface{}) error {
	config := &mapstructure.DecoderConfig{
		DecodeHook: func(source reflect.Type, target reflect.Type, val interface{}) (interface{}, error) {
			sliceType := reflect.TypeOf(hilMapstructureDecodeHookEmptySlice)
			stringSliceType := reflect.TypeOf(hilMapstructureDecodeHookStringSlice)
			mapType := reflect.TypeOf(hilMapstructureDecodeHookEmptyMap)

			if (source == sliceType || source == stringSliceType) && target == mapType {
				return nil, fmt.Errorf("Cannot convert a []interface{} into a map[string]interface{}")
			}

			return val, nil
		},
		WeaklyTypedInput: true,
		Result:           rawVal,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	return decoder.Decode(m)
}
