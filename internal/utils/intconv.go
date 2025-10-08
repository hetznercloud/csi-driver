package utils

import (
	"fmt"
	"math"
)

func UInt64ToInt64(u uint64) (int64, error) {
	if u > math.MaxInt64 {
		return 0, fmt.Errorf("value %d overflows int64", u)
	}
	return int64(u), nil
}
