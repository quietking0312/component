package mnet

import (
	"github.com/xtaci/kcp-go"
	"sync"
)

type KCPServer struct {
	ln         *kcp.Listener
	logger     Log
	block      kcp.BlockCrypt
	NewAgent   func(conn *KCPConn) AgentIface
	maxConnNum int
	conns      map[string]*KCPConn
	mu         sync.Mutex
	wg         sync.WaitGroup
}

func NewKCPServer(maxConnNum int, ag func(conn *KCPConn) AgentIface) *KCPServer {
	return &KCPServer{
		NewAgent:   ag,
		maxConnNum: maxConnNum,
		conns:      make(map[string]*KCPConn),
		logger:     _log,
	}
}

func (k *KCPServer) SetLogger(log Log) {
	k.logger = log
}

func (k *KCPServer) Serve(addr string) error {
	ln, err := kcp.ListenWithOptions(addr, k.block, 10, 3)
	_ = err
	k.ln = ln
	for {
		conn, err := k.ln.AcceptKCP()
		if err != nil {
			return err
		}
		k.mu.Lock()
		if len(k.conns) >= k.maxConnNum {
			k.mu.Unlock()
			conn.Close()
			continue
		}
		kcpConn := newKCPConn("", conn, k.logger)
		ag := k.NewAgent(kcpConn)
		connId, err := ag.Auth()
		if err != nil {
			k.mu.Unlock()
			conn.Close()
			continue
		}
		if oldConn, ok := k.conns[kcpConn.Id]; ok {
			oldConn.Close()
		}
		kcpConn.SetId(connId)
		k.conns[connId] = kcpConn
		k.mu.Unlock()
		go func() {
			ag.Run()
			kcpConn.Close()
			k.mu.Lock()
			delete(k.conns, kcpConn.Id)
			k.mu.Unlock()
		}()

	}
}
