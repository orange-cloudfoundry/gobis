package utils

import "encoding/json"

func InterfaceToMap(i interface{}) map[string]interface{} {
	b, _ := json.Marshal(i)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	return m
}
