package gobis

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"reflect"
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
	if err := json.Unmarshal(b, &m); err != nil {
		log.Errorf("unable to unmarshal JSON: %v", err)
	}
	return m
}
func GetMiddlewareName(i interface{}) string {
	return reflect.ValueOf(i).Elem().Type().Name()
}
