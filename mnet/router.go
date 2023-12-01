package mnet

import (
	"fmt"
	"math"
	"sync"
)

const (
	MsgType404 = "404"
	abortIndex = math.MaxInt8 >> 1
)

var _defaultMsg map[string]any

func init() {
	_defaultMsg = make(map[string]any)
}

func SetDefault404Msg(msg any) {
	_defaultMsg[MsgType404] = msg
}

type HandlerFunc func(c *Context)

type Router struct {
	Middleware  []HandlerFunc
	handler     map[string][]HandlerFunc
	contextPool sync.Pool
}

func NewRouter() *Router {
	r := &Router{
		handler: make(map[string][]HandlerFunc),
	}
	r.contextPool.New = func() any {
		return r.allocateContext()
	}
	return r
}

func (r *Router) allocateContext() *Context {
	return &Context{index: -1}
}

func (r *Router) Use(fc ...HandlerFunc) {
	r.Middleware = append(r.Middleware, fc...)
}

func (r *Router) Register(path string, fc ...HandlerFunc) {
	_, ok := r.handler[path]
	if ok {
		panic(fmt.Sprintf("path: %s is exists", path))
	}
	r.handler[path] = append(r.Middleware, fc...)
}

func (r *Router) Route(msg *Msg, a AgentIface) {
	if r.handler == nil {
		m, o := _defaultMsg[MsgType404]
		if o {
			a.Write(m)
		}
		return
	}
	_, ok := r.handler[msg.Id]
	if !ok {
		m, o := _defaultMsg[MsgType404]
		if o {
			a.Write(m)
		}
		return
	}
	c := r.contextPool.Get().(*Context)
	c.Agent = a
	c.Msg = msg
	c.handler = r.handler[msg.Id]
	c.Next()
	c.reset()
	r.contextPool.Put(c)
}
