package main

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func ObjectToJSONStringFormatted(conf interface{}) string {
	b, err := json.Marshal(conf)
	if err != nil {
		return fmt.Sprintf("%+v", conf)
	}
	var out bytes.Buffer
	err = json.Indent(&out, b, "", "    ")
	if err != nil {
		return fmt.Sprintf("%+v", conf)
	}
	return out.String()
}

func ObjectToJSONByte(obj interface{}) []byte {
	b, err := json.Marshal(obj)
	if err != nil {
		return []byte("{}")
	}
	return b
}

func ObjectToJSONString(obj interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func JSONStringObject(jsonStr string, obj interface{}) bool {
	return JSONBytesObject([]byte(jsonStr), obj)
}

func JSONBytesObject(jsonBytes []byte, obj interface{}) bool {
	err := json.Unmarshal(jsonBytes, &obj)
	return err == nil
}
