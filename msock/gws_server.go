package msock

import (
	"net"
	"net/http"
	"time"

	"github.com/lxzan/gws"
)

// gwsEventHandler gws服务器事件处理器
type gwsEventHandler struct {
	server *Server
}

// OnOpen 连接建立事件
func (h *gwsEventHandler) OnOpen(socket *gws.Conn) {
	defer func() {
		if r := recover(); r != nil {
			h.server.logger.Errorf("panic in gws OnOpen: %v", r)
		}
	}()

	conn := newGWSConn(socket, h.server, h.server.codec)
	if !h.server.connManager.Add(conn) {
		h.server.logger.Warnf("max connections reached, reject gws connection from %s", socket.RemoteAddr())
		_ = socket.NetConn().Close()
		return
	}

	socket.Session().Store("msock_conn", conn)
	h.server.logger.Infof("gws connection established: %s from %s", conn.ID(), conn.RemoteAddr())

	if h.server.handlers.onConnect != nil {
		h.server.handlers.onConnect(conn)
	}
}

// OnClose 连接关闭事件
func (h *gwsEventHandler) OnClose(socket *gws.Conn, err error) {
	defer func() {
		if r := recover(); r != nil {
			h.server.logger.Errorf("panic in gws OnClose: %v", r)
		}
	}()

	v, ok := socket.Session().Load("msock_conn")
	if !ok {
		return
	}
	conn := v.(*gwsConn)

	_ = conn.Close()
	h.server.connManager.Remove(conn.ID())
	h.server.logger.Infof("gws connection closed: %s", conn.ID())

	if h.server.handlers.onDisconnect != nil {
		h.server.handlers.onDisconnect(conn)
	}
}

// OnPing 心跳探测事件
func (h *gwsEventHandler) OnPing(socket *gws.Conn, payload []byte) {
	_ = socket.WritePong(nil)
}

// OnPong 心跳响应事件
func (h *gwsEventHandler) OnPong(socket *gws.Conn, payload []byte) {}

// OnMessage 消息事件
func (h *gwsEventHandler) OnMessage(socket *gws.Conn, message *gws.Message) {
	defer message.Close()

	v, ok := socket.Session().Load("msock_conn")
	if !ok {
		return
	}
	conn := v.(*gwsConn)

	switch message.Opcode {
	case gws.OpcodeBinary:
		func() {
			defer func() {
				if r := recover(); r != nil {
					h.server.logger.Errorf("panic in handler: %v, conn: %s", r, conn.ID())
				}
			}()
			conn.handleBinaryMessage(message.Bytes())
		}()
	case gws.OpcodeText:
		conn.handleTextMessage(message.Bytes())
	}
}

// runGWSServer 运行gws WebSocket服务器
func (s *Server) runGWSServer() error {
	handler := &gwsEventHandler{server: s}

	upgrader := gws.NewUpgrader(handler, &gws.ServerOption{
		ReadBufferSize:      s.config.ReadBufferSize,
		ReadMaxPayloadSize:  s.config.ReadBufferSize * 2,
		WriteMaxPayloadSize: s.config.WriteBufferSize * 2,
		Recovery:            gws.Recovery,
		HandshakeTimeout:    10 * time.Second,
		Authorize: func(r *http.Request, session gws.SessionStorage) bool {
			return true
		},
	})

	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		s.logger.Errorf("gws listen error: %v", err)
		return ErrListenFailed
	}
	s.listener = listener

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		socket, err := upgrader.Upgrade(w, r)
		if err != nil {
			s.logger.Errorf("gws upgrade error: %v", err)
			return
		}

		s.wg.Add(1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Errorf("panic in gws readLoop: %v", r)
				}
				s.wg.Done()
			}()
			socket.ReadLoop()
		}()
	})

	s.httpServer = &http.Server{
		Addr:    s.config.Address,
		Handler: mux,
	}

	s.logger.Infof("gws server listening on %s/ws", listener.Addr().String())

	err = s.httpServer.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
