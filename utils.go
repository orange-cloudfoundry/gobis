package gobis

import (
	"reflect"
	"encoding/json"
)

func InterfaceToMap(is ...interface{}) map[string]interface{} {
	finalMap := make(map[string]interface{})
	for _, i := range is {
		finalMap = mergeMap(finalMap, interfaceToMap(i))
	}
	return finalMap
}
func mergeMap(parent map[string]interface{}, toMerge map[string]interface{}) map[string]interface{} {
	for key, value := range toMerge {
		parent[key] = value
	}
	return parent
}
func interfaceToMap(i interface{}) map[string]interface{} {
	b, _ := json.Marshal(i)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	return m
}
func GetMiddlewareName(i interface{}) string {
	return reflect.ValueOf(i).Elem().Type().Name()
}
