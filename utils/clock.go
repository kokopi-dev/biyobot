package utils

import "time"

func JapanTimeNow() time.Time {
	now := time.Now().In(time.FixedZone("JST", 9*60*60)) // UTC+9
	return now
}
