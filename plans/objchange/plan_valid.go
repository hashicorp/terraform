package objchange

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs/configschema"
)

// AssertPlanValid checks checks whether a planned new state returned by a
// provider's PlanResourceChange method is suitable to achieve a change
// from priorState to config. It returns a slice with nonzero length if
// any problems are detected. Because problems here indicate bugs in the
// provider that generated the plannedState, they are written with provider
// developers as an audience, rather than end-users.
//
// All of the given values must have the same type and must conform to the
// implied type of the given schema, or this function may panic or produce
// garbage results.
//
// During planning, a provider may only make changes to attributes that are
// null (unset) in the configuration and are marked as "computed" in the
// resource type schema, in order to insert any default values the provider
// may know about. If the default value cannot be determined until apply time,
// the provider can return an unknown value. Providers are forbidden from
// planning a change that disagrees with any non-null argument in the
// configuration.
//
// As a special exception, providers _are_ allowed to provide attribute values
// conflicting with configuration if and only if the planned value exactly
// matches the corresponding attribute value in the prior state. The provider
// can use this to signal that the new value is functionally equivalent to
// the old and thus no change is required.
func AssertPlanValid(schema *configschema.Block, priorState, config, plannedState cty.Value) []error {
	return assertPlanValid(schema, priorState, config, plannedState, nil)
}

func assertPlanValid(schema *configschema.Block, priorState, config, plannedState cty.Value, path cty.Path) []error {
	var errs []error
	if plannedState.IsNull() && !config.IsNull() {
		errs = append(errs, path.NewErrorf("planned for absense but config wants existence"))
		return errs
	}
	if config.IsNull() && !plannedState.IsNull() {
		errs = append(errs, path.NewErrorf("planned for existence but config wants absense"))
		return errs
	}
	if plannedState.IsNull() {
		// No further checks possible if the planned value is null
		return errs
	}

	impTy := schema.ImpliedType()

	for name, attrS := range schema.Attributes {
		plannedV := plannedState.GetAttr(name)
		configV := config.GetAttr(name)
		priorV := cty.NullVal(attrS.Type)
		if !priorState.IsNull() {
			priorV = priorState.GetAttr(name)
		}

		path := append(path, cty.GetAttrStep{Name: name})
		moreErrs := assertPlannedValueValid(attrS, priorV, configV, plannedV, path)
		errs = append(errs, moreErrs...)
	}
	for name, blockS := range schema.BlockTypes {
		path := append(path, cty.GetAttrStep{Name: name})
		plannedV := plannedState.GetAttr(name)
		configV := config.GetAttr(name)
		priorV := cty.NullVal(impTy.AttributeType(name))
		if !priorState.IsNull() {
			priorV = priorState.GetAttr(name)
		}
		if plannedV.RawEquals(configV) {
			// Easy path: nothing has changed at all
			continue
		}
		if !plannedV.IsKnown() {
			errs = append(errs, path.NewErrorf("attribute representing nested block must not be unknown itself; set nested attribute values to unknown instead"))
			continue
		}

		switch blockS.Nesting {
		case configschema.NestingSingle:
			moreErrs := assertPlanValid(&blockS.Block, priorV, configV, plannedV, path)
			errs = append(errs, moreErrs...)
		case configschema.NestingList:
			// A NestingList might either be a list or a tuple, depending on
			// whether there are dynamically-typed attributes inside. However,
			// both support a similar-enough API that we can treat them the
			// same for our purposes here.
			if plannedV.IsNull() {
				errs = append(errs, path.NewErrorf("attribute representing a list of nested blocks must be empty to indicate no blocks, not null"))
				continue
			}

			plannedL := plannedV.LengthInt()
			configL := configV.LengthInt()
			if plannedL != configL {
				errs = append(errs, path.NewErrorf("block count in plan (%d) disagrees with count in config (%d)", plannedL, configL))
				continue
			}
			for it := plannedV.ElementIterator(); it.Next(); {
				idx, plannedEV := it.Element()
				path := append(path, cty.IndexStep{Key: idx})
				if !plannedEV.IsKnown() {
					errs = append(errs, path.NewErrorf("element representing nested block must not be unknown itself; set nested attribute values to unknown instead"))
					continue
				}
				if !configV.HasIndex(idx).True() {
					continue // should never happen since we checked the lengths above
				}
				configEV := configV.Index(idx)
				priorEV := cty.NullVal(blockS.ImpliedType())
				if !priorV.IsNull() && priorV.HasIndex(idx).True() {
					priorEV = priorV.Index(idx)
				}

				moreErrs := assertPlanValid(&blockS.Block, priorEV, configEV, plannedEV, path)
				errs = append(errs, moreErrs...)
			}
		case configschema.NestingMap:
			if plannedV.IsNull() {
				errs = append(errs, path.NewErrorf("attribute representing a map of nested blocks must be empty to indicate no blocks, not null"))
				continue
			}

			// A NestingMap might either be a map or an object, depending on
			// whether there are dynamically-typed attributes inside, but
			// that's decided statically and so all values will have the same
			// kind.
			if plannedV.Type().IsObjectType() {
				plannedAtys := plannedV.Type().AttributeTypes()
				configAtys := configV.Type().AttributeTypes()
				for k := range plannedAtys {
					if _, ok := configAtys[k]; !ok {
						errs = append(errs, path.NewErrorf("block key %q from plan is not present in config", k))
						continue
					}
					path := append(path, cty.GetAttrStep{Name: k})

					plannedEV := plannedV.GetAttr(k)
					if !plannedEV.IsKnown() {
						errs = append(errs, path.NewErrorf("element representing nested block must not be unknown itself; set nested attribute values to unknown instead"))
						continue
					}
					configEV := configV.GetAttr(k)
					priorEV := cty.NullVal(blockS.ImpliedType())
					if !priorV.IsNull() && priorV.Type().HasAttribute(k) {
						priorEV = priorV.GetAttr(k)
					}
					moreErrs := assertPlanValid(&blockS.Block, priorEV, configEV, plannedEV, path)
					errs = append(errs, moreErrs...)
				}
				for k := range configAtys {
					if _, ok := plannedAtys[k]; !ok {
						errs = append(errs, path.NewErrorf("block key %q from config is not present in plan", k))
						continue
					}
				}
			} else {
				plannedL := plannedV.LengthInt()
				configL := configV.LengthInt()
				if plannedL != configL {
					errs = append(errs, path.NewErrorf("block count in plan (%d) disagrees with count in config (%d)", plannedL, configL))
					continue
				}
				for it := plannedV.ElementIterator(); it.Next(); {
					idx, plannedEV := it.Element()
					path := append(path, cty.IndexStep{Key: idx})
					if !plannedEV.IsKnown() {
						errs = append(errs, path.NewErrorf("element representing nested block must not be unknown itself; set nested attribute values to unknown instead"))
						continue
					}
					k := idx.AsString()
					if !configV.HasIndex(idx).True() {
						errs = append(errs, path.NewErrorf("block key %q from plan is not present in config", k))
						continue
					}
					configEV := configV.Index(idx)
					priorEV := cty.NullVal(blockS.ImpliedType())
					if !priorV.IsNull() && priorV.HasIndex(idx).True() {
						priorEV = priorV.Index(idx)
					}
					moreErrs := assertPlanValid(&blockS.Block, priorEV, configEV, plannedEV, path)
					errs = append(errs, moreErrs...)
				}
				for it := configV.ElementIterator(); it.Next(); {
					idx, _ := it.Element()
					if !plannedV.HasIndex(idx).True() {
						errs = append(errs, path.NewErrorf("block key %q from config is not present in plan", idx.AsString()))
						continue
					}
				}
			}
		case configschema.NestingSet:
			if plannedV.IsNull() {
				errs = append(errs, path.NewErrorf("attribute representing a set of nested blocks must be empty to indicate no blocks, not null"))
				continue
			}

			// Because set elements have no identifier with which to correlate
			// them, we can't robustly validate the plan for a nested block
			// backed by a set, and so unfortunately we need to just trust the
			// provider to do the right thing. :(
			//
			// (In principle we could correlate elements by matching the
			// subset of attributes explicitly set in config, except for the
			// special diff suppression rule which allows for there to be a
			// planned value that is constructed by mixing part of a prior
			// value with part of a config value, creating an entirely new
			// element that is not present in either prior nor config.)
			for it := plannedV.ElementIterator(); it.Next(); {
				idx, plannedEV := it.Element()
				path := append(path, cty.IndexStep{Key: idx})
				if !plannedEV.IsKnown() {
					errs = append(errs, path.NewErrorf("element representing nested block must not be unknown itself; set nested attribute values to unknown instead"))
					continue
				}
			}

		default:
			panic(fmt.Sprintf("unsupported nesting mode %s", blockS.Nesting))
		}
	}

	return errs
}

