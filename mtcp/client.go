package mtcp

import (
	"fmt"
	"net"
)

func client() {
	conn, err := net.Dial("tcp", "127.0.0.1:8000")
	if err != nil {
		fmt.Println("")
		return
	}
	defer conn.Close()
	conn.Write([]byte("发起连接"))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}
	fmt.Println(string(buf[:n]))
}
