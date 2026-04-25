module github.com/quietking0312/component/msock

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/quietking0312/component/mlog v0.0.0
	github.com/stretchr/testify v1.11.1
	github.com/xtaci/kcp-go/v5 v5.6.72
)

replace github.com/quietking0312/component/mlog => ../mlog

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/klauspost/reedsolomon v1.12.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

exclude (
	google.golang.org/genproto v0.0.0-20180817151627-c66870c02cf8
	google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55
)
