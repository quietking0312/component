package msock

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// RPC 响应状态
const (
	rpcStatusOK  byte = 0
	rpcStatusErr byte = 1
)

// rpcState 客户端 RPC 运行时状态
type rpcState struct {
	replyRouteID uint32
	timeout      time.Duration
	seq          atomic.Uint32
	pending      sync.Map // map[uint32]*rpcPending
}

// rpcResult 一次 RPC 调用的结果
type rpcResult struct {
	data []byte
	err  error
}

// rpcPending 一次进行中的 RPC 调用，记录发出请求所用的连接，
// 以便该连接断开时能立即让等待方返回，而不是干等到超时。
type rpcPending struct {
	ch   chan rpcResult
	conn Conn
}

// EnableRPC 开启客户端 RPC 能力，需在 SetRouter 之后调用。
// replyRouteID 为专用于 RPC 响应的路由ID，需与业务路由及心跳路由ID不冲突。
// defaultTimeout 为 Call 未指定 deadline 时使用的默认超时，<=0 时使用 10s。
func (c *Client) EnableRPC(replyRouteID uint32, defaultTimeout time.Duration) error {
	if c.router == nil {
		return ErrRouterNotSet
	}
	if defaultTimeout <= 0 {
		defaultTimeout = 10 * time.Second
	}

	state := &rpcState{
		replyRouteID: replyRouteID,
		timeout:      defaultTimeout,
	}
	c.rpc.Store(state)
	c.router.Register(replyRouteID, c.handleRPCReply)
	return nil
}

// handleRPCReply 处理 RPC 响应，按 seq 匹配等待中的调用
func (c *Client) handleRPCReply(conn Conn, msg Message) {
	state := c.rpc.Load()
	if state == nil {
		return
	}

	data := msg.Data()
	if len(data) < 1 {
		return
	}
	seq := msg.Seq()
	status := data[0]
	payload := data[1:]

	v, ok := state.pending.LoadAndDelete(seq)
	if !ok {
		return
	}
	pend := v.(*rpcPending)

	if status == rpcStatusErr {
		pend.ch <- rpcResult{err: fmt.Errorf("rpc error: %s", payload)}
		return
	}
	pend.ch <- rpcResult{data: payload}
}

// failPendingRPC 让绑定在 conn 上的所有等待中调用立即以 err 返回，
// 用于连接断开（重启、掉线等）时避免业务侧一直等到超时才发现问题。
func (c *Client) failPendingRPC(conn Conn, err error) {
	state := c.rpc.Load()
	if state == nil {
		return
	}
	state.pending.Range(func(key, value any) bool {
		pend := value.(*rpcPending)
		if pend.conn != conn {
			return true
		}
		state.pending.Delete(key)
		select {
		case pend.ch <- rpcResult{err: err}:
		default:
		}
		return true
	})
}

// Call 发起一次 RPC 调用并阻塞等待响应。
// routeID 为请求路由ID，data 为请求负载。ctx 未设置 deadline 时使用 EnableRPC 配置的默认超时。
func (c *Client) Call(ctx context.Context, routeID uint32, data []byte) ([]byte, error) {
	state := c.rpc.Load()
	if state == nil {
		return nil, ErrRPCNotEnabled
	}

	conn := c.pickConn()
	if conn == nil {
		return nil, ErrNoAvailableConn
	}

	seq := state.seq.Add(1)
	pend := &rpcPending{ch: make(chan rpcResult, 1), conn: conn}
	state.pending.Store(seq, pend)
	defer state.pending.Delete(seq)

	msg := NewMessage(routeID, data)
	msg.SetSeq(seq)

	if err := c.sendOn(conn, msg); err != nil {
		return nil, err
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, state.timeout)
		defer cancel()
	}

	select {
	case res := <-pend.ch:
		return res.data, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.closeCh:
		return nil, ErrConnClosed
	}
}

// CallTimeout 是 Call 的便捷封装，直接指定超时时间
func (c *Client) CallTimeout(routeID uint32, data []byte, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.Call(ctx, routeID, data)
}

// ========== 服务端配套辅助函数 ==========

// ParseRPCRequest 解析 RPC 请求消息，返回 seq 和请求负载
func ParseRPCRequest(msg Message) (seq uint32, payload []byte, err error) {
	return msg.Seq(), msg.Data(), nil
}

// ReplyRPC 向指定连接发送一次成功的 RPC 响应
func ReplyRPC(conn Conn, replyRouteID uint32, seq uint32, payload []byte) error {
	msg := NewMessage(replyRouteID, encodeRPCReplyBody(rpcStatusOK, payload))
	msg.SetSeq(seq)
	return conn.Send(msg)
}

// ReplyRPCError 向指定连接发送一次失败的 RPC 响应
func ReplyRPCError(conn Conn, replyRouteID uint32, seq uint32, rpcErr error) error {
	var errMsg string
	if rpcErr != nil {
		errMsg = rpcErr.Error()
	}
	msg := NewMessage(replyRouteID, encodeRPCReplyBody(rpcStatusErr, []byte(errMsg)))
	msg.SetSeq(seq)
	return conn.Send(msg)
}

// encodeRPCReplyBody 组装 RPC 响应 body：[1字节 status][payload]
func encodeRPCReplyBody(status byte, payload []byte) []byte {
	body := make([]byte, 1+len(payload))
	body[0] = status
	copy(body[1:], payload)
	return body
}
