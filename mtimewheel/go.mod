module github.com/quietking0312/component/mtimewheel

go 1.25.0

require (
	github.com/quietking0312/component/mlog v0.0.0
	github.com/stretchr/testify v1.11.1
	go.uber.org/zap v1.27.1
)

replace github.com/quietking0312/component/mlog => ../mlog

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
