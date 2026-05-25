package mssh

import (
	"fmt"
	"testing"
)

// 以下测试需要真实 SSH 服务器，填入实际地址和密码后运行。

var testCfg = &Config{
	User:     "ubuntu",
	Password: "",
	Addr:     "127.0.0.1:22",
}

func TestRun(t *testing.T) {
	cli, err := New(testCfg)
	if err != nil {
		t.Skip("no ssh server:", err)
	}
	defer cli.Close()

	out, err := cli.Run("uname -a && ls -lh /tmp")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(out)
}

// TestShell 开启交互式 Shell，支持 vim/top 等 PTY 程序。
// 本地终端会切换为 raw 模式，退出后自动恢复。
func TestShell(t *testing.T) {
	cli, err := New(testCfg)
	if err != nil {
		t.Skip("no ssh server:", err)
	}
	defer cli.Close()

	if err := cli.Shell(nil); err != nil {
		t.Fatal(err)
	}
}

func TestUpload(t *testing.T) {
	cli, err := New(testCfg)
	if err != nil {
		t.Skip("no ssh server:", err)
	}
	defer cli.Close()

	err = cli.Upload("ssh.go", "/tmp/", &UploadOption{
		OnProgress: func(written int64) {
			fmt.Printf("uploaded %d bytes\n", written)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("upload done")
}

func TestDownload(t *testing.T) {
	cli, err := New(testCfg)
	if err != nil {
		t.Skip("no ssh server:", err)
	}
	defer cli.Close()

	err = cli.Download("/tmp/ssh.go", "./", &UploadOption{
		OnProgress: func(written int64) {
			fmt.Printf("downloaded %d bytes\n", written)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("download done")
}

func TestReadDir(t *testing.T) {
	cli, err := New(testCfg)
	if err != nil {
		t.Skip("no ssh server:", err)
	}
	defer cli.Close()

	entries, err := cli.ReadDir("/tmp")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		fmt.Printf("%s\t%d\t%s\n", e.Mode(), e.Size(), e.Name())
	}
}
