package msock

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewServer 测试服务器创建
func TestNewServer(t *testing.T) {
	tests := []struct {
		name    string
		opts    []ServerOption
		wantErr bool
	}{
		{
			name: "default server",
			opts: nil,
		},
		{
			name: "with custom address",
			opts: []ServerOption{WithAddress(":0")},
		},
		{
			name: "with custom codec",
			opts: []ServerOption{WithCodec(NewTLVCodec())},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewServer(tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, server)
		})
	}
}

// TestServer_TCP 测试TCP服务器
func TestServer_TCP(t *testing.T) {
	router := NewRouter()
	received := make(chan Message, 10)

	router.Register(1, func(conn Conn, msg Message) {
		received <- msg
		reply := NewMessage(2, []byte("reply"))
		conn.Send(reply)
	})

	server, err := NewServer(
		WithAddress(":0"),
		WithConnType(ConnTypeTCP),
		WithCodec(NewSimpleCodec()),
	)
	assert.NoError(t, err)
	server.SetRouter(router)

	// 启动服务器
	go server.Run()
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// 获取实际地址
	addr := server.listener.Addr().String()

	// 创建客户端
	clientRouter := NewRouter()
	clientReceived := make(chan Message, 10)
	clientRouter.Register(2, func(conn Conn, msg Message) {
		clientReceived <- msg
	})

	client := NewClient(
		WithConnType(ConnTypeTCP),
		WithCodec(NewSimpleCodec()),
	)
	client.SetRouter(clientRouter)

	err = client.Connect(addr)
	assert.NoError(t, err)
	defer client.Close()

	time.Sleep(50 * time.Millisecond)

	// 发送消息
	msg := NewMessage(1, []byte("hello"))
	err = client.Send(msg)
	assert.NoError(t, err)

	// 验证服务器收到消息
	select {
	case m := <-received:
		assert.Equal(t, uint32(1), m.RouteID())
		assert.Equal(t, "hello", string(m.Data()))
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to receive message")
	}

	// 验证客户端收到回复
	select {
	case m := <-clientReceived:
		assert.Equal(t, uint32(2), m.RouteID())
		assert.Equal(t, "reply", string(m.Data()))
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for client to receive reply")
	}
}

// TestServer_GWS 测试gws服务器
func TestServer_GWS(t *testing.T) {
	router := NewRouter()
	received := make(chan Message, 10)

	router.Register(1, func(conn Conn, msg Message) {
		received <- msg
		reply := NewMessage(2, []byte("reply"))
		conn.Send(reply)
	})

	server, err := NewServer(
		WithAddress(":0"),
		WithConnType(ConnTypeGWS),
		WithCodec(NewSimpleCodec()),
	)
	assert.NoError(t, err)
	server.SetRouter(router)

	// 启动服务器
	go server.Run()
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// 获取实际地址
	addr := server.listener.Addr().String()

	// 创建客户端
	clientRouter := NewRouter()
	clientReceived := make(chan Message, 10)
	clientRouter.Register(2, func(conn Conn, msg Message) {
		clientReceived <- msg
	})

	client := NewClient(
		WithConnType(ConnTypeGWS),
		WithCodec(NewSimpleCodec()),
	)
	client.SetRouter(clientRouter)

	err = client.Connect("ws://" + addr + "/ws")
	assert.NoError(t, err)
	defer client.Close()

	time.Sleep(50 * time.Millisecond)

	// 发送消息
	msg := NewMessage(1, []byte("hello"))
	err = client.Send(msg)
	assert.NoError(t, err)

	// 验证服务器收到消息
	select {
	case m := <-received:
		assert.Equal(t, uint32(1), m.RouteID())
		assert.Equal(t, "hello", string(m.Data()))
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to receive message")
	}

	// 验证客户端收到回复
	select {
	case m := <-clientReceived:
		assert.Equal(t, uint32(2), m.RouteID())
		assert.Equal(t, "reply", string(m.Data()))
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for client to receive reply")
	}
}

