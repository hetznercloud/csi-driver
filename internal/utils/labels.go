package utils

import (
	"errors"
	"strings"
)

func ConvertLabelsToMap(labelsString string) (map[string]string, error) {
	result := map[string]string{}
	splitFn := func(c rune) bool {
		return c == ','
	}
	vals := strings.FieldsFunc(labelsString, splitFn)
	for _, val := range vals {
		pair := strings.SplitN(val, "=", 2)
		key := strings.TrimSpace(pair[0])
		if key == "" {
			return nil, errors.New("empty key")
		}
		value := ""
		if len(pair) > 1 {
			value = strings.TrimSpace(pair[1])
		}
		result[strings.TrimSpace(pair[0])] = value
	}
	return result, nil
}
