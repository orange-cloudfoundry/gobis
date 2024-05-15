package gobis

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
)

// AddContextValue Add a context value to a http request without having to override request by yourself
func AddContextValue(req *http.Request, key, val interface{}) {
	parentContext := req.Context()
	ctxValueReq := req.WithContext(context.WithValue(parentContext, key, val))
	*req = *ctxValueReq
}

// InjectContextValue Inject a value from a request context to an interface
// This is the same behaviour as json.Unmarshal
func InjectContextValue(req *http.Request, key, inject interface{}) error {
	if reflect.TypeOf(inject).Kind() != reflect.Ptr {
		return fmt.Errorf("you should pass a pointer")
	}
	reflectType := reflect.TypeOf(inject).Elem()
	val := req.Context().Value(key)
	if val == nil {
		return nil
	}
	if !reflect.TypeOf(val).AssignableTo(reflectType) {
		return fmt.Errorf("inject for type '%s' incompatible with type '%s'", reflect.TypeOf(val), reflectType)
	}
	reflect.ValueOf(inject).Elem().Set(reflect.ValueOf(val))
	return nil
}