func assertPlannedValueValid(attrS *configschema.Attribute, priorV, configV, plannedV cty.Value, path cty.Path) []error {
	var errs []error
	if plannedV.RawEquals(configV) {
		// This is the easy path: provider didn't change anything at all.
		return errs
	}
	if plannedV.RawEquals(priorV) && !priorV.IsNull() {
		// Also pretty easy: there is a prior value and the provider has
		// returned it unchanged. This indicates that configV and plannedV
		// are functionally equivalent and so the provider wishes to disregard
		// the configuration value in favor of the prior.
		return errs
	}
	if attrS.Computed && configV.IsNull() {
		// The provider is allowed to change the value of any computed
		// attribute that isn't explicitly set in the config.
		return errs
	}

	// If none of the above conditions match, the provider has made an invalid
	// change to this attribute.
	if priorV.IsNull() {
		if attrS.Sensitive {
			errs = append(errs, path.NewErrorf("sensitive planned value does not match config value"))
		} else {
			errs = append(errs, path.NewErrorf("planned value %#v does not match config value %#v", plannedV, configV))
		}
		return errs
	}
	if attrS.Sensitive {
		errs = append(errs, path.NewErrorf("sensitive planned value does not match config value nor prior value"))
	} else {
		errs = append(errs, path.NewErrorf("planned value %#v does not match config value %#v nor prior value %#v", plannedV, configV, priorV))
	}
	return errs
}
