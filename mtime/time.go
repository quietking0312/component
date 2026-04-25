package mtime

import (
	"math"
	"sync/atomic"
	"time"
)

// ========== 时间模板定义 ==========

const (
	// TimeTemplate1 标准日期时间格式
	TimeTemplate1 = "2006-01-02 15:04:05"
	// TimeTemplate2 斜杠分隔日期时间格式
	TimeTemplate2 = "2006/01/02 15:04:05"
	// TimeTemplate3 日期格式
	TimeTemplate3 = "2006-01-02"
	// TimeTemplate4 时间格式
	TimeTemplate4 = "15:04:05"
	// TimeTemplate5 紧凑日期格式
	TimeTemplate5 = "20060102"
	// TimeTemplate6 紧凑日期时间格式
	TimeTemplate6 = "20060102150405"
	// TimeTemplate7 小时精度格式
	TimeTemplate7 = "2006010215"
	// TimeTemplate8 年月格式
	TimeTemplate8 = "200601"

	// TimeLayoutDate 日期格式：2006-01-02
	TimeLayoutDate = "2006-01-02"
	// TimeLayoutDateTime 日期时间格式：2006-01-02 15:04:05
	TimeLayoutDateTime = "2006-01-02 15:04:05"
	// TimeLayoutDateTimeMs 带毫秒的日期时间格式：2006-01-02 15:04:05.000
	TimeLayoutDateTimeMs = "2006-01-02 15:04:05.000"
	// TimeLayoutCompactDate 紧凑日期格式：20060102
	TimeLayoutCompactDate = "20060102"
	// TimeLayoutCompactDateTime 紧凑日期时间格式：20060102150405
	TimeLayoutCompactDateTime = "20060102150405"
	// TimeLayoutISO8601 ISO8601格式：2006-01-02T15:04:05Z
	TimeLayoutISO8601 = time.RFC3339
	// TimeLayoutISO8601NoZone ISO8601无时区格式：2006-01-02T15:04:05
	TimeLayoutISO8601NoZone = "2006-01-02T15:04:05"
	// TimeLayoutRFC1123 RFC1123格式：Mon, 02 Jan 2006 15:04:05 MST
	TimeLayoutRFC1123 = time.RFC1123
)

// ========== Mock 时间源（用于调试） ==========

type clockSource struct {
	fn func() time.Time
}

var globalClock atomic.Pointer[clockSource]

// Now 获取当前时间（支持 Mock）
func Now() time.Time {
	if c := globalClock.Load(); c != nil && c.fn != nil {
		return c.fn()
	}
	return time.Now()
}

// Freeze 冻结时间到指定时刻，后续所有 Now() 调用都返回该时间
func Freeze(t time.Time) {
	globalClock.Store(&clockSource{fn: func() time.Time { return t }})
}

// Unfreeze 解除时间冻结，恢复真实时间
func Unfreeze() {
	globalClock.Store(nil)
}

// SetOffset 设置全局时间偏移量（在真实时间基础上偏移）
func SetOffset(offset time.Duration) {
	globalClock.Store(&clockSource{fn: func() time.Time { return time.Now().Add(offset) }})
}

// ========== 时间戳与字符串互转 ==========

// TimestampToStr 时间戳(秒)转字符串
func TimestampToStr(timestamp int64, layout string) string {
	return time.Unix(timestamp, 0).Format(layout)
}

// TimestampMsToStr 时间戳(毫秒)转字符串
func TimestampMsToStr(timestampMs int64, layout string) string {
	return time.UnixMilli(timestampMs).Format(layout)
}

// TimestampUsToStr 时间戳(微秒)转字符串
func TimestampUsToStr(timestampUs int64, layout string) string {
	return time.UnixMicro(timestampUs).Format(layout)
}

// StrToTimestamp 字符串转时间戳(秒)
func StrToTimestamp(timeStr string, layout string) (int64, error) {
	t, err := time.ParseInLocation(layout, timeStr, time.Local)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

// StrToTimestampMs 字符串转时间戳(毫秒)
func StrToTimestampMs(timeStr string, layout string) (int64, error) {
	t, err := time.ParseInLocation(layout, timeStr, time.Local)
	if err != nil {
		return 0, err
	}
	return t.UnixMilli(), nil
}

// StrToTimestampUs 字符串转时间戳(微秒)
func StrToTimestampUs(timeStr string, layout string) (int64, error) {
	t, err := time.ParseInLocation(layout, timeStr, time.Local)
	if err != nil {
		return 0, err
	}
	return t.UnixMicro(), nil
}

// ========== time.Time 与字符串互转 ==========

// TimeToStr 时间转字符串
func TimeToStr(t time.Time, layout string) string {
	return t.Format(layout)
}

// StrToTime 字符串转时间（本地时区）
func StrToTime(timeStr string, layout string) (time.Time, error) {
	return time.ParseInLocation(layout, timeStr, time.Local)
}

// StrToTimeUTC 字符串转UTC时间
func StrToTimeUTC(timeStr string, layout string) (time.Time, error) {
	return time.Parse(layout, timeStr)
}

// ========== 便捷函数 ==========

// NowTimestamp 当前时间戳(秒)
func NowTimestamp() int64 {
	return Now().Unix()
}

// NowTimestampMs 当前时间戳(毫秒)
func NowTimestampMs() int64 {
	return Now().UnixMilli()
}

// NowTimestampUs 当前时间戳(微秒)
func NowTimestampUs() int64 {
	return Now().UnixMicro()
}

// NowTimestampNs 当前时间戳(纳秒)
func NowTimestampNs() int64 {
	return Now().UnixNano()
}

// NowStr 当前时间格式化字符串
func NowStr(layout string) string {
	return Now().Format(layout)
}

// NowDateStr 当前日期字符串 (2006-01-02)
func NowDateStr() string {
	return Now().Format(TimeLayoutDate)
}

// NowDateTimeStr 当前日期时间字符串 (2006-01-02 15:04:05)
func NowDateTimeStr() string {
	return Now().Format(TimeLayoutDateTime)
}

// ========== 时间计算 ==========

// BeginOfDay 获取当天的开始时间
func BeginOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay 获取当天的结束时间
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// BeginOfWeek 获取当周的开始时间(周一)
func BeginOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	diff := time.Duration(weekday-1) * 24 * time.Hour
	return BeginOfDay(t.Add(-diff))
}

// BeginOfMonth 获取当月的开始时间
func BeginOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth 获取当月的结束时间
func EndOfMonth(t time.Time) time.Time {
	return BeginOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// AddDays 添加天数
func AddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

// AddMonths 添加月数
func AddMonths(t time.Time, months int) time.Time {
	return t.AddDate(0, months, 0)
}

// AddYears 添加年数
func AddYears(t time.Time, years int) time.Time {
	return t.AddDate(years, 0, 0)
}

// ========== 时间比较 ==========

// IsSameDay 是否为同一天
func IsSameDay(t1, t2 time.Time) bool {
	return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day()
}

// DaysBetween 计算两个时间相差的天数（向下取整，返回非负数）
func DaysBetween(t1, t2 time.Time) int {
	diff := t2.Sub(t1)
	return int(math.Abs(diff.Hours() / 24))
}

// IsLeapYear 是否为闰年
func IsLeapYear(year int) bool {
	return (year%4 == 0 && year%100 != 0) || (year%400 == 0)
}

// GetWeek 获取当前星期 (0=周日, 1=周一, ..., 6=周六)
func GetWeek(t time.Time) int {
	return int(t.Weekday())
}
