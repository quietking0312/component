package mtcp

import (
	"bufio"
	"fmt"
	"net"
)

func server() {
	listen, err := net.Listen("tcp", "127.0.0.1:8000")
	if err != nil {
		return
	}
	defer listen.Close()
	conn, err := listen.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	read := bufio.NewReaderSize(conn, 4096)
	buf := make([]byte, 4096)
	n, err := read.Read(buf)
	if err != nil {
		return
	}
	conn.Write(buf[:n])
	fmt.Println(string(buf[:n]))
}
