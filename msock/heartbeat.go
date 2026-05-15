package msock

import (
	"fmt"
	"time"
)

// ========== 服务端心跳检测 ==========

// startHeartbeatChecker 服务端启动心跳检测 goroutine
func (s *Server) startHeartbeatChecker() {
	if s.config.HeartbeatInterval <= 0 || s.config.HeartbeatTimeout <= 0 {
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.config.HeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.checkHeartbeats()
			case <-s.stopCh:
				return
			}
		}
	}()
}

// checkHeartbeats 遍历所有连接，踢掉超时的连接
func (s *Server) checkHeartbeats() {
	now := time.Now()
	timeout := s.config.HeartbeatTimeout

	for _, conn := range s.connManager.GetAll() {
		bc := extractBaseConn(conn)
		if bc == nil {
			continue
		}
		if now.Sub(bc.LastHeartbeat()) > timeout {
			s.logger.Warn(fmt.Sprintf("heartbeat timeout, close conn: %s", conn.ID()))
			_ = conn.Close()
		}
	}
}

// extractBaseConn 从 Conn 接口中提取 *baseConn
func extractBaseConn(conn Conn) *baseConn {
	switch c := conn.(type) {
	case *tcpConn:
		return c.baseConn
	case *tcpClientConn:
		return c.baseConn
	case *kcpConn:
		return c.baseConn
	case *kcpClientConn:
		return c.baseConn
	case *wsConn:
		return c.baseConn
	case *wsClientConn:
		return c.baseConn
	case *gwsConn:
		return c.baseConn
	case *gwsClientConn:
		return c.baseConn
	}
	return nil
}

// handleHeartbeat 处理收到的心跳包
// 服务端收到 Ping → 回复 Pong；收到 Pong → 无需额外处理（时间戳已在 handleMessage 中更新）
func (s *Server) handleHeartbeat(conn Conn, msg Message) {
	if msg.RouteID() == s.config.HeartbeatPingID {
		var body []byte
		if s.config.HeartbeatPongData != nil {
			body = s.config.HeartbeatPongData(msg)
		}
		pong := NewMessage(s.config.HeartbeatPongID, body)
		if err := conn.Send(pong); err != nil && !conn.IsClosed() {
			s.logger.Warn(fmt.Sprintf("send pong error: %v, conn: %s", err, conn.ID()))
		}
	}
}

// RegisterHeartbeat 向 Router 注册心跳处理器，需在 SetRouter 之后调用
func (s *Server) RegisterHeartbeat(router *Router) {
	if s.config.HeartbeatInterval <= 0 {
		return
	}
	router.Register(s.config.HeartbeatPingID, func(conn Conn, msg Message) {
		s.handleHeartbeat(conn, msg)
	})
	router.Register(s.config.HeartbeatPongID, func(conn Conn, msg Message) {
		s.handleHeartbeat(conn, msg)
	})
}

// ========== 客户端心跳发送 ==========

// startHeartbeat 客户端启动心跳 goroutine
// 按 HeartbeatInterval 定时发送 Ping，超过 HeartbeatTimeout 未收到 Pong 则关闭连接触发重连。
func (c *Client) startHeartbeat() {
	if c.config.HeartbeatInterval <= 0 || c.config.HeartbeatTimeout <= 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(c.config.HeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.mu.RLock()
				conn := c.conn
				c.mu.RUnlock()

				if conn == nil || conn.IsClosed() {
					return
				}

				// 检查上次 Pong 是否超时
				bc := extractBaseConn(conn)
				if bc != nil && time.Since(bc.LastHeartbeat()) > c.config.HeartbeatTimeout {
					c.logger.Warn(fmt.Sprintf("heartbeat timeout, closing connection"))
					_ = conn.Close()
					return
				}

				// 发送 Ping
				ping := NewMessage(c.config.HeartbeatPingID, nil)
				if err := conn.Send(ping); err != nil && !conn.IsClosed() {
					c.logger.Warn(fmt.Sprintf("send ping error: %v", err))
				}

			case <-c.closeCh:
				return
			}
		}
	}()
}

// HandleHeartbeat 客户端收到 Pong 时调用，更新心跳时间戳
// 需要在客户端 Router 中注册 HeartbeatPongID 路由时调用此函数
func (c *Client) HandleHeartbeat(conn Conn, msg Message) {
	bc := extractBaseConn(conn)
	if bc != nil {
		bc.TouchHeartbeat()
	}
}

// RegisterHeartbeat 向 Router 注册客户端心跳 Pong 处理器
func (c *Client) RegisterHeartbeat(router *Router) {
	if c.config.HeartbeatInterval <= 0 {
		return
	}
	router.Register(c.config.HeartbeatPongID, func(conn Conn, msg Message) {
		c.HandleHeartbeat(conn, msg)
	})
}
