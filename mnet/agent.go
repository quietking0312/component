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
	Closed() bool
	Get(string) (any, bool)
	Set(key string, value any)
	SetId(id string)
	SetLog(log Log) AgentIface
	SetAuth(auth AuthFunc) AgentIface
	SetTimeout(t time.Duration) AgentIface
	SetCloseCallback(fc func()) AgentIface
	GetId() string
	RemoteAddr() net.Addr
}

type RouterIface interface {
	Route(msg *Msg, a AgentIface)
}

type Agent struct {
	Id            string
	conn          Conn
	log           Log
	parser        PackParser
	AuthFunc      AuthFunc // 第一个数据包调用该函数
	keys          map[string]any
	router        RouterIface
	timeout       time.Duration
	readChan      chan *Msg
	writeChan     chan []byte
	closeFlag     bool
	closeCallback func() // 链接 关闭 回调
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
}

type AuthFunc func(msg *Msg, a *Agent) (string, error)

// NewAgent
// conn 链接
// parser 包解析
// router 路由
func NewAgent(conn Conn, parser PackParser, router RouterIface) *Agent {
	ctx, cancel := context.WithCancel(context.Background())
	return &Agent{
		Id:        "",
		conn:      conn,
		log:       _log,
		parser:    parser,
		keys:      make(map[string]any),
		router:    router,
		readChan:  make(chan *Msg),
		writeChan: make(chan []byte, 1024),
		timeout:   20 * time.Minute,
		closeFlag: false,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (a *Agent) SetLog(log Log) AgentIface {
	a.log = log
	return a
}

func (a *Agent) SetTimeout(t time.Duration) AgentIface {
	if t <= 0 {
		return a
	}
	a.timeout = t
	return a
}

func (a *Agent) SetAuth(auth AuthFunc) AgentIface {
	a.AuthFunc = auth
	return a
}

func (a *Agent) SetCloseCallback(fc func()) AgentIface {
	a.closeCallback = fc
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
	defer a.cancel()
	go a.read()
	ticker := time.NewTicker(a.timeout)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C: // 超时
			go a.Close()
		case data := <-a.writeChan: // 往真实链接写入数据
			_, err := a.conn.Write(data)
			if err != nil {
				a.log.Error(fmt.Errorf("write message, %v", err))
				go a.Close()
			}
		case msg := <-a.readChan:
			ticker.Reset(a.timeout)
			a.router.Route(msg, a)
		case <-a.ctx.Done(): // 读取协程 关闭
			go a.Close()
			return
		}
	}
}

func (a *Agent) read() {
	defer a.cancel()
	for {
		select {
		case <-a.ctx.Done():
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
			select {
			case a.readChan <- m:
			case <-a.ctx.Done():
				return
			}
		}
	}
}

func (a *Agent) Write(msg any) {
	data, err := a.parser.Marshal(msg)
	if err != nil {
		a.log.Error(fmt.Errorf("parser.Marshal, %v", err))
	}
	a.writeChan <- data
}

func (a *Agent) Close() {
	if !a.closeFlag {
		a.closeFlag = true
		a.conn.Close()
		a.cancel()
		if a.closeCallback != nil {
			a.closeCallback()
		}
	}
}

func (a *Agent) Closed() bool {
	return a.closeFlag
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
