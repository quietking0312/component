package mnet

import (
	"fmt"
	"github.com/gorilla/websocket"
)

type WSConn struct {
	Id        string
	conn      *websocket.Conn
	closeFlag bool
	log       Log
}

func newWSConn(id string, conn *websocket.Conn, log Log) *WSConn {
	wsConn := new(WSConn)
	wsConn.Id = id
	wsConn.conn = conn
	wsConn.closeFlag = false
	wsConn.log = log

	return wsConn
}

func (ws *WSConn) SetId(id string) {
	ws.Id = id
}

func (ws *WSConn) Close() error {
	if ws.closeFlag {
		return nil
	}
	ws.closeFlag = true
	ws.conn.Close()
	return nil
}

func (ws *WSConn) Read() (int, []byte, error) {
	_, p, err := ws.conn.ReadMessage()
	return len(p), p, err
}

func (ws *WSConn) Write(b []byte) (int, error) {
	if ws.closeFlag || b == nil {
		return 0, nil
	}
	err := ws.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		ws.log.Error(fmt.Errorf("id:%s write error %+v", ws.Id, err))
		ws.Close()
		return 0, err
	}
	return len(b), nil
}
