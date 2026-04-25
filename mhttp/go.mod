module github.com/quietking0312/component/mhttp

go 1.25.0

require (
	github.com/quietking0312/component/mlog v0.0.0
	go.uber.org/zap v1.27.1
	golang.org/x/net v0.53.0
)

require (
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)

replace github.com/quietking0312/component/mlog => ../mlog
