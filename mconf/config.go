package mconf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	// 全局默认 viper 实例，Init 前为 nil
	viperInstance *viper.Viper
)

// Config 配置管理器
type Config struct {
	v *viper.Viper
}

// Init 初始化全局配置
//   - configPath: 配置文件路径，为空时使用默认搜索路径
//   - envPrefix: 环境变量前缀，为空时不启用环境变量
func Init(configPath string, envPrefix string) error {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
	}

	// 读取环境变量
	if envPrefix != "" {
		v.SetEnvPrefix(envPrefix)
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv()
	}

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("read config file failed: %w", err)
		}
	}

	viperInstance = v
	return nil
}

// InitWithConfig 使用自定义 viper 实例初始化全局配置
func InitWithConfig(v *viper.Viper) {
	viperInstance = v
}

// setDefaults 设置默认值
func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.read_timeout", 60)
	v.SetDefault("server.write_timeout", 60)

	v.SetDefault("database.driver", "mysql")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.database", "admin")
	v.SetDefault("database.username", "root")
	v.SetDefault("database.password", "")
	v.SetDefault("database.charset", "utf8mb4")
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.max_idle_conns", 10)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
	v.SetDefault("log.output", "./logs/app.log")
	v.SetDefault("log.console", true)
	v.SetDefault("log.max_size", 100)
	v.SetDefault("log.max_backups", 10)
	v.SetDefault("log.max_age", 30)

	v.SetDefault("jwt.secret", "your-secret-key")
	v.SetDefault("jwt.expire", 86400)
	v.SetDefault("jwt.issuer", "admin-server")
}

// GetConfig 获取配置管理器实例
func GetConfig() *Config {
	return &Config{v: getViper()}
}

// getViper 安全获取全局 viper 实例
func getViper() *viper.Viper {
	if viperInstance == nil {
		viperInstance = viper.New()
	}
	return viperInstance
}

// ========== 配置读取方法 ==========

// Get 获取配置值
func Get(key string) interface{} {
	return getViper().Get(key)
}

// GetString 获取字符串配置
func GetString(key string) string {
	return getViper().GetString(key)
}

// GetInt 获取整数配置
func GetInt(key string) int {
	return getViper().GetInt(key)
}

// GetInt32 获取 int32 配置
func GetInt32(key string) int32 {
	return getViper().GetInt32(key)
}

// GetInt64 获取 int64 配置
func GetInt64(key string) int64 {
	return getViper().GetInt64(key)
}

// GetUint 获取无符号整数配置
func GetUint(key string) uint {
	return getViper().GetUint(key)
}

// GetBool 获取布尔配置
func GetBool(key string) bool {
	return getViper().GetBool(key)
}

// GetFloat64 获取浮点数配置
func GetFloat64(key string) float64 {
	return getViper().GetFloat64(key)
}

// GetTime 获取时间配置
func GetTime(key string) time.Time {
	return getViper().GetTime(key)
}

// GetDuration 获取时长配置
func GetDuration(key string) time.Duration {
	return getViper().GetDuration(key)
}

// GetStringSlice 获取字符串数组配置
func GetStringSlice(key string) []string {
	return getViper().GetStringSlice(key)
}

// GetIntSlice 获取整数数组配置
func GetIntSlice(key string) []int {
	return getViper().GetIntSlice(key)
}

// GetStringMap 获取 Map 配置
func GetStringMap(key string) map[string]interface{} {
	return getViper().GetStringMap(key)
}

// GetStringMapString 获取 Map[string]string 配置
func GetStringMapString(key string) map[string]string {
	return getViper().GetStringMapString(key)
}

// IsSet 检查配置是否存在
func IsSet(key string) bool {
	return getViper().IsSet(key)
}

// Sub 获取子配置
func Sub(key string) *viper.Viper {
	return getViper().Sub(key)
}

// AllSettings 获取所有配置
func AllSettings() map[string]interface{} {
	return getViper().AllSettings()
}

