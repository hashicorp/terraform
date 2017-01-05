package generic

import (
	"fmt"
	"reflect"
)

type Map interface {
	IsEmpty() bool
	Count() int
	Keys() []interface{}
	Has(key interface{}) bool
	Except(keys []interface{}) Map
	IsNil(key interface{}) bool
	NotNil(key interface{}) bool
	Get(key interface{}) interface{}
	Set(key interface{}, value interface{})
	Delete(key interface{})
	String() string
}

type ConcreteMap map[interface{}]interface{}

func newEmptyMap() Map {
	return &ConcreteMap{}
}

func NewMap(data ...interface{}) Map {
	if len(data) == 0 {
		return newEmptyMap()
	} else if len(data) > 1 {
		panic("NewMap called with more than one argument")
	}

	switch data := data[0].(type) {
	case nil:
		return newEmptyMap()
	case Map:
		return data
	case map[string]string:
		stringMap := newEmptyMap()
		for key, val := range data {
			stringMap.Set(key, val)
		}
		return stringMap
	case map[string]interface{}:
		stringToInterfaceMap := newEmptyMap()
		for key, val := range data {
			stringToInterfaceMap.Set(key, val)
		}
		return stringToInterfaceMap
	case map[interface{}]interface{}:
		mapp := ConcreteMap(data)
		return &mapp
	}

	fmt.Printf("\n\n map: %T", data)
	panic("NewMap called with unexpected argument")
}

func (data *ConcreteMap) IsEmpty() bool {
	return data.Count() == 0
}

func (data *ConcreteMap) Count() int {
	return len(*data)
}

func (data *ConcreteMap) Has(key interface{}) bool {
	_, ok := (*data)[key]
	return ok
}

func (data *ConcreteMap) Except(keys []interface{}) Map {
	otherMap := NewMap()
	Each(data, func(key, value interface{}) {
		if !Contains(keys, key) {
			otherMap.Set(key, value)
		}
	})
	return otherMap
}

func (data *ConcreteMap) IsNil(key interface{}) bool {
	maybe, ok := (*data)[key]
	return ok && maybe == nil
}

func (data *ConcreteMap) NotNil(key interface{}) bool {
	maybe, ok := (*data)[key]
	return ok && maybe != nil
}

func (data *ConcreteMap) Keys() (keys []interface{}) {
	keys = make([]interface{}, 0, data.Count())
	for key := range *data {
		keys = append(keys, key)
	}

	return
}

func (data *ConcreteMap) Get(key interface{}) interface{} {
	return (*data)[key]
}

func (data *ConcreteMap) Set(key, value interface{}) {
	(*data)[key] = value
}

func (data *ConcreteMap) Delete(key interface{}) {
	delete(*data, key)
}

func (data *ConcreteMap) String() string {
	return fmt.Sprintf("% v", *data)
}

func IsMappable(value interface{}) bool {
	if value == nil {
		return false
	}
	switch value.(type) {
	case Map:
		return true
	default:
		return reflect.TypeOf(value).Kind() == reflect.Map
	}
}

type Iterator func(key, val interface{})

func Each(collection Map, cb Iterator) {
	for _, key := range collection.Keys() {
		cb(key, collection.Get(key))
	}
}

func Contains(collection, item interface{}) bool {
	switch collection := collection.(type) {
	case Map:
		return collection.Has(item)
	case []interface{}:
		for _, val := range collection {
			if val == item {
				return true
			}
		}
		return false
	}

	panic("unexpected type passed to Contains")
}
