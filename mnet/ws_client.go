package mnet

import (
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

type WSClient struct {
	url       string
	header    http.Header
	dialer    websocket.Dialer
	closeFlag bool
	logger    Log
}

func NewWSClient(uri string) *WSClient {
	return &WSClient{
		url:       uri,
		closeFlag: false,
		logger:    _log,
	}
}

func (cli *WSClient) Dial() *websocket.Conn {
	for {
		conn, _, err := cli.dialer.Dial(cli.url, cli.header)
		if err == nil || cli.closeFlag {
			return conn
		}

		time.Sleep(5 * time.Second)
	}
}

func (cli *WSClient) Close() error {
	if cli.closeFlag {
		return nil
	}
	return nil
}
