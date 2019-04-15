package protocol

import "time"

// CalcMessageClock calculates a new clock value for Message.
// It is used to properly sort messages and accomodate the fact
// that time might be different on each device.
func CalcMessageClock(lastObservedValue int64, timeInMs TimestampInMs) int64 {
	clock := lastObservedValue
	if clock < int64(timeInMs) {
		// Added time should be larger than time skew tollerance for a message.
		// Here, we use 5 minutes which is much larger
		// than accepted message time skew by Whisper.
		clock = int64(timeInMs) + int64(5*time.Minute/time.Millisecond)
	} else {
		clock++
	}
	return clock
}
