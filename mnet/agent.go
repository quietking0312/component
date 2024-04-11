package mnet

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"net"
	"sync"
	"time"
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
}

type RouterIface interface {
	Route(msg *Msg, a AgentIface)
}

type Agent struct {
	Id        string
	conn      Conn
	log       Log
	parser    PackParser
	AuthFunc  AuthFunc // 第一个数据包调用该函数
	keys      map[string]any
	router    RouterIface
	timeout   time.Duration
	readChan  chan *Msg
	closeFlag bool
	closeChan chan struct{}
	mu        sync.RWMutex
}

type AuthFunc func(msg *Msg, a *Agent) (string, error)

func NewAgent(conn Conn, parser PackParser, router RouterIface) *Agent {
	return &Agent{
		Id:        "",
		conn:      conn,
		log:       _log,
		parser:    parser,
		keys:      make(map[string]any),
		router:    router,
		readChan:  make(chan *Msg),
		timeout:   20 * time.Minute,
		closeFlag: false,
		closeChan: make(chan struct{}, 1),
	}
}

func (a *Agent) SetLog(log Log) *Agent {
	a.log = log
	return a
}

func (a *Agent) SetTimeout(t time.Duration) *Agent {
	if t <= 0 {
		return a
	}
	a.timeout = t
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	closeChan := make(chan int8)
	go a.read(ctx, closeChan)
	ticker := time.NewTicker(a.timeout)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			go a.Close()
		case msg := <-a.readChan:
			ticker.Reset(a.timeout)
			a.router.Route(msg, a)
		case <-a.closeChan:
			return
		case <-closeChan:
			return
		}
	}
}

func (a *Agent) read(ctx context.Context, c chan int8) {
	defer func() {
		c <- 1
	}()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, msg, err := a.conn.Read()
			if err != nil {
				a.log.Error(fmt.Errorf("read message, %v", err))
				return
			}
			m, err := a.parser.Unmarshal(msg)
			if err != nil {
				a.log.Error(fmt.Errorf("unmarshal message, %v", err))
				return
			}
			a.readChan <- m
		}
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

func (a *Agent) Close() {
	if !a.closeFlag {
		a.closeFlag = true
		a.closeChan <- struct{}{}
	}
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
