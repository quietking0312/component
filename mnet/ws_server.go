package mnet

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"sync"
)

type WSServer struct {
	Addr       string
	logger     Log
	NewAgent   func(*WSConn) Agent
	ln         net.Listener
	upgrader   websocket.Upgrader
	MaxConnNum int

	conns      map[string]*WSConn
	mutexConns sync.Mutex
	wg         sync.WaitGroup
}

func NewWSServer(maxConnNum int, ag func(conn *WSConn) Agent) *WSServer {
	return &WSServer{
		NewAgent:   ag,
		MaxConnNum: maxConnNum,
		conns:      make(map[string]*WSConn),
		logger:     _log,
	}
}

func (ws *WSServer) SetLogger(log Log) {
	ws.logger = log
}

func (ws *WSServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(65535)
	ws.wg.Add(1)
	defer ws.wg.Done()

	wsConn := newWSConn("", conn, ws.logger)
	if oldConn, ok := ws.conns[wsConn.Id]; ok {
		oldConn.Close()
	}
	ag := ws.NewAgent(wsConn)

	connId, err := ag.Auth()
	if err != nil {
		return
	}
	wsConn.SetId(connId)
	if len(ws.conns) >= ws.MaxConnNum {
		wsConn.Close()
		ws.logger.Error(fmt.Errorf("conn num:%d >= maxConn:%d", len(ws.conns), ws.MaxConnNum))
		return
	}
	ws.mutexConns.Lock()
	ws.conns[wsConn.Id] = wsConn
	ws.mutexConns.Unlock()

	ag.Run()

	wsConn.Close()
	ws.mutexConns.Lock()
	delete(ws.conns, wsConn.Id)
	ws.mutexConns.Unlock()
	ag.Close()
}

func (ws *WSServer) Serve(ln net.Listener) {
	ws.Addr = ln.Addr().String()
	ws.ln = ln
	httpServer := &http.Server{
		Addr:    ws.Addr,
		Handler: ws,
	}
	httpServer.Serve(ln)
}

func (ws *WSServer) Close() {
	ws.ln.Close()
	ws.mutexConns.Lock()
	for _, conn := range ws.conns {
		conn.Close()
	}
	ws.mutexConns.Unlock()
	ws.wg.Wait()
}
