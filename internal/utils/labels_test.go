package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertLabelsToMap(t *testing.T) {
	tests := []struct {
		name       string
		env        string
		expected   map[string]string
		errMessage string
	}{
		{"valid", "test1=test1", map[string]string{"test1": "test1"}, ""},
		{"mutiple items", "test1=test1,test2=test2", map[string]string{"test1": "test1", "test2": "test2"}, ""},
		{"empty", "", map[string]string{}, ""},
		{"multiple colons", "test1=test1=test1", map[string]string{"test1": "test1=test1"}, ""},
		{"space", "test1= test1", map[string]string{"test1": "test1"}, ""},
		{"no value", "test1=", map[string]string{"test1": ""}, ""},
		{"space key", "=test1", nil, "empty key"},
		{"no equal sign", "test1", map[string]string{"test1": ""}, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ConvertLabelsToMap(test.env)
			assert.Equal(t, test.expected, result)
			if err != nil {
				assert.Equal(t, err.Error(), test.errMessage)
			}
		})
	}
}
