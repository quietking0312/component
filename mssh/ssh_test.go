package mssh

import (
	"fmt"
	"github.com/quietking0312/component/mbar"
	"os"
	"path"
	"testing"
)

func TestCli_Connect(t *testing.T) {
	cli := &Cli{
		User: "ubuntu",
		Pwd:  "",
		Addr: "",
	}
	if _, err := cli.Connect(); err != nil {
		t.Fatal(err)
	}
	result, err := cli.Run("ls -l")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(result)
	defer cli.client.Close()
	if err := cli.UploadFile("ssh.go", "/home/ubuntu"); err != nil {
		t.Fatal(err)
	}
	result, err = cli.Run("ls -l")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(result)

	if err := cli.DownloadFile("/home/ubuntu/ssh.go", "./", "temp.text"); err != nil {
		t.Fatal(err)
	}
}

func TestCli_UploadFileAndProgress(t *testing.T) {
	cli := &Cli{
		User: "ubuntu",
		Pwd:  "#nmp3?c;G+L!Wy2R",
		Addr: "152.136.171.104:22",
	}
	if _, err := cli.Connect(); err != nil {
		t.Fatal(err)
	}
	defer cli.client.Close()

	result, err := cli.Run("ls -l")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(result)

	srcFile, err := os.Open("D:\\myprojects\\ops\\ops.conf")
	if err != nil {
		fmt.Println("open:", err)
		t.Fatal(err)
	}
	s, _ := srcFile.Stat()
	fmt.Println("totalSize: ", s.Size())
	defer srcFile.Close()

	b := mbar.NewBar(int(s.Size()))

	ch := make(chan int64, 1000)
	go func() {
		if err := cli.UploadFileAndProgress(srcFile, path.Join("/data", path.Base("hello.conf")), ch); err != nil {
			t.Error(err)
			return
		}
	}()
	for {
		select {
		case s := <-ch:
			fmt.Println(s)
			b.Add(int(s))
		}
	}
}
