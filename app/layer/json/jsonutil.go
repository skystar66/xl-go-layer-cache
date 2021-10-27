package json

import jsoniter "github.com/json-iterator/go"

func ToJson(obj interface{}) string {
	value, _ := jsoniter.MarshalToString(obj)
	return value
}
func FromJson(json string, obj interface{}) interface{} {
	jsoniter.UnmarshalFromString(json, obj)
	return obj
}
