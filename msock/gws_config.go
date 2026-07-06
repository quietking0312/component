package msock

import (
	"net/http"
	"time"

	"github.com/lxzan/gws"
)

// GWSConfig 用于配置 gws 服务端选项。
// 零值或 nil 表示使用 msock 的默认行为（与 ServerConfig 中的缓冲区大小保持一致）。
type GWSConfig struct {
	// ReadBufferSize 读取缓冲区大小。
	ReadBufferSize int `mapstructure:"read_buffer_size" json:"read_buffer_size" yaml:"read_buffer_size"`
	// ReadMaxPayloadSize 读取最大负载大小。
	ReadMaxPayloadSize int `mapstructure:"read_max_payload_size" json:"read_max_payload_size" yaml:"read_max_payload_size"`
	// WriteBufferSize 写入缓冲区大小（gws 已废弃该参数，建议留空）。
	WriteBufferSize int `mapstructure:"write_buffer_size" json:"write_buffer_size" yaml:"write_buffer_size"`
	// WriteMaxPayloadSize 写入最大负载大小。
	WriteMaxPayloadSize int `mapstructure:"write_max_payload_size" json:"write_max_payload_size" yaml:"write_max_payload_size"`
	// ParallelEnabled 是否启用并行处理。
	ParallelEnabled bool `mapstructure:"parallel_enabled" json:"parallel_enabled" yaml:"parallel_enabled"`
	// ParallelGolimit 并行协程限制，<=0 表示不限制。
	ParallelGolimit int `mapstructure:"parallel_golimit" json:"parallel_golimit" yaml:"parallel_golimit"`
	// CheckUtf8Enabled 是否启用 UTF-8 检查。
	CheckUtf8Enabled bool `mapstructure:"check_utf8_enabled" json:"check_utf8_enabled" yaml:"check_utf8_enabled"`
	// HandshakeTimeout 握手超时时间。
	HandshakeTimeout time.Duration `mapstructure:"handshake_timeout" json:"handshake_timeout" yaml:"handshake_timeout"`
	// SubProtocols WebSocket 子协议列表。
	SubProtocols []string `mapstructure:"sub_protocols" json:"sub_protocols" yaml:"sub_protocols"`
	// ResponseHeader 握手时附加的响应头。
	ResponseHeader http.Header `mapstructure:"response_header" json:"response_header" yaml:"response_header"`
	// PermessageDeflate 压缩扩展配置。
	PermessageDeflate GWSPermessageDeflateConfig `mapstructure:"permessage_deflate" json:"permessage_deflate" yaml:"permessage_deflate"`

	// Recovery 自定义 panic 恢复函数，为 nil 时使用 gws.Recovery。
	Recovery func(logger gws.Logger) `mapstructure:"-" json:"-" yaml:"-"`
	// Authorize 自定义鉴权函数，为 nil 时允许所有请求。
	Authorize func(r *http.Request, session gws.SessionStorage) bool `mapstructure:"-" json:"-" yaml:"-"`
	// NewSession 自定义 SessionStorage 工厂，为 nil 时使用 gws 默认实现。
	NewSession func() gws.SessionStorage `mapstructure:"-" json:"-" yaml:"-"`
}

// GWSPermessageDeflateConfig 是 gws.PermessageDeflate 的可配置子集。
type GWSPermessageDeflateConfig struct {
	// Enabled 是否开启压缩。
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	// Level 压缩级别。
	Level int `mapstructure:"level" json:"level" yaml:"level"`
	// Threshold 压缩阈值，长度小于阈值的消息不会被压缩。
	Threshold int `mapstructure:"threshold" json:"threshold" yaml:"threshold"`
	// PoolSize 压缩器内存池大小。
	PoolSize int `mapstructure:"pool_size" json:"pool_size" yaml:"pool_size"`
	// ServerContextTakeover 服务端上下文接管。
	ServerContextTakeover bool `mapstructure:"server_context_takeover" json:"server_context_takeover" yaml:"server_context_takeover"`
	// ClientContextTakeover 客户端上下文接管。
	ClientContextTakeover bool `mapstructure:"client_context_takeover" json:"client_context_takeover" yaml:"client_context_takeover"`
	// ServerMaxWindowBits 服务端滑动窗口指数（8~15）。
	ServerMaxWindowBits int `mapstructure:"server_max_window_bits" json:"server_max_window_bits" yaml:"server_max_window_bits"`
	// ClientMaxWindowBits 客户端滑动窗口指数（8~15）。
	ClientMaxWindowBits int `mapstructure:"client_max_window_bits" json:"client_max_window_bits" yaml:"client_max_window_bits"`
}

