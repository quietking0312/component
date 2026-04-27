package mconf

import (
	"fmt"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config wraps viper.Viper. All state is instance-local; there is no global singleton.
type Config struct {
	v *viper.Viper
}

// New creates a new empty Config. Call SetDefault / Set before LoadFile if you need defaults.
func New() *Config {
	return &Config{v: viper.New()}
}

// LoadFile loads configuration from a file.
//   - path: absolute or relative file path. If empty, searches for "config" with any
//     supported extension (e.g. config.yaml, config.json, config.toml) in "." and "./config".
func LoadFile(path string) (*Config, error) {
	c := New()
	if path != "" {
		c.v.SetConfigFile(path)
	} else {
		c.v.SetConfigName("config")
		c.v.AddConfigPath(".")
		c.v.AddConfigPath("./config")
	}
	if err := c.v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return c, nil
}

// LoadFile sets the config file path (or default search paths if empty) and reads it.
// When path is empty, it searches for "config" with any supported extension
// (e.g. config.yaml, config.json, config.toml) in "." and "./config".
func (c *Config) LoadFile(path string) error {
	if path != "" {
		c.v.SetConfigFile(path)
	} else {
		c.v.SetConfigName("config")
		c.v.AddConfigPath(".")
		c.v.AddConfigPath("./config")
	}
	return c.v.ReadInConfig()
}

// ReadInConfig reads the configuration file that was previously configured.
func (c *Config) ReadInConfig() error {
	return c.v.ReadInConfig()
}

// EnableEnv enables automatic environment-variable parsing.
//   - prefix: env var prefix, e.g. "APP" maps APP_SERVER_HOST to server.host.
func (c *Config) EnableEnv(prefix string) *Config {
	if prefix != "" {
		c.v.SetEnvPrefix(prefix)
		c.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		c.v.AutomaticEnv()
	}
	return c
}

// SetDefault sets the default value for key. Useful before LoadFile so that
// missing keys in the config file fall back to your business defaults.
func (c *Config) SetDefault(key string, value interface{}) {
	c.v.SetDefault(key, value)
}

// Set overrides a configuration value at runtime.
func (c *Config) Set(key string, value interface{}) {
	c.v.Set(key, value)
}

// Get retrieves a raw value by key.
func (c *Config) Get(key string) interface{} {
	return c.v.Get(key)
}

// GetString retrieves a string value.
func (c *Config) GetString(key string) string {
	return c.v.GetString(key)
}

// GetInt retrieves an int value.
func (c *Config) GetInt(key string) int {
	return c.v.GetInt(key)
}

// GetInt32 retrieves an int32 value.
func (c *Config) GetInt32(key string) int32 {
	return c.v.GetInt32(key)
}

// GetInt64 retrieves an int64 value.
func (c *Config) GetInt64(key string) int64 {
	return c.v.GetInt64(key)
}

// GetUint retrieves a uint value.
func (c *Config) GetUint(key string) uint {
	return c.v.GetUint(key)
}

// GetBool retrieves a bool value.
func (c *Config) GetBool(key string) bool {
	return c.v.GetBool(key)
}

// GetFloat64 retrieves a float64 value.
func (c *Config) GetFloat64(key string) float64 {
	return c.v.GetFloat64(key)
}

// GetTime retrieves a time.Time value.
func (c *Config) GetTime(key string) time.Time {
	return c.v.GetTime(key)
}

// GetDuration retrieves a time.Duration value.
func (c *Config) GetDuration(key string) time.Duration {
	return c.v.GetDuration(key)
}

// GetStringSlice retrieves a []string value.
func (c *Config) GetStringSlice(key string) []string {
	return c.v.GetStringSlice(key)
}

// GetIntSlice retrieves a []int value.
func (c *Config) GetIntSlice(key string) []int {
	return c.v.GetIntSlice(key)
}

// GetStringMap retrieves a map[string]interface{} value.
func (c *Config) GetStringMap(key string) map[string]interface{} {
	return c.v.GetStringMap(key)
}

// GetStringMapString retrieves a map[string]string value.
func (c *Config) GetStringMapString(key string) map[string]string {
	return c.v.GetStringMapString(key)
}

// IsSet reports whether key has a value.
func (c *Config) IsSet(key string) bool {
	return c.v.IsSet(key)
}

// Sub returns a sub-tree of the config.
func (c *Config) Sub(key string) *viper.Viper {
	return c.v.Sub(key)
}

// AllSettings returns all settings as a map.
func (c *Config) AllSettings() map[string]interface{} {
	return c.v.AllSettings()
}

// Unmarshal unmarshals the config into rawVal.
func (c *Config) Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return c.v.Unmarshal(rawVal, opts...)
}

// UnmarshalKey unmarshals the config subtree at key into rawVal.
func (c *Config) UnmarshalKey(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return c.v.UnmarshalKey(key, rawVal, opts...)
}

// WatchConfig starts watching the config file for changes.
func (c *Config) WatchConfig() {
	c.v.WatchConfig()
}

// OnConfigChange registers a callback invoked when the config file changes.
func (c *Config) OnConfigChange(run func()) {
	c.v.OnConfigChange(func(in fsnotify.Event) {
		run()
	})
}

// OnConfigChangeEvent registers a callback invoked with the fsnotify event when the config file changes.
func (c *Config) OnConfigChangeEvent(run func(event fsnotify.Event)) {
	c.v.OnConfigChange(func(in fsnotify.Event) {
		run(in)
	})
}

// ConfigFileUsed returns the path of the config file currently in use.
func (c *Config) ConfigFileUsed() string {
	return c.v.ConfigFileUsed()
}

// BindEnv binds one or more environment variables to a config key.
func (c *Config) BindEnv(input ...string) error {
	return c.v.BindEnv(input...)
}

// WriteConfig writes the current configuration to the file path set by SetConfigFile.
func (c *Config) WriteConfig() error {
	return c.v.WriteConfig()
}

// SafeWriteConfig writes the current configuration only if the file does not already exist.
func (c *Config) SafeWriteConfig() error {
	return c.v.SafeWriteConfig()
}

// WriteConfigAs writes the current configuration to the specified file.
func (c *Config) WriteConfigAs(filename string) error {
	return c.v.WriteConfigAs(filename)
}

// SafeWriteConfigAs writes the current configuration to the specified file only if it does not exist.
func (c *Config) SafeWriteConfigAs(filename string) error {
	return c.v.SafeWriteConfigAs(filename)
}

// Viper returns the underlying viper instance for advanced use.
func (c *Config) Viper() *viper.Viper {
	return c.v
}
