package msock

import "errors"

var (
	// ErrConnClosed 连接已关闭
	ErrConnClosed = errors.New("connection closed")

	// ErrServerClosed 服务器已关闭
	ErrServerClosed = errors.New("server closed")

	// ErrInvalidMessage 无效消息
	ErrInvalidMessage = errors.New("invalid message")

	// ErrCodecNotSet 编解码器未设置
	ErrCodecNotSet = errors.New("codec not set")

	// ErrRouterNotSet 路由器未设置
	ErrRouterNotSet = errors.New("router not set")

	// ErrMaxConnections 达到最大连接数
	ErrMaxConnections = errors.New("max connections reached")

	// ErrListenFailed 监听失败
	ErrListenFailed = errors.New("listen failed")

	// ErrAcceptFailed 接受连接失败
	ErrAcceptFailed = errors.New("accept connection failed")

	// ErrTimeout 超时
	ErrTimeout = errors.New("timeout")

	// ErrPacketTooLarge 数据包太大
	ErrPacketTooLarge = errors.New("packet too large")

	// ErrInvalidPacket 无效数据包
	ErrInvalidPacket = errors.New("invalid packet")

	// ErrUnsupportedProtocol 不支持的协议
	ErrUnsupportedProtocol = errors.New("unsupported protocol")

	// ErrSendChannelFull 发送通道已满
	ErrSendChannelFull = errors.New("send channel full")

	// ErrNoAvailableConn 连接池中无可用连接
	ErrNoAvailableConn = errors.New("no available connection in pool")

	// ErrRPCNotEnabled RPC 未启用
	ErrRPCNotEnabled = errors.New("rpc not enabled, call EnableRPC first")
)
