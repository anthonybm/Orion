package util

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

var cocoaUnixDelta int64 = 978307200

// webkit timestamps use Jan 1, 1601 as epoch start
// unix timestamps start Jan 1, 1970
// this constant represents the difference in nanoseconds
const chromeEpoch = 11644473600000000
const msToS = 1000000

// CocoaTime converts cocoa webkit DB timestamp to ISO8601, UTC format
func CocoaTime(seconds int64) (string, error) {
	if seconds != 0 {
		return time.Unix(seconds+cocoaUnixDelta, 0).UTC().Format(time.RFC3339), nil
	}
	return "", errors.New("given time was nil or zero")
}

// ChromeTime converts webkit/chrome microsecond timestamps to UTC strings of ISO8601 format
func ChromeTime(chrometime interface{}) (string, error) {
	switch v := chrometime.(type) {
	case string:
		if v != "" && v != "0" {
			t, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return "ERROR", err
			}
			return time.Unix(int64(t-chromeEpoch)/msToS, int64(t%msToS)).UTC().Format(time.RFC3339), nil
		} else if v == "" || v == "0" {
			return "", nil
		}
	case uint64:
		if v != 0 {
			return time.Unix(int64(v-chromeEpoch)/msToS, int64(v%msToS)).UTC().Format(time.RFC3339), nil
		} else if v == 0 {
			return "", nil
		}
	}

	return "", errors.New("could not parse chrome time " + fmt.Sprint(chrometime))
}
