package differ

// ChangeSlice is a Change that represents a Tuple, Set, or List type, and has
// converted the relevant interfaces into slices for easier access.
type ChangeSlice struct {
	// Before contains the value before the proposed change.
	Before []interface{}

	// After contains the value after the proposed change.
	After []interface{}

	// Unknown contains the unknown status of any elements of this list/set.
	Unknown []interface{}

	// BeforeSensitive contains the before sensitive status of any elements of
	//this list/set.
	BeforeSensitive []interface{}

	// AfterSensitive contains the after sensitive status of any elements of
	//this list/set.
	AfterSensitive []interface{}

	// ReplacePaths matches the same attributes in Change exactly.
	ReplacePaths []interface{}
}

func (change Change) asSlice() ChangeSlice {
	return ChangeSlice{
		Before:          genericToSlice(change.Before),
		After:           genericToSlice(change.After),
		Unknown:         genericToSlice(change.Unknown),
		BeforeSensitive: genericToSlice(change.BeforeSensitive),
		AfterSensitive:  genericToSlice(change.AfterSensitive),
		ReplacePaths:    change.ReplacePaths,
	}
}

func (s ChangeSlice) getChild(beforeIx, afterIx int, propagateReplace bool) Change {
	before, beforeExplicit := getFromGenericSlice(s.Before, beforeIx)
	after, afterExplicit := getFromGenericSlice(s.After, afterIx)
	unknown, _ := getFromGenericSlice(s.Unknown, afterIx)
	beforeSensitive, _ := getFromGenericSlice(s.BeforeSensitive, beforeIx)
	afterSensitive, _ := getFromGenericSlice(s.AfterSensitive, afterIx)

	return Change{
		BeforeExplicit:  beforeExplicit,
		AfterExplicit:   afterExplicit,
		Before:          before,
		After:           after,
		Unknown:         unknown,
		BeforeSensitive: beforeSensitive,
		AfterSensitive:  afterSensitive,
		ReplacePaths:    s.processReplacePaths(beforeIx, propagateReplace),
	}
}

func (s ChangeSlice) processReplacePaths(ix int, propagateReplace bool) []interface{} {
	var ret []interface{}
	for _, p := range s.ReplacePaths {
		path := p.([]interface{})

		if len(path) == 0 {
			// This means that the current value is causing a replacement but
			// not its children. Normally, we'd skip this as we do with maps
			// but sets display the replace suffix on all their children even
			// if they themselves are specified, so we want to pass this on.
			if propagateReplace {
				ret = append(ret, path)
			}
			// If we don't want to propagate the replace we just skip over this
			// entry. If we do, we've added it to the returned set of paths
			// already, so we still want to skip over the rest of this.
			continue
		}

		if int(path[0].(float64)) == ix {
			ret = append(ret, path[1:])
		}
	}
	return ret
}

func getFromGenericSlice(generic []interface{}, ix int) (interface{}, bool) {
	if generic == nil {
		return nil, false
	}
	if ix < 0 || ix >= len(generic) {
		return nil, false
	}
	return generic[ix], true
}

func genericToSlice(generic interface{}) []interface{} {
	if concrete, ok := generic.([]interface{}); ok {
		return concrete
	}
	return nil
}
