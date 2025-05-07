package docker

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func parseAndSumPercentLines(lines []string) float64 {
	var total float64
	for _, line := range lines {
		val := strings.TrimSuffix(strings.TrimSpace(line), "%")
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			total += f
		}
	}
	return total
}

func countNonEmptyLines(lines []string) int {
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

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
