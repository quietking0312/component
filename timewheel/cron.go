package timewheel

import "github.com/robfig/cron/v3"

var defaultCron *cron.Cron

func NewCron() *cron.Cron {
	c := cron.New()
	return c
}

func NewDefaultCron() {
	defaultCron = NewCron()
}

func GetDefaultCron() *cron.Cron {
	return defaultCron
}
