package msock

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// runWebSocketServer 运行WebSocket服务器（在Server.runWebSocket中调用）
func (s *Server) runWebSocketServer() error {
	upgrader := NewWebSocketUpgrader()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.logger.Error(fmt.Sprintf("websocket upgrade error: %v", err))
			return
		}

		s.wg.Add(1)
		go s.handleWSConn(conn)
	})

	s.httpServer = &http.Server{
		Addr:    s.config.Address,
		Handler: mux,
	}

	s.logger.Info(fmt.Sprintf("websocket server listening on %s/ws", s.config.Address))

	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// handleWSConn 处理WebSocket连接
func (s *Server) handleWSConn(wsConnObj *websocket.Conn) {
	defer s.wg.Done()

	conn := newWSConn(wsConnObj, s)

	if !s.connManager.Add(conn) {
		s.logger.Warn(fmt.Sprintf("max connections reached, reject websocket connection from %s", wsConnObj.RemoteAddr()))
		_ = wsConnObj.Close()
		return
	}
	defer s.connManager.Remove(conn.ID())

	s.metrics.totalConns.Add(1)
	s.logger.Info(fmt.Sprintf("websocket connection established: %s from %s", conn.ID(), conn.RemoteAddr()))

	if s.handlers.onConnect != nil {
		s.handlers.onConnect(conn)
	}

	conn.readLoop()

	s.logger.Info(fmt.Sprintf("websocket connection closed: %s", conn.ID()))

	if s.handlers.onDisconnect != nil {
		s.handlers.onDisconnect(conn)
	}
}

// RegisterWSRoute 注册WebSocket处理路由（如果使用HTTP路由）
func (s *Server) RegisterWSRoute(path string, upgrader *WebSocketUpgrader) {
	if s.httpServer == nil {
		return
	}
}
