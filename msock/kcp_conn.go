package msock

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/xtaci/kcp-go/v5"
)

// KCPConfig KCP配置
type KCPConfig struct {
	// 发送窗口大小
	SendWindow int
	// 接收窗口大小
	RecvWindow int
	// 数据包最大传输单元
	Mtu int
	// 是否启用 FEC
	EnableFEC bool
	// FEC 数据分片数
	DataShards int
	// FEC 校验分片数
	ParityShards int
	// 是否启用加密
	EnableCrypt bool
	// 加密密钥（EnableCrypt 为 true 时必填）
	CryptKey string

	// 以下对应 kcp.UDPSession.SetNoDelay 的四个参数
	// NoDelay: 0=关闭，1=开启 nodelay 模式
	NoDelay int
	// Interval: 内部刷新时间间隔（毫秒）
	Interval int
	// Resend: 快速重传模式，0=关闭，2=推荐值
	Resend int
	// NC: 是否关闭流量控制，0=开启，1=关闭
	NC int
}

// DefaultKCPConfig 返回默认KCP配置
func DefaultKCPConfig() *KCPConfig {
	return &KCPConfig{
		SendWindow:   128,
		RecvWindow:   128,
		Mtu:          1350,
		EnableFEC:    false,
		DataShards:   10,
		ParityShards: 3,
		EnableCrypt:  false,
		CryptKey:     "",
		NoDelay:      1,
		Interval:     10,
		Resend:       2,
		NC:           1,
	}
}

// newKCPBlockCrypt 从密钥字符串创建 AES 块加密器（固定填充为 32 字节）
func newKCPBlockCrypt(key string) (kcp.BlockCrypt, error) {
	k := make([]byte, 32)
	copy(k, []byte(key))
	return kcp.NewAESBlockCrypt(k)
}

// kcpConn KCP连接实现
type kcpConn struct {
	*baseConn
	net.Conn
	server       *Server
	codec        Codec
	writeTimeout time.Duration
}

// newKCPConn 创建KCP连接
func newKCPConn(conn net.Conn, server *Server) *kcpConn {
	c := &kcpConn{
		baseConn: newBaseConn(ConnTypeKCP, server.config.IDGenerator()),
		Conn:     conn,
		server:   server,
		codec:    server.codec,
	}
	c.initSendQueue(128)
	c.sendWg.Add(1)
	go c.sendLoop()
	return c
}

// LocalAddr 返回本地地址
func (c *kcpConn) LocalAddr() net.Addr {
	return c.Conn.LocalAddr()
}

// RemoteAddr 返回远程地址
func (c *kcpConn) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

// Send 发送消息
func (c *kcpConn) Send(msg Message) error {
	if c.IsClosed() {
		return ErrConnClosed
	}

	data, err := c.codec.Encode(msg)
	if err != nil {
		return err
	}

	return c.SendBytes(data)
}

// sendLoop KCP发送协程
func (c *kcpConn) sendLoop() {
	defer c.sendWg.Done()
	for data := range c.sendCh {
		writeTimeout := c.writeTimeout
		if writeTimeout <= 0 {
			if c.server != nil && c.server.config.WriteTimeout > 0 {
				writeTimeout = c.server.config.WriteTimeout
			} else {
				writeTimeout = 10 * time.Second
			}
		}
		if err := c.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			c.Close()
			return
		}
		if _, err := c.Conn.Write(data); err != nil {
			c.Close()
			return
		}
	}
}

// SendBytes 发送原始字节
func (c *kcpConn) SendBytes(data []byte) error {
	return c.sendAsync(data)
}

// Close 关闭连接
func (c *kcpConn) Close() error {
	if c.IsClosed() {
		return nil
	}
	c.close()
	c.waitSendDone()
	return c.Conn.Close()
}

// SetReadDeadline 设置读取超时
func (c *kcpConn) SetReadDeadline(t time.Time) error {
	return c.Conn.SetReadDeadline(t)
}

// SetWriteDeadline 设置写入超时
func (c *kcpConn) SetWriteDeadline(t time.Time) error {
	return c.Conn.SetWriteDeadline(t)
}

// readLoop 读取循环（两阶段解码）
func (c *kcpConn) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			c.server.logger.Error(fmt.Sprintf("panic in kcp readLoop: %v", r))
		}
		c.Close()
	}()

	codec := c.server.codec
	headerSize := codec.HeaderSize()
	header := make([]byte, headerSize)

	for {
		if c.IsClosed() {
			return
		}

		if c.server.config.ReadTimeout > 0 {
			if err := c.SetReadDeadline(time.Now().Add(c.server.config.ReadTimeout)); err != nil {
				c.server.logger.Warn(fmt.Sprintf("set read deadline error: %v", err))
				return
			}
		}

		var msg Message

		if headerSize == 0 {
			lc, ok := codec.(*LineCodec)
			if !ok {
				c.server.logger.Error("codec headerSize=0 but is not LineCodec")
				return
			}
			line, err := scanLine(c.Conn, lc.MaxPacketSize())
			if err != nil {
				if !c.IsClosed() {
					c.server.logger.Debug(fmt.Sprintf("kcp read error: %v", err))
				}
				return
			}
			msg = NewMessage(0, line)
		} else {
			if _, err := io.ReadFull(c.Conn, header); err != nil {
				if !c.IsClosed() {
					c.server.logger.Debug(fmt.Sprintf("kcp read header error: %v", err))
				}
				return
			}

			routeID, bodyLen, err := codec.DecodeHeader(header)
			if err != nil {
				c.server.logger.Error(fmt.Sprintf("kcp decode header error: %v", err))
				return
			}

			var body []byte
			if bodyLen > 0 {
				body = acquireBody(bodyLen)
				if _, err = io.ReadFull(c.Conn, body); err != nil {
					releaseBody(body)
					if !c.IsClosed() {
						c.server.logger.Debug(fmt.Sprintf("kcp read body error: %v", err))
					}
					return
				}
			}

			msg, err = codec.DecodeBody(routeID, body)
			if bodyLen > 0 {
				releaseBody(body)
			}
			if err != nil {
				c.server.logger.Error(fmt.Sprintf("kcp decode body error: %v", err))
				return
			}
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					c.server.logger.Error(fmt.Sprintf("panic in handler: %v, conn: %s", r, c.ID()))
				}
			}()
			c.server.handleMessage(c, msg)
		}()
	}
}
