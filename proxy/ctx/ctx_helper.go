package ctx

import (
	"net/http"
	"context"
	"reflect"
	"fmt"
)

func AddContextValue(req *http.Request, key, val interface{}) {
	parentContext := req.Context()
	ctxValueReq := req.WithContext(context.WithValue(parentContext, key, val))
	*req = *ctxValueReq
}

func InjectContextValue(req *http.Request, key, inject interface{}) error {
	if reflect.TypeOf(inject).Kind() != reflect.Ptr {
		return fmt.Errorf("You should pass a pointer")
	}
	reflectType := reflect.TypeOf(inject).Elem()
	val := req.Context().Value(key)
	if val == nil {
		return nil
	}
	if !reflect.TypeOf(val).AssignableTo(reflectType) {
		return fmt.Errorf("Inject for type '%s' incompatible with type '%s'", reflect.TypeOf(val), reflectType)
	}
	reflect.ValueOf(inject).Elem().Set(reflect.ValueOf(val))
	return nil
}