// canonicalizeHTTPHeader 将 HTTP 头信息的 key 规范化为标准格式。
func canonicalizeHTTPHeader(h http.Header) http.Header {
	if len(h) == 0 {
		return h
	}
	canonical := make(http.Header, len(h))
	for k, v := range h {
		canonical[http.CanonicalHeaderKey(k)] = v
	}
	return canonical
}

// DefaultGWSConfig 返回与历史默认行为一致的 GWS 配置。
func DefaultGWSConfig() *GWSConfig {
	return &GWSConfig{
		ReadBufferSize:      4096,
		ReadMaxPayloadSize:  8192,
		WriteMaxPayloadSize: 8192,
		ParallelEnabled:     true,
		HandshakeTimeout:    10 * time.Second,
	}
}

// buildGWSServerOption 根据 ServerConfig 构造 gws.ServerOption。
func buildGWSServerOption(cfg *ServerConfig) *gws.ServerOption {
	opt := &gws.ServerOption{
		Recovery: gws.Recovery,
		Authorize: func(r *http.Request, session gws.SessionStorage) bool {
			return true
		},
	}

	if cfg.GWSConfig == nil {
		readBuf := cfg.ReadBufferSize
		if readBuf <= 0 {
			readBuf = 4096
		}
		writeBuf := cfg.WriteBufferSize
		if writeBuf <= 0 {
			writeBuf = 4096
		}

		opt.ReadBufferSize = readBuf
		opt.ReadMaxPayloadSize = readBuf * 2
		opt.WriteMaxPayloadSize = writeBuf * 2
		opt.HandshakeTimeout = 10 * time.Second
		opt.ParallelEnabled = true
		return opt
	}

	gwsCfg := cfg.GWSConfig

	readBuf := gwsCfg.ReadBufferSize
	if readBuf <= 0 {
		readBuf = cfg.ReadBufferSize
		if readBuf <= 0 {
			readBuf = 4096
		}
	}
	readMax := gwsCfg.ReadMaxPayloadSize
	if readMax <= 0 {
		readMax = readBuf * 2
	}

	writeBase := gwsCfg.WriteBufferSize
	if writeBase <= 0 {
		writeBase = cfg.WriteBufferSize
		if writeBase <= 0 {
			writeBase = 4096
		}
	}
	writeMax := gwsCfg.WriteMaxPayloadSize
	if writeMax <= 0 {
		writeMax = writeBase * 2
	}

	opt.ReadBufferSize = readBuf
	opt.ReadMaxPayloadSize = readMax
	opt.WriteBufferSize = gwsCfg.WriteBufferSize
	opt.WriteMaxPayloadSize = writeMax
	opt.ParallelEnabled = gwsCfg.ParallelEnabled
	opt.ParallelGolimit = gwsCfg.ParallelGolimit
	opt.CheckUtf8Enabled = gwsCfg.CheckUtf8Enabled
	if gwsCfg.HandshakeTimeout > 0 {
		opt.HandshakeTimeout = gwsCfg.HandshakeTimeout
	} else {
		opt.HandshakeTimeout = 10 * time.Second
	}
	opt.SubProtocols = gwsCfg.SubProtocols
	if len(gwsCfg.ResponseHeader) > 0 {
		opt.ResponseHeader = canonicalizeHTTPHeader(gwsCfg.ResponseHeader)
	}
	opt.PermessageDeflate = gws.PermessageDeflate{
		Enabled:               gwsCfg.PermessageDeflate.Enabled,
		Level:                 gwsCfg.PermessageDeflate.Level,
		Threshold:             gwsCfg.PermessageDeflate.Threshold,
		PoolSize:              gwsCfg.PermessageDeflate.PoolSize,
		ServerContextTakeover: gwsCfg.PermessageDeflate.ServerContextTakeover,
		ClientContextTakeover: gwsCfg.PermessageDeflate.ClientContextTakeover,
		ServerMaxWindowBits:   gwsCfg.PermessageDeflate.ServerMaxWindowBits,
		ClientMaxWindowBits:   gwsCfg.PermessageDeflate.ClientMaxWindowBits,
	}
	if gwsCfg.Recovery != nil {
		opt.Recovery = gwsCfg.Recovery
	}
	if gwsCfg.Authorize != nil {
		opt.Authorize = gwsCfg.Authorize
	}
	if gwsCfg.NewSession != nil {
		opt.NewSession = gwsCfg.NewSession
	}

	return opt
}