// TestRouter 测试路由器
func TestRouter(t *testing.T) {
	router := NewRouter()

	// 注册处理器
	called := make(map[uint32]bool)
	router.Register(1, func(conn Conn, msg Message) {
		called[1] = true
	})
	router.Register(2, func(conn Conn, msg Message) {
		called[2] = true
	})

	// 测试路由计数
	assert.Equal(t, 2, router.RouteCount())

	// 测试获取处理器
	handler1 := router.Get(1)
	assert.NotNil(t, handler1)

	handler2 := router.Get(2)
	assert.NotNil(t, handler2)

	handler3 := router.Get(3)
	assert.NotNil(t, handler3) // 返回 notFound 处理器

	// 执行处理器
	msg := NewMessage(1, []byte("test"))
	handler1(nil, msg)
	assert.True(t, called[1])

	// 移除处理器
	router.Remove(1)
	assert.Equal(t, 1, router.RouteCount())
}

// TestRouter_Group 测试路由组
func TestRouter_Group(t *testing.T) {
	router := NewRouter()

	group1 := router.Group(1)
	group1.Register(1, func(conn Conn, msg Message) {})
	group1.Register(2, func(conn Conn, msg Message) {})

	group2 := router.Group(2)
	group2.Register(1, func(conn Conn, msg Message) {})

	// 验证路由ID组合 (prefix << 16 | routeID)
	handler := router.Get((1 << 16) | 1)
	assert.NotNil(t, handler)

	handler = router.Get((1 << 16) | 2)
	assert.NotNil(t, handler)

	handler = router.Get((2 << 16) | 1)
	assert.NotNil(t, handler)
}

// TestRouter_Middleware 测试中间件
func TestRouter_Middleware(t *testing.T) {
	router := NewRouter()

	var order []string

	// 中间件1
	mw1 := func(next Handler) Handler {
		return func(conn Conn, msg Message) {
			order = append(order, "mw1-before")
			next(conn, msg)
			order = append(order, "mw1-after")
		}
	}

	// 中间件2
	mw2 := func(next Handler) Handler {
		return func(conn Conn, msg Message) {
			order = append(order, "mw2-before")
			next(conn, msg)
			order = append(order, "mw2-after")
		}
	}

	router.Use(mw1, mw2)

	var handlerCalled bool
	router.Register(1, func(conn Conn, msg Message) {
		handlerCalled = true
		order = append(order, "handler")
	})

	msg := NewMessage(1, []byte("test"))
	router.Handle(nil, msg)

	assert.True(t, handlerCalled)
	assert.Equal(t, []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}, order)
}

// TestCodec_SimpleCodec 测试简单编解码器
func TestCodec_SimpleCodec(t *testing.T) {
	codec := NewSimpleCodec()

	msg := NewMessage(100, []byte("hello world"))
	msg.SetSeq(42)

	// 编码
	data, err := codec.Encode(msg)
	assert.NoError(t, err)
	assert.Equal(t, 12+11, len(data))

	// 两阶段解码
	routeID, seq, bodyLen, err := codec.DecodeHeader(data[:codec.HeaderSize()])
	assert.NoError(t, err)
	assert.Equal(t, uint32(100), routeID)
	assert.Equal(t, uint32(42), seq)
	assert.Equal(t, 11, bodyLen)

	decoded, err := codec.DecodeBody(routeID, seq, data[codec.HeaderSize():])
	assert.NoError(t, err)
	assert.Equal(t, uint32(100), decoded.RouteID())
	assert.Equal(t, uint32(42), decoded.Seq())
	assert.Equal(t, "hello world", string(decoded.Data()))
}

