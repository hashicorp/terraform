package helper

import (
	"github.com/hashicorp/terraform/helper/schema"
	"strconv"
)

type Elem map[string]interface{}

func (elem *Elem) GetOk(key string) (interface{}, bool) {
	if value, ok := (*elem)[key]; ok {
		return value, true
	}
	return nil, false
}

func (elem *Elem) Get(key string) interface{} {
	if value, ok := (*elem)[key]; ok {
		return value
	}
	return nil
}

func (elem *Elem) GetString(key string) string {
	if v, ok := elem.GetOk(key); ok == true {
		return string(v.(string))
	}
	return ""
}

func (elem *Elem) GetBool(key string) bool {
	if v, ok := elem.GetOk(key); ok == true {
		b, err := strconv.ParseBool(v.(string))
		if err != nil {
			return false
		}

		return b
	}
	return false
}

func (elem *Elem) GetInt(key string) int {
	if v, ok := elem.GetOk(key); ok == true {
		return int(v.(int))
	}
	return 0
}

func (elem *Elem) GetFloat(key string) float64 {
	if v, ok := elem.GetOk(key); ok == true {
		return float64(v.(float64))
	}
	return 0
}

func (elem *Elem) GetList(key string) *schema.Set {
	return elem.GetSet(key)
}

func (elem *Elem) GetSet(key string) *schema.Set {
	if v, ok := elem.GetOk(key); ok {
		return v.(*schema.Set)
	}
	return nil
}

func (elem *Elem) Contains(key string) bool {
	d := *elem
	_, ok := d[key]
	return ok
}

func ListSet(set *schema.Set) []Elem {
	if set == nil || set.Len() == 0 {
		return []Elem{}
	}

	var elementsTemp []Elem
	var elements []Elem
	for _, elem := range set.List() {
		if element, ok := elem.(map[string]interface{}); ok {
			elementsTemp = append(elementsTemp, Elem(element))
		}
	}

	for _, elem := range elementsTemp {
		if elem == nil || len(elem) == 0 {
			continue
		}
		elements = append(elements, elem)
	}

	return elements
}
