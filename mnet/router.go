package mnet

import (
	"fmt"
	"math"
	"reflect"
	"sync"
)

const (
	MsgType404 = "404"
	abortIndex = math.MaxInt8 >> 1
)

var _defaultMsg map[string]HandlerFunc

func init() {
	_defaultMsg = make(map[string]HandlerFunc)
}

func SetDefault404Msg(handle HandlerFunc) {
	_defaultMsg[MsgType404] = handle
}

type HandlerFunc func(c Context)

type Router struct {
	Middleware     []HandlerFunc
	handler        map[string][]HandlerFunc
	DefaultContext func() Context
	contextPool    sync.Pool
}

func NewRouter() *Router {
	r := &Router{
		handler: make(map[string][]HandlerFunc),
	}
	r.SetContext(NewMContext)
	return r
}

func (r *Router) SetContext(c func() Context) {
	r.DefaultContext = c
	r.contextPool.New = func() any {
		return r.DefaultContext()
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

func (r *Router) GetMsg(p any) string {
	pt := reflect.TypeOf(p)
	switch pt.Kind() {
	case reflect.Ptr:
		return pt.Elem().Name()
	case reflect.Struct:
		return pt.Name()
	default:
		return pt.String()
	}
}

func (r *Router) Route(msg *Msg, a AgentIface) {
	c := r.contextPool.Get().(Context)
	c.SetAgent(a)
	c.SetMsg(msg)

	defer func() {
		c.Reset()
		r.contextPool.Put(c)
	}()
	if r.handler == nil {
		m, o := _defaultMsg[MsgType404]
		if o {
			m(c)
		}
		return
	}
	c.SetHandler(r.handler[msg.Id])
	_, ok := r.handler[msg.Id]
	if !ok {
		m, o := _defaultMsg[MsgType404]
		if o {
			m(c)
		}
		return
	}
	c.Next()
}
