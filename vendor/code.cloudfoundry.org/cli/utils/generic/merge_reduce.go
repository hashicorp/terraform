package generic

func Merge(collection, otherCollection Map) Map {
	mergedMap := NewMap()

	iterator := func(key, value interface{}) {
		mergedMap.Set(key, value)
	}

	Each(collection, iterator)
	Each(otherCollection, iterator)

	return mergedMap
}

func DeepMerge(maps ...Map) Map {
	mergedMap := NewMap()
	return Reduce(maps, mergedMap, mergeReducer)
}

func mergeReducer(key, val interface{}, reduced Map) Map {
	switch {
	case reduced.Has(key) == false:
		reduced.Set(key, val)
		return reduced

	case IsMappable(val):
		maps := []Map{NewMap(reduced.Get(key)), NewMap(val)}
		mergedMap := Reduce(maps, NewMap(), mergeReducer)
		reduced.Set(key, mergedMap)
		return reduced

	case IsSliceable(val):
		reduced.Set(key, append(reduced.Get(key).([]interface{}), val.([]interface{})...))
		return reduced

	default:
		reduced.Set(key, val)
		return reduced
	}
}

type Reducer func(key, val interface{}, reducedVal Map) Map

func Reduce(collections []Map, resultVal Map, cb Reducer) Map {
	for _, collection := range collections {
		for _, key := range collection.Keys() {
			resultVal = cb(key, collection.Get(key), resultVal)
		}
	}
	return resultVal
}
