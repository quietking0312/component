package msock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	rpcTestReqRouteID   uint32 = 100
	rpcTestReplyRouteID uint32 = 101
	rpcTestHangRouteID  uint32 = 102
)

func startRPCTestServer(t *testing.T) (*Server, string) {
	t.Helper()

	router := NewRouter()
	router.Register(rpcTestReqRouteID, func(conn Conn, msg Message) {
		seq, payload, err := ParseRPCRequest(msg)
		assert.NoError(t, err)
		ReplyRPC(conn, rpcTestReplyRouteID, seq, append([]byte("echo:"), payload...))
	})
	router.Register(rpcTestHangRouteID, func(conn Conn, msg Message) {
		// 故意不回复，用于测试超时
	})

	server, err := NewServer(
		WithAddress(":0"),
		WithConnType(ConnTypeTCP),
		WithCodec(NewSimpleCodec()),
	)
	assert.NoError(t, err)
	server.SetRouter(router)

	go server.Run()
	t.Cleanup(func() { server.Stop() })

	time.Sleep(100 * time.Millisecond)
	return server, server.listener.Addr().String()
}

func TestClient_RPC_Call(t *testing.T) {
	_, addr := startRPCTestServer(t)

	clientRouter := NewRouter()
	client := NewClient(
		WithConnType(ConnTypeTCP),
		WithCodec(NewSimpleCodec()),
	)
	client.SetRouter(clientRouter)
	assert.NoError(t, client.EnableRPC(rpcTestReplyRouteID, time.Second))

	assert.NoError(t, client.Connect(addr))
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.Call(ctx, rpcTestReqRouteID, []byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, "echo:hello", string(resp))
}

func TestClient_RPC_Timeout(t *testing.T) {
	_, addr := startRPCTestServer(t)

	clientRouter := NewRouter()
	client := NewClient(
		WithConnType(ConnTypeTCP),
		WithCodec(NewSimpleCodec()),
	)
	client.SetRouter(clientRouter)
	assert.NoError(t, client.EnableRPC(rpcTestReplyRouteID, 300*time.Millisecond))

	assert.NoError(t, client.Connect(addr))
	defer client.Close()

	_, err := client.CallTimeout(rpcTestHangRouteID, []byte("hello"), 300*time.Millisecond)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
}

func TestClient_RPC_FailOnDisconnect(t *testing.T) {
	_, addr := startRPCTestServer(t)

	clientRouter := NewRouter()
	client := NewClient(
		WithConnType(ConnTypeTCP),
		WithCodec(NewSimpleCodec()),
	)
	client.SetRouter(clientRouter)
	assert.NoError(t, client.EnableRPC(rpcTestReplyRouteID, 5*time.Second))

	assert.NoError(t, client.Connect(addr))
	defer client.Close()

	resultCh := make(chan error, 1)
	go func() {
		_, err := client.CallTimeout(rpcTestHangRouteID, []byte("hello"), 5*time.Second)
		resultCh <- err
	}()

	// 等待请求发出后，模拟连接因重启/掉线被关闭，而不是整个 Client.Close()
	time.Sleep(100 * time.Millisecond)
	conn := client.Conn()
	assert.NotNil(t, conn)
	assert.NoError(t, conn.Close())

	select {
	case err := <-resultCh:
		assert.ErrorIs(t, err, ErrConnClosed)
	case <-time.After(1 * time.Second):
		t.Fatal("Call 应在连接断开后立即返回，而不是等到超时")
	}
}

func TestClient_RPC_NotEnabled(t *testing.T) {
	client := NewClient(WithConnType(ConnTypeTCP))
	_, err := client.CallTimeout(rpcTestReqRouteID, []byte("hello"), time.Second)
	assert.ErrorIs(t, err, ErrRPCNotEnabled)
}
