package date

import "time"

// TODO make it possibe to configure
// I dont consider "localtime by default" to be sane decision
var tz = time.UTC

func NowTime() time.Time {
	return time.Now().In(tz)
}

func NowTimeUTC() time.Time {
	return time.Now().UTC()
}

func NowTimeUnix() int64 {
	return time.Now().Unix()
}

func NowTimeUnixMilli() int64 {
	t := time.Now()
	u := t.Unix()
	un := t.UnixNano()
	return (u * 1000) + (un/int64(time.Millisecond))%1000
}

func UnixTime(u int64) time.Time {
	return time.Unix(u, 0).In(tz)
}

func UnixTimeUTC(u int64) time.Time {
	return time.Unix(u, 0).UTC()
}
