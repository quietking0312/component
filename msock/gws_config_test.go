package msock

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildGWSServerOption_Default(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.ReadBufferSize = 8192
	cfg.WriteBufferSize = 4096
	cfg.GWSConfig = nil

	opt := buildGWSServerOption(cfg)

	assert.Equal(t, 8192, opt.ReadBufferSize)
	assert.Equal(t, 16384, opt.ReadMaxPayloadSize)
	assert.Equal(t, 8192, opt.WriteMaxPayloadSize)
	assert.Equal(t, 10*time.Second, opt.HandshakeTimeout)
	assert.True(t, opt.ParallelEnabled)
	assert.NotNil(t, opt.Recovery)
	assert.NotNil(t, opt.Authorize)
}

func TestBuildGWSServerOption_Custom(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.GWSConfig = &GWSConfig{
		ReadBufferSize:      1024,
		ReadMaxPayloadSize:  2048,
		WriteMaxPayloadSize: 4096,
		ParallelEnabled:     false,
		ParallelGolimit:     8,
		CheckUtf8Enabled:    true,
		HandshakeTimeout:    5 * time.Second,
		SubProtocols:        []string{"chat"},
		PermessageDeflate: GWSPermessageDeflateConfig{
			Enabled: true,
			Level:   4,
		},
	}

	opt := buildGWSServerOption(cfg)

	assert.Equal(t, 1024, opt.ReadBufferSize)
	assert.Equal(t, 2048, opt.ReadMaxPayloadSize)
	assert.Equal(t, 4096, opt.WriteMaxPayloadSize)
	assert.False(t, opt.ParallelEnabled)
	assert.Equal(t, 8, opt.ParallelGolimit)
	assert.True(t, opt.CheckUtf8Enabled)
	assert.Equal(t, 5*time.Second, opt.HandshakeTimeout)
	assert.Equal(t, []string{"chat"}, opt.SubProtocols)
	assert.True(t, opt.PermessageDeflate.Enabled)
	assert.Equal(t, 4, opt.PermessageDeflate.Level)
}

func TestGWSConfig_JSONUnmarshal(t *testing.T) {
	raw := []byte(`{
		"read_buffer_size": 2048,
		"read_max_payload_size": 4096,
		"parallel_enabled": true,
		"parallel_golimit": 4,
		"handshake_timeout": 20000000000,
		"sub_protocols": ["chat", "game"],
		"permessage_deflate": {
			"enabled": true,
			"level": 4
		}
	}`)

	var cfg GWSConfig
	assert.NoError(t, json.Unmarshal(raw, &cfg))

	assert.Equal(t, 2048, cfg.ReadBufferSize)
	assert.Equal(t, 4096, cfg.ReadMaxPayloadSize)
	assert.True(t, cfg.ParallelEnabled)
	assert.Equal(t, 4, cfg.ParallelGolimit)
	assert.Equal(t, 20*time.Second, cfg.HandshakeTimeout)
	assert.Equal(t, []string{"chat", "game"}, cfg.SubProtocols)
	assert.True(t, cfg.PermessageDeflate.Enabled)
	assert.Equal(t, 4, cfg.PermessageDeflate.Level)
}

func TestWithGWSConfig(t *testing.T) {
	custom := &GWSConfig{ReadBufferSize: 1234}
	cfg := DefaultServerConfig()
	WithGWSConfig(custom)(cfg)
	assert.Equal(t, custom, cfg.GWSConfig)
}
