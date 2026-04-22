package exptime

import (
	"strconv"
	"time"
)

func GetExpTime() string {
	return strconv.FormatInt(time.Now().Unix()+1800, 10)
}

func IsExpired(unixTimeStr string) bool {
	unixTime, err := strconv.ParseInt(unixTimeStr, 10, 64)

	if err != nil {
		return false
	}

	return time.Now().Unix() >= unixTime
}
