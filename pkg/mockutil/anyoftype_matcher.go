package mockutil

import (
	"fmt"
	"reflect"

	"github.com/golang/mock/gomock"
)

func AnyOfType(val interface{}) gomock.Matcher {
	return &anyOfTypeMatcher{argType: getType(val)}
}

type anyOfTypeMatcher struct{ argType reflect.Type }

func (m anyOfTypeMatcher) Matches(x interface{}) bool {
	return m.argType == getType(x)
}

func (m anyOfTypeMatcher) String() string {
	return fmt.Sprintf("of type %v", m.argType)
}

func getType(val interface{}) reflect.Type {
	t := reflect.TypeOf(val)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}