// ========== 配置设置方法 ==========

// Set 设置配置值
func Set(key string, value interface{}) {
	getViper().Set(key, value)
}

// SetDefault 设置默认值
func SetDefault(key string, value interface{}) {
	getViper().SetDefault(key, value)
}

// ========== 配置文件操作 ==========

// ReadInConfig 读取配置文件
func ReadInConfig() error {
	return getViper().ReadInConfig()
}

// WriteConfig 写入配置文件
func WriteConfig() error {
	return getViper().WriteConfig()
}

// SafeWriteConfig 安全写入配置文件（不存在时才写入）
func SafeWriteConfig() error {
	return getViper().SafeWriteConfig()
}

// WriteConfigAs 写入配置文件到指定路径
func WriteConfigAs(filename string) error {
	return getViper().WriteConfigAs(filename)
}

// SafeWriteConfigAs 安全写入配置文件到指定路径
func SafeWriteConfigAs(filename string) error {
	return getViper().SafeWriteConfigAs(filename)
}

// ========== 配置监听 ==========

// WatchConfig 监听配置文件变化
func WatchConfig() {
	getViper().WatchConfig()
}

// OnConfigChange 配置变化回调
func OnConfigChange(run func()) {
	getViper().OnConfigChange(func(in fsnotify.Event) {
		run()
	})
}

// OnConfigChangeEvent 配置变化回调（带事件信息）
func OnConfigChangeEvent(run func(event fsnotify.Event)) {
	getViper().OnConfigChange(func(in fsnotify.Event) {
		run(in)
	})
}

// ========== 辅助方法 ==========

// ConfigFileUsed 获取当前使用的配置文件路径
func ConfigFileUsed() string {
	return getViper().ConfigFileUsed()
}

// BindEnv 绑定环境变量
func BindEnv(input ...string) error {
	return getViper().BindEnv(input...)
}

// Unmarshal 反序列化配置到结构体
func Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return getViper().Unmarshal(rawVal, opts...)
}

// UnmarshalKey 反序列化指定 key 到结构体
func UnmarshalKey(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return getViper().UnmarshalKey(key, rawVal, opts...)
}

// ========== 配置对象方法 ==========

// NewConfig 创建新的配置对象
func NewConfig() *Config {
	return &Config{v: viper.New()}
}

// GetViper 获取内部的 viper 实例
func (c *Config) GetViper() *viper.Viper {
	return c.v
}

// GetString 获取字符串配置
func (c *Config) GetString(key string) string {
	return c.v.GetString(key)
}

// GetInt 获取整数配置
func (c *Config) GetInt(key string) int {
	return c.v.GetInt(key)
}

// GetBool 获取布尔配置
func (c *Config) GetBool(key string) bool {
	return c.v.GetBool(key)
}

// Get 获取配置值
func (c *Config) Get(key string) interface{} {
	return c.v.Get(key)
}

// Set 设置配置值
func (c *Config) Set(key string, value interface{}) {
	c.v.Set(key, value)
}

// ========== 配置生成 ==========

// GenerateDefaultConfig 生成默认配置文件
func GenerateDefaultConfig(path string) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 默认配置内容
	content := `# Admin Server 配置文件
# 服务器配置
server:
  host: 0.0.0.0
  port: 8080
  mode: debug  # debug, release, test
  read_timeout: 60
  write_timeout: 60

# 数据库配置
database:
  driver: mysql
  host: localhost
  port: 3306
  database: admin
  username: root
  password: ""
  charset: utf8mb4
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 3600

# 日志配置
log:
  level: info  # debug, info, warn, error, fatal
  format: console  # console, json
  output: ./logs/app.log
  console: true
  max_size: 100  # MB
  max_backups: 10
  max_age: 30  # days
  compress: true

# JWT配置
jwt:
  secret: your-secret-key-change-in-production
  expire: 86400  # seconds
  issuer: admin-server
`

	return os.WriteFile(path, []byte(content), 0644)
}
