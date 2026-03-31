package utils

import "time"

// 2026-01-14 18:56:13 转 时间戳
func UnixTimeFromStr(str string, loc ...string) int64 {
	if len(loc) == 0 {
		loc = append(loc, "Asia/Shanghai")
	}
	location, _ := time.LoadLocation(loc[0])
	t, _ := time.ParseInLocation("2006-01-02 15:04:05", str, location)
	return t.Unix()
}

func TimeFromStr(str string, loc ...string) time.Time {
	if len(loc) == 0 {
		loc = append(loc, "Asia/Shanghai")
	}
	location, _ := time.LoadLocation(loc[0])
	t, _ := time.ParseInLocation("2006-01-02 15:04:05", str, location)
	return t
}

func UnixTime(ts ...int64) int64 {
	location, _ := time.LoadLocation("Asia/Shanghai")
	if len(ts) == 0 {
		t := time.Now().In(location)
		return t.Unix()
	}
	t := time.Unix(ts[0], 0).In(location)
	return t.Unix()
}

func TimeL(loc *time.Location, ts ...int64) time.Time {
	if loc == nil {
		loc, _ = time.LoadLocation("Asia/Shanghai")
	}
	if len(ts) == 0 {
		t := time.Now().In(loc)
		return t
	}
	t := time.Unix(ts[0], 0).In(loc)
	return t
}

func Day(t time.Time) (int, int) {
	year, month, day := t.Date()
	start := time.Date(year, month, day, 0, 0, 0, 0, t.Location())
	end := time.Date(year, month, day, 23, 59, 59, 999, t.Location())
	return int(start.Unix()), int(end.Unix())
}

func Month(t time.Time) (int, int) {
	year, month, _ := t.Date()
	start := time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
	if month == 12 {
		month = 1
		year = year + 1
	}
	end := time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
	return int(start.Unix()), int(end.Unix()) - 1
}

func UnixYear(ts int64) (int, int) {
	t := time.Unix(int64(ts), 0)
	year, _, _ := t.Date()
	start := time.Date(year, 1, 1, 0, 0, 0, 0, t.Location())
	end := time.Date(year+1, 1, 1, 0, 0, 0, 0, t.Location())
	return int(start.Unix()), int(end.Unix()) - 1
}

func UnixYearStart(t int64) int {
	ti := time.Unix(t, 0)
	year, _, _ := ti.Date()
	start := time.Date(year, 1, 1, 0, 0, 0, 0, ti.Location())
	return int(start.Unix())
}

func UnixYearEnd(t int64) int {
	ti := time.Unix(t, 0)
	year, _, _ := ti.Date()
	end := time.Date(year, 12, 31, 23, 59, 59, 0, ti.Location())
	return int(end.Unix())
}
