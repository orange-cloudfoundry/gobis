package gobis

import (
	"gopkg.in/yaml.v2"
	"runtime"
	"reflect"
	"strings"
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
	b, _ := yaml.Marshal(i)
	var m map[string]interface{}
	yaml.Unmarshal(b, &m)
	return m
}
func GetFunctionName(i interface{}) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	splittedName := strings.Split(fullName, ".")
	return splittedName[len(splittedName) - 1]
}