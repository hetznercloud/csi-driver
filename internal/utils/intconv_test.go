package utils

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUint64ToInt64(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		val, err := UInt64ToInt64(42)
		require.NoError(t, err)
		assert.Equal(t, int64(42), val)
	})

	t.Run("overflows int64", func(t *testing.T) {
		var input uint64 = math.MaxInt64 + 1
		_, err := UInt64ToInt64(input)
		assert.EqualError(t, err, fmt.Sprintf("value %d overflows int64", input))
	})
}
