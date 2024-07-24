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

var _defaultOpts = defaultOpts()

func SetDefaultOpt(opt *TimeOpts) {
	_defaultOpts = opt
}

func GetDefaultOpts() *TimeOpts {
	return _defaultOpts
}

type TimeOpts struct {
	Years    int
	Months   int
	Days     int
	Duration time.Duration
}

func defaultOpts() *TimeOpts {
	return &TimeOpts{
		Years:    0,
		Months:   0,
		Days:     0,
		Duration: 0,
	}
}

func GetTime(opts ...func(opt *TimeOpts)) time.Time {
	opt := new(TimeOpts)
	*opt = *_defaultOpts
	for _, o := range opts {
		o(opt)
	}
	var timeObj = time.Now()
	timeObj = timeObj.AddDate(opt.Years, opt.Months, opt.Days).Add(opt.Duration)
	return timeObj
}

// GetWeek 获取当前星期
// 0, 1, 2, 3, 4, 5, 6
func GetWeek(opts ...func(opt *TimeOpts)) int {
	opt := new(TimeOpts)
	*opt = *_defaultOpts
	for _, o := range opts {
		o(opt)
	}
	var timeObj = time.Now()
	timeObj = timeObj.AddDate(opt.Years, opt.Months, opt.Days).Add(opt.Duration)
	return int(timeObj.Weekday())
}

func GetDayTime(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func GetWeekTime(t time.Time) time.Time {
	weekDay := t.Weekday()
	if weekDay > 0 {
		weekDay -= 1
	} else {
		weekDay = 6
	}
	monday := t.AddDate(0, 0, -int(weekDay))
	return GetDayTime(monday)
}

func TimeStampToTime(t int64) time.Time {

	return time.Unix(t, 0)
}

func StringToTime(t string, layout string, location *time.Location) time.Time {
	if location == nil {
		location = time.Local
	}
	stamp, _ := time.ParseInLocation(layout, t, location)
	return stamp
}
