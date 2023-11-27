package mnet

import (
	"fmt"
	"github.com/google/uuid"
	"net"
	"sync"
)

type PackParser interface {
	Unmarshal([]byte) (msg *Msg, err error)
	Marshal(data any) ([]byte, error)
}

type AgentIface interface {
	Run()
	Auth() (string, error)
	Write(any)
	LocalAddr() net.Addr
	Close()
	Get(string) (any, bool)
	Set(key string, value any)
	SetId(id string)
	GetId() string
	RemoteAddr() net.Addr
	Next()
	Abort()
}

type RouterIface interface {
	Route(msg *Msg, a AgentIface)
	Next()
	Abort()
}

type Agent struct {
	Id         string
	conn       Conn
	log        Log
	parser     PackParser
	AuthFunc   AuthFunc // 第一个数据包调用该函数
	keys       map[string]any
	routerPool sync.Pool
	router     RouterIface
	mu         sync.RWMutex
}

type AuthFunc func(msg *Msg, a *Agent) (string, error)

func NewAgent(conn Conn, parser PackParser, router func() RouterIface) *Agent {
	return &Agent{
		Id:     "",
		conn:   conn,
		log:    _log,
		parser: parser,
		keys:   make(map[string]any),
		routerPool: sync.Pool{
			New: func() any {
				return router()
			},
		},
	}
}

func (a *Agent) SetLog(log Log) *Agent {
	a.log = log
	return a
}

func (a *Agent) SetAuth(auth AuthFunc) *Agent {
	a.AuthFunc = auth
	return a
}

func (a *Agent) Auth() (string, error) {
	if a.AuthFunc != nil {
		_, msg, err := a.conn.Read()
		if err != nil {
			a.log.Error(fmt.Errorf("read message, %v", err))
			return "", err
		}
		m, err := a.parser.Unmarshal(msg)
		if err != nil {
			a.log.Error(fmt.Errorf("unmarshal message, %v", err))
			return "", err
		}
		id, err := a.AuthFunc(m, a)
		a.Id = id
		return id, err
	}
	a.Id = uuid.New().String()
	return a.Id, nil
}

func (a *Agent) SetId(id string) {
	a.Id = id
}

func (a *Agent) GetId() string {
	return a.Id
}

func (a *Agent) Run() {
	for {
		_, msg, err := a.conn.Read()
		if err != nil {
			a.log.Error(fmt.Errorf("read message, %v", err))
			break
		}
		m, err := a.parser.Unmarshal(msg)
		if err != nil {
			a.log.Error(fmt.Errorf("unmarshal message, %v", err))
			break
		}
		a.Route(m)
	}
}

func (a *Agent) Write(msg any) {
	data, err := a.parser.Marshal(msg)
	if err != nil {
		a.log.Error(fmt.Errorf("parser.Marshal, %v", err))
	}
	_, err = a.conn.Write(data)
	if err != nil {
		a.log.Error(fmt.Errorf("write message, %v", err))
	}
}

func (a *Agent) Route(msg *Msg) {
	handler := a.routerPool.Get().(RouterIface)
	a.router = handler
	handler.Route(msg, a)
	a.router = nil
	a.routerPool.Put(handler)
}

func (a *Agent) Next() {
	if a.router != nil {
		a.router.Next()
	}
}

func (a *Agent) Abort() {
	if a.router != nil {
		a.router.Abort()
	}
}

func (a *Agent) Close() {
	a.conn.Close()
}

func (a *Agent) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}

func (a *Agent) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

func (a *Agent) Get(key string) (value any, exists bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	value, exists = a.keys[key]
	return
}

func (a *Agent) Set(key string, value any) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.keys == nil {
		a.keys = make(map[string]any)
	}
	a.keys[key] = value
}

func (a *Agent) Del(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.keys, key)
}
