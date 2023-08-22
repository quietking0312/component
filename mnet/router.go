package mnet

import "fmt"

const (
	MsgType404 = "404"
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
	Agent     AgentIface
	routeFlag bool
	handler   map[string][]HandlerFunc
}

func NewRouter() *Router {
	return &Router{
		handler: make(map[string][]HandlerFunc),
	}
}

func (r *Router) SetAgent(a AgentIface) {
	r.Agent = a
}

func (r *Router) Register(path string, fc ...HandlerFunc) {
	_, ok := r.handler[path]
	if ok {
		panic(fmt.Sprintf("path: %s is exists", path))
	}
	r.handler[path] = fc
}

func (r *Router) Route(msg *Msg) {
	if r.handler == nil {
		m, o := _defaultMsg[MsgType404]
		if o {
			r.Agent.Write(m)
		}
		return
	}
	handles, ok := r.handler[msg.Id]
	if !ok {
		m, o := _defaultMsg[MsgType404]
		if o {
			r.Agent.Write(m)
		}
		return
	}
	r.routeFlag = true
	for _, fc := range handles {
		fc(*msg, r.Agent)
		if !r.routeFlag {
			break
		}
	}
}
func (r *Router) Next() {
	r.routeFlag = true
}

func (r *Router) About() {
	r.routeFlag = false
}
