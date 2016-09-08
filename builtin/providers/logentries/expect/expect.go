package test

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"reflect"
	"regexp"
)

type TestExistsCheckFactoryFunc func(resource string, fact interface{}) resource.TestCheckFunc

type TestExpectValue interface {
	Execute(val interface{}) error
	String() string
}

func TestCheckResourceExpectation(res string, fact interface{}, existsFunc TestExistsCheckFactoryFunc, expectation map[string]TestExpectValue) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if err := existsFunc(res, fact)(s); err != nil {
			return fmt.Errorf("Expectation existence check error: %s", err)
		}

		value := reflect.ValueOf(fact).Elem()
		for i := 0; i < value.NumField(); i++ {
			t := value.Type().Field(i).Tag
			tv := t.Get("tfresource")

			// TODO: Support data types other than string
			fv := value.Field(i).Interface().(string)
			if expect, ok := expectation[tv]; ok {
				if err := expect.Execute(fv); err != nil {
					return fmt.Errorf("Expected %s, got \"%s\" (for %s)", expectation[tv], fv, tv)
				}
			}
		}
		return nil
	}
}

type RegexTestExpectValue struct {
	Value interface{}
	TestExpectValue
}

func (t *RegexTestExpectValue) Execute(val interface{}) error {
	expr := t.Value.(string)
	if !regexp.MustCompile(expr).MatchString(val.(string)) {
		return fmt.Errorf("Expected regexp match for \"%s\": %s", expr, val)
	}
	return nil
}

func (t *RegexTestExpectValue) String() string {
	return fmt.Sprintf("regex[%s]", t.Value.(string))
}

func RegexMatches(exp string) TestExpectValue {
	return &RegexTestExpectValue{Value: exp}
}

type EqualsTestExpectValue struct {
	Value interface{}
	TestExpectValue
}

func (t *EqualsTestExpectValue) Execute(val interface{}) error {
	expr := t.Value.(string)
	if val.(string) != t.Value.(string) {
		return fmt.Errorf("Expected %s and %s to be equal", expr, val)
	}
	return nil
}

func (t *EqualsTestExpectValue) String() string {
	return fmt.Sprintf("equals[%s]", t.Value.(string))
}

func Equals(exp string) TestExpectValue {
	return &EqualsTestExpectValue{Value: exp}
}
