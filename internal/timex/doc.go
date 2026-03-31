package timex

import (
	"sync"
	"time"
)

// 时间相关函数

var localOnce sync.Once
var local *time.Location

func InitLocation(location string) {
	if local == nil {
		local, _ = time.LoadLocation(location)
	}
	time.Local = local
}

func getLocal() *time.Location {
	localOnce.Do(func() {
		if local == nil {
			local = time.Local
		}
	})
	return local
}

func UnixTime() int64 {
	t := time.Now().In(getLocal())
	return t.Unix()
}

func Now() time.Time {
	return time.Now().In(getLocal())
}
