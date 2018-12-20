package dynblock

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

type iteration struct {
	IteratorName string
	Key          cty.Value
	Value        cty.Value
	Inherited    map[string]*iteration
}

func (s *expandSpec) MakeIteration(key, value cty.Value) *iteration {
	return &iteration{
		IteratorName: s.iteratorName,
		Key:          key,
		Value:        value,
		Inherited:    s.inherited,
	}
}

func (i *iteration) Object() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"key":   i.Key,
		"value": i.Value,
	})
}

func (i *iteration) EvalContext(base *hcl.EvalContext) *hcl.EvalContext {
	new := base.NewChild()

	if i != nil {
		new.Variables = map[string]cty.Value{}
		for name, otherIt := range i.Inherited {
			new.Variables[name] = otherIt.Object()
		}
		new.Variables[i.IteratorName] = i.Object()
	}

	return new
}

func (i *iteration) MakeChild(iteratorName string, key, value cty.Value) *iteration {
	if i == nil {
		// Create entirely new root iteration, then
		return &iteration{
			IteratorName: iteratorName,
			Key:          key,
			Value:        value,
		}
	}

	inherited := map[string]*iteration{}
	for name, otherIt := range i.Inherited {
		inherited[name] = otherIt
	}
	inherited[i.IteratorName] = i
	return &iteration{
		IteratorName: iteratorName,
		Key:          key,
		Value:        value,
		Inherited:    inherited,
	}
}
