package utils

import (
	"errors"
	"fmt"
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
		if len(pair) < 2 {
			return nil, fmt.Errorf("invalid value %s", val)
		}
		key := strings.TrimSpace(pair[0])
		if key == "" {
			return nil, errors.New("empty key")
		}
		value := strings.TrimSpace(pair[1])
		if value == "" {
			return nil, errors.New("empty value")
		}
		result[strings.TrimSpace(pair[0])] = strings.TrimSpace(pair[1])
	}
	return result, nil
}
