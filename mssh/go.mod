module github.com/quietking0312/component/mssh

go 1.25.0

require (
	github.com/pkg/sftp v1.13.10
	github.com/quietking0312/component/mbar v0.0.0
	golang.org/x/crypto v0.50.0
)

require (
	github.com/kr/fs v0.1.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
)

replace github.com/quietking0312/component/mbar => ../mbar
