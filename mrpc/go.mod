module github.com/quietking0312/component/mrpc

go 1.25.0

require (
	github.com/golang/protobuf v1.5.4
	github.com/quietking0312/component/mcyptos v0.0.0
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
)

replace github.com/quietking0312/component/mcyptos => ../mcyptos

exclude google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55
