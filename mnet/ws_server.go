package mnet

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"sync"
	"time"
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
	conns      map[string]AgentIface
	mutexConns sync.Mutex
	wg         sync.WaitGroup
	closeFlag  chan int16
}

func NewWSServer(maxConnNum int, ag func(conn *WSConn) AgentIface) *WSServer {
	return &WSServer{
		NewAgent:   ag,
		MaxConnNum: maxConnNum,
		conns:      make(map[string]AgentIface),
		logger:     _log,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		closeFlag: make(chan int16),
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

	if oldConn, ok := ws.conns[ag.GetId()]; ok {
		oldConn.Close()
	}
	if len(ws.conns) >= ws.MaxConnNum {
		ag.Close()
		ws.logger.Error(fmt.Errorf("conn num:%d >= maxConn:%d", len(ws.conns), ws.MaxConnNum))
		return
	}
	ws.mutexConns.Lock()
	ws.conns[wsConn.Id] = ag
	ws.mutexConns.Unlock()

	ag.Run()
	ag.Close()
	ws.mutexConns.Lock()
	delete(ws.conns, ag.GetId())
	ws.mutexConns.Unlock()
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
	go func() {
		httpServer.Serve(ln)
	}()
	select {
	case n := <-ws.closeFlag:
		ws.logger.Info(fmt.Sprintf("ws server shutting down the server..."))
		ctx := context.Background()
		var cancel context.CancelFunc
		if n > 0 {
			ctx, cancel = context.WithTimeout(context.Background(), time.Duration(n)*time.Second)
			defer cancel()
		}
		_ = httpServer.Shutdown(ctx)
		go ws.Close()

	}
}

func (ws *WSServer) Shutdown() {
	ws.closeFlag <- 20
	x := 500 * time.Millisecond
	ticker := time.NewTicker(x)
	defer ticker.Stop()
	for {
		if len(ws.conns) == 0 {
			break
		}
		ws.logger.Info(fmt.Sprintf("%d links remaining", len(ws.conns)))
		select {
		case <-ticker.C:
			if x.Seconds() > (time.Duration(20) * time.Second).Seconds() {
				return
			}
			x += x
		}
	}
	ws.logger.Info(fmt.Sprintf("ws server has been shut down gracefully"))
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
