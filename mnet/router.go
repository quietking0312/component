package mnet

import (
	"fmt"
	"math"
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

type HandlerFunc func(msg Msg, a AgentIface)

type Router struct {
	Agent      AgentIface
	Middleware []HandlerFunc
	handler    map[string][]HandlerFunc
	index      int8
	msg        *Msg
}

func NewRouter() *Router {
	return &Router{
		handler: make(map[string][]HandlerFunc),
		index:   -1,
	}
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
	r.Agent = a
	r.msg = msg
	r.Next()
	r.reset()
}

func (r *Router) reset() {
	r.Agent = nil
	r.index = -1
	r.msg = nil
}

func (r *Router) Next() {
	r.index++
	h, ok := r.handler[r.msg.Id]
	if !ok {
		return
	}
	for r.index < int8(len(h)) {
		h[r.index](*r.msg, r.Agent)
		r.index++
	}
}

func (r *Router) Abort() {
	r.index = abortIndex
}
