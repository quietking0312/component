package mnet

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"sync"
)

type WSServer struct {
	Addr    string
	LocalIP string
	logger  Log

	NewAgent   func(*WSConn) AgentIface
	ln         net.Listener
	upgrader   websocket.Upgrader
	MaxConnNum int
	Auth       func(http.ResponseWriter, *http.Request) (string, error) // http 用户校验 为nil时调用 AgentIface 的用户校验
	conns      map[string]*WSConn
	mutexConns sync.Mutex
	wg         sync.WaitGroup
}

func NewWSServer(maxConnNum int, ag func(conn *WSConn) AgentIface) *WSServer {
	return &WSServer{
		NewAgent:   ag,
		MaxConnNum: maxConnNum,
		conns:      make(map[string]*WSConn),
		logger:     _log,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (ws *WSServer) SetLogger(log Log) {
	ws.logger = log
}

func (ws *WSServer) SetUpgrader(upgrader websocket.Upgrader) {
	ws.upgrader = upgrader
}

func (ws *WSServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var connId string
	var err error
	if ws.Auth != nil {
		connId, err = ws.Auth(w, r)
		if err != nil {
			ws.logger.Error(err)
			return
		}
	}

	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		ws.logger.Error(err)
		return
	}
	ws.wg.Add(1)
	defer ws.wg.Done()
	wsConn := newWSConn("", conn, ws.logger)
	ag := ws.NewAgent(wsConn)
	if connId == "" {
		connId, err = ag.Auth()
		if err != nil {
			ws.logger.Error(err)
			return
		}
	}
	wsConn.SetId(connId)
	ag.SetId(connId)

	if oldConn, ok := ws.conns[wsConn.GetId()]; ok {
		oldConn.Close()
	}
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
	delete(ws.conns, wsConn.GetId())
	ws.mutexConns.Unlock()
	ag.Close()
}

func (ws *WSServer) Serve(ln net.Listener) {
	ws.Addr = ln.Addr().String()
	ws.LocalIP = GetLocalIP()
	ws.ln = ln
	httpServer := &http.Server{
		Addr:    ws.Addr,
		Handler: ws,
	}
	ws.logger.Info(fmt.Sprintf("server start ip:%s addr: %s", ws.LocalIP, ws.Addr))
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
