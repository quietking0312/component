package mconf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	// 全局 viper 实例
	Viper *viper.Viper
)

// Config 配置管理器
type Config struct {
	v *viper.Viper
}

// Init 初始化配置
func Init(configPath string) error {
	Viper = viper.New()

	// 设置默认值
	setDefaults()

	// 配置文件路径
	if configPath != "" {
		Viper.SetConfigFile(configPath)
	} else {
		// 默认配置文件路径
		Viper.SetConfigName("config")
		Viper.SetConfigType("yaml")
		Viper.AddConfigPath(".")
		Viper.AddConfigPath("./config")
		Viper.AddConfigPath("/etc/admin/")
	}

	// 读取环境变量
	Viper.SetEnvPrefix("ADMIN")
	Viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	Viper.AutomaticEnv()

	// 读取配置文件
	if err := Viper.ReadInConfig(); err != nil {
		// 配置文件不存在时忽略错误（使用默认值）
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("read config file failed: %w", err)
		}
	}

	return nil
}

// InitWithConfig 使用自定义配置初始化
func InitWithConfig(v *viper.Viper) {
	Viper = v
}

// setDefaults 设置默认值
func setDefaults() {
	// 服务器配置
	Viper.SetDefault("server.host", "0.0.0.0")
	Viper.SetDefault("server.port", 8080)
	Viper.SetDefault("server.mode", "debug")
	Viper.SetDefault("server.read_timeout", 60)
	Viper.SetDefault("server.write_timeout", 60)

	// 数据库配置
	Viper.SetDefault("database.driver", "mysql")
	Viper.SetDefault("database.host", "localhost")
	Viper.SetDefault("database.port", 3306)
	Viper.SetDefault("database.database", "admin")
	Viper.SetDefault("database.username", "root")
	Viper.SetDefault("database.password", "")
	Viper.SetDefault("database.charset", "utf8mb4")
	Viper.SetDefault("database.max_open_conns", 100)
	Viper.SetDefault("database.max_idle_conns", 10)

	// 日志配置
	Viper.SetDefault("log.level", "info")
	Viper.SetDefault("log.format", "console")
	Viper.SetDefault("log.output", "./logs/app.log")
	Viper.SetDefault("log.console", true)
	Viper.SetDefault("log.max_size", 100)
	Viper.SetDefault("log.max_backups", 10)
	Viper.SetDefault("log.max_age", 30)

	// JWT配置
	Viper.SetDefault("jwt.secret", "your-secret-key")
	Viper.SetDefault("jwt.expire", 86400)
	Viper.SetDefault("jwt.issuer", "admin-server")
}

// GetConfig 获取配置管理器实例
func GetConfig() *Config {
	return &Config{v: Viper}
}

// GetViper 获取 viper 实例
func GetViper() *viper.Viper {
	return Viper
}

// ========== 配置读取方法 ==========

// Get 获取配置值
func Get(key string) interface{} {
	return Viper.Get(key)
}

// GetString 获取字符串配置
func GetString(key string) string {
	return Viper.GetString(key)
}

// GetInt 获取整数配置
func GetInt(key string) int {
	return Viper.GetInt(key)
}

// GetInt32 获取 int32 配置
func GetInt32(key string) int32 {
	return Viper.GetInt32(key)
}

// GetInt64 获取 int64 配置
func GetInt64(key string) int64 {
	return Viper.GetInt64(key)
}

// GetUint 获取无符号整数配置
func GetUint(key string) uint {
	return Viper.GetUint(key)
}

// GetBool 获取布尔配置
func GetBool(key string) bool {
	return Viper.GetBool(key)
}

// GetFloat64 获取浮点数配置
func GetFloat64(key string) float64 {
	return Viper.GetFloat64(key)
}

// GetTime 获取时间配置
func GetTime(key string) interface{} {
	return Viper.GetTime(key)
}

// GetDuration 获取时长配置
func GetDuration(key string) interface{} {
	return Viper.GetDuration(key)
}

// GetStringSlice 获取字符串数组配置
func GetStringSlice(key string) []string {
	return Viper.GetStringSlice(key)
}

// GetIntSlice 获取整数数组配置
func GetIntSlice(key string) []int {
	return Viper.GetIntSlice(key)
}

// GetStringMap 获取 Map 配置
func GetStringMap(key string) map[string]interface{} {
	return Viper.GetStringMap(key)
}

// GetStringMapString 获取 Map[string]string 配置
func GetStringMapString(key string) map[string]string {
	return Viper.GetStringMapString(key)
}

// IsSet 检查配置是否存在
func IsSet(key string) bool {
	return Viper.IsSet(key)
}

// Sub 获取子配置
func Sub(key string) *viper.Viper {
	return Viper.Sub(key)
}

// AllSettings 获取所有配置
func AllSettings() map[string]interface{} {
	return Viper.AllSettings()
}

// ========== 配置设置方法 ==========

// Set 设置配置值
func Set(key string, value interface{}) {
	Viper.Set(key, value)
}

// SetDefault 设置默认值
func SetDefault(key string, value interface{}) {
	Viper.SetDefault(key, value)
}

// ========== 配置文件操作 ==========

// ReadInConfig 读取配置文件
func ReadInConfig() error {
	return Viper.ReadInConfig()
}

// WriteConfig 写入配置文件
func WriteConfig() error {
	return Viper.WriteConfig()
}

// SafeWriteConfig 安全写入配置文件（不存在时才写入）
func SafeWriteConfig() error {
	return Viper.SafeWriteConfig()
}

// WriteConfigAs 写入配置文件到指定路径
func WriteConfigAs(filename string) error {
	return Viper.WriteConfigAs(filename)
}

// SafeWriteConfigAs 安全写入配置文件到指定路径
func SafeWriteConfigAs(filename string) error {
	return Viper.SafeWriteConfigAs(filename)
}

// ========== 配置监听 ==========

// WatchConfig 监听配置文件变化
func WatchConfig() {
	Viper.WatchConfig()
}

// OnConfigChange 配置变化回调
func OnConfigChange(run func()) {
	Viper.OnConfigChange(func(in fsnotify.Event) {
		run()
	})
}

// OnConfigChangeEvent 配置变化回调（带事件信息）
func OnConfigChangeEvent(run func(event fsnotify.Event)) {
	Viper.OnConfigChange(func(in fsnotify.Event) {
		run(in)
	})
}

// ========== 辅助方法 ==========

// ConfigFileUsed 获取当前使用的配置文件路径
func ConfigFileUsed() string {
	return Viper.ConfigFileUsed()
}

// BindEnv 绑定环境变量
func BindEnv(input ...string) error {
	return Viper.BindEnv(input...)
}

// Unmarshal 反序列化配置到结构体
func Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return Viper.Unmarshal(rawVal, opts...)
}

// UnmarshalKey 反序列化指定 key 到结构体
func UnmarshalKey(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return Viper.UnmarshalKey(key, rawVal, opts...)
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
