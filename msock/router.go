package msock

import (
	"sync"
)

// Router 消息路由器
type Router struct {
	mu       sync.RWMutex
	handlers map[uint32]Handler
	chains   []Middleware
	notFound Handler
}

// Middleware 中间件函数
type Middleware func(next Handler) Handler

// NewRouter 创建新的路由器
func NewRouter() *Router {
	return &Router{
		handlers: make(map[uint32]Handler),
		notFound: defaultNotFoundHandler,
	}
}

// Register 注册消息处理器
func (r *Router) Register(routeID uint32, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[routeID] = handler
}

// RegisterMultiple 批量注册消息处理器
func (r *Router) RegisterMultiple(handlers map[uint32]Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for routeID, handler := range handlers {
		r.handlers[routeID] = handler
	}
}

// Get 获取指定路由ID的处理器
func (r *Router) Get(routeID uint32) Handler {
	r.mu.RLock()
	handler, ok := r.handlers[routeID]
	r.mu.RUnlock()

	if !ok {
		return r.notFound
	}

	// 应用中间件链
	return r.applyMiddleware(handler)
}

// Remove 移除指定路由ID的处理器
func (r *Router) Remove(routeID uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, routeID)
}

// Use 添加中间件
func (r *Router) Use(mw ...Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chains = append(r.chains, mw...)
}

// SetNotFoundHandler 设置未找到路由时的处理器
func (r *Router) SetNotFoundHandler(handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notFound = handler
}

// applyMiddleware 将中间件链应用到处理器
func (r *Router) applyMiddleware(handler Handler) Handler {
	// 从后往前应用中间件
	for i := len(r.chains) - 1; i >= 0; i-- {
		handler = r.chains[i](handler)
	}
	return handler
}

// Handle 处理消息
func (r *Router) Handle(conn Conn, msg Message) {
	if msg == nil {
		return
	}
	handler := r.Get(msg.RouteID())
	handler(conn, msg)
}

// RouteCount 返回注册的路由数量
func (r *Router) RouteCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.handlers)
}

// defaultNotFoundHandler 默认的未找到路由处理器
func defaultNotFoundHandler(conn Conn, msg Message) {
	// 默认不处理，可以记录日志或发送错误响应
}

// Group 路由组
type Group struct {
	prefix uint32
	router *Router
	chains []Middleware
}

// Group 创建路由组，可以设置统一的前缀和中间件
// prefix: 路由ID前缀，组内所有路由ID都会与该前缀组合
func (r *Router) Group(prefix uint32, mws ...Middleware) *Group {
	return &Group{
		prefix: prefix,
		router: r,
		chains: mws,
	}
}

// Register 在组内注册消息处理器
// routeID: 组内的相对路由ID，最终路由ID = prefix << 16 | routeID
func (g *Group) Register(routeID uint32, handler Handler) {
	finalRouteID := (g.prefix << 16) | (routeID & 0xFFFF)

	// 应用组的中间件
	finalHandler := handler
	for i := len(g.chains) - 1; i >= 0; i-- {
		finalHandler = g.chains[i](finalHandler)
	}

	g.router.Register(finalRouteID, finalHandler)
}

// Use 为路由组添加中间件
func (g *Group) Use(mw ...Middleware) {
	g.chains = append(g.chains, mw...)
}

// Chain 创建中间件链
func Chain(mws ...Middleware) Middleware {
	return func(next Handler) Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}

// Recovery 恢复中间件，捕获 panic
func Recovery(logger Logger) Middleware {
	return func(next Handler) Handler {
		return func(conn Conn, msg Message) {
			defer func() {
				if err := recover(); err != nil {
					if logger != nil {
						connID := "nil"
						if conn != nil {
							connID = conn.ID()
						}
						logger.Errorf("panic recovered: %v, conn: %s", err, connID)
					}
				}
			}()
			next(conn, msg)
		}
	}
}

// Logging 日志中间件
func Logging(logger Logger) Middleware {
	return func(next Handler) Handler {
		return func(conn Conn, msg Message) {
			if logger != nil {
				logger.Debugf("recv msg, conn: %s, route: %d, size: %d",
					conn.ID(), msg.RouteID(), len(msg.Data()))
			}
			next(conn, msg)
		}
	}
}

// Auth 认证中间件示例（需要配合连接上下文使用）
func Auth(authFunc func(conn Conn) bool, logger Logger) Middleware {
	return func(next Handler) Handler {
		return func(conn Conn, msg Message) {
			if !authFunc(conn) {
				if logger != nil {
					logger.Warnf("auth failed, conn: %s", conn.ID())
				}
				return
			}
			next(conn, msg)
		}
	}
}

// Validate 消息校验中间件
func Validate(validateFunc func(msg Message) error, logger Logger) Middleware {
	return func(next Handler) Handler {
		return func(conn Conn, msg Message) {
			if err := validateFunc(msg); err != nil {
				if logger != nil {
					logger.Warnf("validate failed: %v, conn: %s", err, conn.ID())
				}
				return
			}
			next(conn, msg)
		}
	}
}

// RateLimit 限流中间件（简单计数器实现，生产环境建议使用令牌桶）
func RateLimit(maxRequests int, logger Logger) Middleware {
	type limiter struct {
		count int
		mu    sync.Mutex
	}

	limiters := make(map[string]*limiter)
	var limitersMu sync.RWMutex

	return func(next Handler) Handler {
		return func(conn Conn, msg Message) {
			connID := conn.ID()

			limitersMu.RLock()
			l, ok := limiters[connID]
			limitersMu.RUnlock()

			if !ok {
				l = &limiter{}
				limitersMu.Lock()
				limiters[connID] = l
				limitersMu.Unlock()
			}

			l.mu.Lock()
			l.count++
			current := l.count
			l.mu.Unlock()

			if current > maxRequests {
				if logger != nil {
					logger.Warnf("rate limit exceeded, conn: %s", connID)
				}
				return
			}

			next(conn, msg)
		}
	}
}

// Timeout 超时中间件
func Timeout(timeoutFn func(), logger Logger) Middleware {
	return func(next Handler) Handler {
		return func(conn Conn, msg Message) {
			done := make(chan struct{})
			go func() {
				defer close(done)
				next(conn, msg)
			}()

			select {
			case <-done:
			case <-conn.Context().Done():
				if logger != nil {
					logger.Warnf("handler timeout, conn: %s", conn.ID())
				}
				if timeoutFn != nil {
					timeoutFn()
				}
			}
		}
	}
}
