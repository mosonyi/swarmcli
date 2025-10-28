package docker

import (
	"fmt"
	"reflect"
)

func StructFieldsAsStringArray(v interface{}) []string {
	val := reflect.ValueOf(v)
	// Ensure we have a struct
	var fields []string

	// Ensure we have a struct
	if val.Kind() == reflect.Struct {
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			// Convert each field to string and add it to the array
			fields = append(fields, fmt.Sprint(field))
		}
	}

	return fields
}
