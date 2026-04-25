package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// 标准 CORS HTTP Header 常量
const (
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderVary                          = "Vary"
)

// DefaultCORSConfig 返回默认的 CORS 配置
// 默认允许所有来源，允许 GET/POST/PUT/DELETE/OPTIONS 方法，允许 Content-Type 和 Authorization 请求头
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
}

// CORSConfig CORS 跨域配置
type CORSConfig struct {
	AllowOrigins     []string // 允许的来源列表，["*"] 表示允许所有
	AllowMethods     []string // 允许的 HTTP 方法
	AllowHeaders     []string // 允许的请求头
	ExposeHeaders    []string // 允许浏览器访问的响应头
	AllowCredentials bool     // 是否允许携带凭证（Cookie/Authorization）
	MaxAge           int      // 预检请求缓存时间（秒）
}

// CORSOption CORS 配置选项函数
type CORSOption func(*CORSConfig)

// WithAllowOrigins 设置允许的来源
func WithAllowOrigins(origins ...string) CORSOption {
	return func(c *CORSConfig) {
		c.AllowOrigins = origins
	}
}

// WithAllowMethods 设置允许的方法
func WithAllowMethods(methods ...string) CORSOption {
	return func(c *CORSConfig) {
		c.AllowMethods = methods
	}
}

// WithAllowHeaders 设置允许的请求头
func WithAllowHeaders(headers ...string) CORSOption {
	return func(c *CORSConfig) {
		c.AllowHeaders = headers
	}
}

// WithExposeHeaders 设置暴露的响应头
func WithExposeHeaders(headers ...string) CORSOption {
	return func(c *CORSConfig) {
		c.ExposeHeaders = headers
	}
}

// WithAllowCredentials 设置是否允许凭证
func WithAllowCredentials(allow bool) CORSOption {
	return func(c *CORSConfig) {
		c.AllowCredentials = allow
	}
}

// WithMaxAge 设置预检缓存时间（秒）
func WithMaxAge(seconds int) CORSOption {
	return func(c *CORSConfig) {
		c.MaxAge = seconds
	}
}

// NewCORSConfig 创建 CORS 配置
func NewCORSConfig(opts ...CORSOption) *CORSConfig {
	cfg := DefaultCORSConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// isOriginAllowed 检查来源是否在允许列表中
func (cfg *CORSConfig) isOriginAllowed(origin string) bool {
	if len(cfg.AllowOrigins) == 0 {
		return false
	}
	for _, o := range cfg.AllowOrigins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}

// Handler 返回标准库的 http.Handler 中间件
// 用于非 Gin 框架的 HTTP 服务，或作为独立函数处理预检请求
func (cfg *CORSConfig) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.applyHeaders(w, r)

		// 预检请求直接返回 204
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if next != nil {
			next.ServeHTTP(w, r)
		}
	})
}

// GinMiddleware 返回 Gin 框架的中间件
func (cfg *CORSConfig) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg.applyHeaders(c.Writer, c.Request)

		// 预检请求直接返回 204，不再进入后续处理器
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// applyHeaders 统一设置 CORS 响应头
func (cfg *CORSConfig) applyHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}

	// 如果配置了特定来源白名单，且当前来源不在列表中，则不设置跨域头
	if !cfg.isOriginAllowed(origin) {
		return
	}

	// 当允许凭证时，不能将 Origin 设为 "*"，必须回显具体来源
	if cfg.AllowCredentials && origin == "*" {
		origin = r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
	}

	w.Header().Set(HeaderAccessControlAllowOrigin, origin)
	w.Header().Add(HeaderVary, "Origin")

	if len(cfg.AllowMethods) > 0 {
		w.Header().Set(HeaderAccessControlAllowMethods, strings.Join(cfg.AllowMethods, ","))
	}
	if len(cfg.AllowHeaders) > 0 {
		w.Header().Set(HeaderAccessControlAllowHeaders, strings.Join(cfg.AllowHeaders, ","))
	}
	if len(cfg.ExposeHeaders) > 0 {
		w.Header().Set(HeaderAccessControlExposeHeaders, strings.Join(cfg.ExposeHeaders, ","))
	}
	if cfg.AllowCredentials {
		w.Header().Set(HeaderAccessControlAllowCredentials, "true")
	}
	if cfg.MaxAge > 0 {
		w.Header().Set(HeaderAccessControlMaxAge, strconv.Itoa(cfg.MaxAge))
	}
}

// Cors 便捷函数：使用默认配置处理标准库请求
// 适用于独立调用场景，如独立设置响应头
func Cors(w http.ResponseWriter, r *http.Request) {
	DefaultCORSConfig().applyHeaders(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
	}
}

// GinCors 便捷函数：使用默认配置返回 Gin 中间件
func GinCors() gin.HandlerFunc {
	return DefaultCORSConfig().GinMiddleware()
}
