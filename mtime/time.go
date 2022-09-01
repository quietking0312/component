package mtime

import "time"

const (
	// TimeTemplate1 时间转换的模板，golang里面只能是 "2006-01-02 15:04:05" （go的诞生时间）
	TimeTemplate1 = "2006-01-02 15:04:05"
	TimeTemplate2 = "2006/01/02 15:04:05"
	TimeTemplate3 = "2006-01-02"
	TimeTemplate4 = "15:04:05"
	TimeTemplate5 = "20060102"
	TimeTemplate6 = "20060102150405"
	TimeTemplate7 = "2006010215"
	TimeTemplate8 = "200601"
)

type TimeOpts struct {
	Years  int
	Months int
	Days   int
}

func defaultOpts() *TimeOpts {
	return &TimeOpts{
		Years:  0,
		Months: 0,
		Days:   0,
	}
}

func GetTime(opts ...func(opt *TimeOpts)) int64 {
	opt := defaultOpts()
	for _, o := range opts {
		o(opt)
	}
	var timeObj = time.Now()
	timeObj = timeObj.AddDate(opt.Years, opt.Months, opt.Days)
	return timeObj.Unix()
}

// GetWeek 获取当前星期
// 0, 1, 2, 3, 4, 5, 6
func GetWeek(opts ...func(opt *TimeOpts)) int {
	opt := defaultOpts()
	for _, o := range opts {
		o(opt)
	}
	var timeObj = time.Now()
	timeObj = timeObj.AddDate(opt.Years, opt.Months, opt.Days)
	return int(timeObj.Weekday())
}

func IntToString(t int64, layout string) string {

	return time.Unix(t, 0).Format(layout)
}

func StringToInt(t string, layout string, location *time.Location) int64 {
	if location == nil {
		location = time.Local
	}
	stamp, _ := time.ParseInLocation(layout, t, location)
	return stamp.Unix()
}
