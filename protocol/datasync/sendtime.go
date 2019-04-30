package datasync

import "math"

func CalculateSendTime(count uint64, lastTime int64) int64 {
	return lastTime + int64(math.Pow(float64(count), 2.0))
}
