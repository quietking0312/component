package mnet

import (
	"net"
	"sync"
)

type TCPServer struct {
	ln         net.Listener
	logger     Log
	checkAuth  func()
	NewAgent   func(conn *TCPConn) AgentIface
	conns      map[string]*TCPConn
	maxConnNum int
	mu         sync.Mutex
	wg         sync.WaitGroup
}

func NewTCPServer(maxConnNum int, ag func(conn *TCPConn) AgentIface) *TCPServer {
	return &TCPServer{
		NewAgent:   ag,
		maxConnNum: maxConnNum,
		conns:      make(map[string]*TCPConn),
		logger:     _log,
	}
}

func (t *TCPServer) SetLogger(log Log) {
	t.logger = log
}

func (t *TCPServer) Serve(ln net.Listener) {
	t.ln = ln
	for {
		conn, err := t.ln.Accept()
		if err != nil {
			return
		}
		t.mu.Lock()
		if len(t.conns) >= t.maxConnNum {
			t.mu.Unlock()
			conn.Close()
			continue
		}
		tcpConn := newTCPConn("", conn, t.logger)
		ag := t.NewAgent(tcpConn)
		connId, err := ag.Auth()
		if err != nil {
			t.mu.Unlock()
			conn.Close()
			continue
		}
		if oldConn, ok := t.conns[tcpConn.Id]; ok {
			oldConn.Close()
		}
		tcpConn.SetId(connId)
		t.conns[connId] = tcpConn
		t.mu.Unlock()
		go func() {
			ag.Run()
			tcpConn.Close()
			t.mu.Lock()
			delete(t.conns, tcpConn.Id)
			t.mu.Unlock()
		}()
	}
}