// TestCodec_TLVCodec 测试TLV编解码器
func TestCodec_TLVCodec(t *testing.T) {
	codec := NewTLVCodec()

	msg := &TLVMessage{msgType: 5, data: []byte("test")}

	// 编码
	data, err := codec.Encode(msg)
	assert.NoError(t, err)
	assert.Equal(t, 3+4, len(data))

	// 两阶段解码
	routeID, seq, bodyLen, err := codec.DecodeHeader(data[:codec.HeaderSize()])
	assert.NoError(t, err)
	assert.Equal(t, uint32(5), routeID)
	assert.Equal(t, uint32(0), seq)
	assert.Equal(t, 4, bodyLen)

	decoded, err := codec.DecodeBody(routeID, seq, data[codec.HeaderSize():])
	assert.NoError(t, err)

	tlvMsg, ok := decoded.(*TLVMessage)
	assert.True(t, ok)
	assert.Equal(t, byte(5), tlvMsg.MsgType())
	assert.Equal(t, "test", string(tlvMsg.Data()))
}

// TestCodec_LineCodec 测试行编解码器
func TestCodec_LineCodec(t *testing.T) {
	codec := NewLineCodec()

	msg := NewMessage(0, []byte("hello"))

	// 编码
	data, err := codec.Encode(msg)
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", string(data))

	// LineCodec 用 ScanLine 解码
	decoded, n, err := codec.ScanLine(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, "hello", string(decoded.Data()))
}

// TestConnManager 测试连接管理器
func TestConnManager(t *testing.T) {
	manager := NewConnManager(10)

	// 创建模拟连接
	conn := &mockConn{
		id:   "test-1",
		data: make(map[interface{}]interface{}),
	}

	// 添加连接
	ok := manager.Add(conn)
	assert.True(t, ok)
	assert.Equal(t, 1, manager.Count())

	// 获取连接
	got, ok := manager.Get("test-1")
	assert.True(t, ok)
	assert.Equal(t, "test-1", got.ID())

	// 获取所有连接
	all := manager.GetAll()
	assert.Len(t, all, 1)

	// 移除连接
	manager.Remove("test-1")
	assert.Equal(t, 0, manager.Count())
}

// TestConnManager_MaxConnections 测试最大连接数限制
func TestConnManager_MaxConnections(t *testing.T) {
	manager := NewConnManager(2)

	conn1 := &mockConn{id: "1", data: make(map[interface{}]interface{})}
	conn2 := &mockConn{id: "2", data: make(map[interface{}]interface{})}
	conn3 := &mockConn{id: "3", data: make(map[interface{}]interface{})}

	assert.True(t, manager.Add(conn1))
	assert.True(t, manager.Add(conn2))
	assert.False(t, manager.Add(conn3)) // 超出限制

	assert.Equal(t, 2, manager.Count())
}

// TestMessage 测试消息
func TestMessage(t *testing.T) {
	msg := NewMessage(100, []byte("test data"))

	assert.Equal(t, uint32(100), msg.RouteID())
	assert.Equal(t, "test data", string(msg.Data()))

	// 修改数据
	msg.SetData([]byte("new data"))
	assert.Equal(t, "new data", string(msg.Data()))
}

// TestRecoveryMiddleware 测试恢复中间件
func TestRecoveryMiddleware(t *testing.T) {
	logger := &testLogger{}
	router := NewRouter()
	router.Use(Recovery(logger))

	router.Register(1, func(conn Conn, msg Message) {
		panic("test panic")
	})

	// 不应该panic（因为Recovery会捕获）
	assert.NotPanics(t, func() {
		msg := NewMessage(1, nil)
		router.Handle(nil, msg)
	})

	// 验证错误被记录
	assert.True(t, len(logger.errors) > 0)
	if len(logger.errors) > 0 {
		assert.Contains(t, logger.errors[0], "panic recovered")
	}
}

// TestLoggingMiddleware 测试日志中间件
func TestLoggingMiddleware(t *testing.T) {
	logger := &testLogger{}
	router := NewRouter()
	router.Use(Logging(logger))

	router.Register(1, func(conn Conn, msg Message) {})

	msg := NewMessage(1, []byte("test"))
	router.Handle(&mockConn{id: "test"}, msg)

	assert.True(t, len(logger.debugs) > 0)
}

// TestClient_PoolSize 测试客户端连接池会建立多个连接
func TestClient_PoolSize(t *testing.T) {
	router := NewRouter()
	received := make(chan Message, 10)

	router.Register(1, func(conn Conn, msg Message) {
		received <- msg
		reply := NewMessage(2, []byte("reply"))
		conn.Send(reply)
	})

	server, err := NewServer(
		WithAddress(":0"),
		WithConnType(ConnTypeTCP),
		WithCodec(NewSimpleCodec()),
	)
	assert.NoError(t, err)
	server.SetRouter(router)

	go server.Run()
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	addr := server.listener.Addr().String()

	clientRouter := NewRouter()
	clientReceived := make(chan Message, 10)
	clientRouter.Register(2, func(conn Conn, msg Message) {
		clientReceived <- msg
	})

	client := NewClient(
		WithConnType(ConnTypeTCP),
		WithCodec(NewSimpleCodec()),
		WithPoolSize(3),
	)
	client.SetRouter(clientRouter)

	err = client.Connect(addr)
	assert.NoError(t, err)
	defer client.Close()

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 3, client.PoolSize())
	assert.Equal(t, 3, client.AvailableConns())

	msg := NewMessage(1, []byte("hello"))
	err = client.Send(msg)
	assert.NoError(t, err)

	select {
	case m := <-received:
		assert.Equal(t, uint32(1), m.RouteID())
		assert.Equal(t, "hello", string(m.Data()))
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to receive message")
	}

	select {
	case m := <-clientReceived:
		assert.Equal(t, uint32(2), m.RouteID())
		assert.Equal(t, "reply", string(m.Data()))
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for client to receive reply")
	}
}

// BenchmarkRouter 路由性能测试
func BenchmarkRouter(b *testing.B) {
	router := NewRouter()
	router.Register(1, func(conn Conn, msg Message) {})

	msg := NewMessage(1, []byte("benchmark"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Handle(nil, msg)
	}
}

// BenchmarkCodec_Encode 编码性能测试
func BenchmarkCodec_Encode(b *testing.B) {
	codec := NewSimpleCodec()
	msg := NewMessage(1, make([]byte, 256))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		codec.Encode(msg)
	}
}

// BenchmarkCodec_Decode 解码性能测试
func BenchmarkCodec_Decode(b *testing.B) {
	codec := NewSimpleCodec()
	msg := NewMessage(1, make([]byte, 256))
	data, _ := codec.Encode(msg)
	hs := codec.HeaderSize()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		routeID, seq, bodyLen, _ := codec.DecodeHeader(data[:hs])
		codec.DecodeBody(routeID, seq, data[hs:hs+bodyLen])
	}
}

// ========== Mock 实现 ==========

type mockConn struct {
	id     string
	closed bool
	data   map[interface{}]interface{}
	mu     sync.RWMutex
}

func (m *mockConn) ID() string                         { return m.id }
func (m *mockConn) Type() ConnType                     { return ConnTypeTCP }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) Send(msg Message) error             { return nil }
func (m *mockConn) SendBytes(data []byte) error        { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }
func (m *mockConn) Context() context.Context           { return context.Background() }

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConn) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

func (m *mockConn) SetValue(key, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

func (m *mockConn) GetValue(key interface{}) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	return v, ok
}

type testLogger struct {
	debugs []string
	infos  []string
	warns  []string
	errors []string
	mu     sync.Mutex
}

func (l *testLogger) Debug(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.debugs = append(l.debugs, fmt.Sprintf(msg, args...))
}

func (l *testLogger) Info(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infos = append(l.infos, fmt.Sprintf(msg, args...))
}

func (l *testLogger) Warn(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warns = append(l.warns, fmt.Sprintf(msg, args...))
}

func (l *testLogger) Error(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors = append(l.errors, fmt.Sprintf(msg, args...))
}
