package mnet

type Context interface {
	Reset()
	Next()
	Abort()
	Write(any)
	SetAgent(iface AgentIface)
	GetAgent() AgentIface
	SetHandler([]HandlerFunc)
	SetMsg(msg *Msg)
	GetMsg() Msg
}

type MContext struct {
	Msg     *Msg
	Agent   AgentIface
	index   int8
	handler []HandlerFunc
}

func NewMContext() Context {
	return &MContext{index: -1}
}

func (c *MContext) SetAgent(a AgentIface) {
	c.Agent = a
}

func (c *MContext) GetAgent() AgentIface {
	return c.Agent
}

func (c *MContext) SetMsg(msg *Msg) {
	c.Msg = msg
}

func (c *MContext) GetMsg() Msg {
	return *c.Msg
}

func (c *MContext) SetHandler(handler []HandlerFunc) {
	c.handler = handler
}

func (c *MContext) Reset() {
	c.Agent = nil
	c.index = -1
	c.Msg = nil
	c.handler = nil
}

func (c *MContext) Next() {
	c.index++
	for c.index < int8(len(c.handler)) {
		c.handler[c.index](c)
		c.index++
	}
}

func (c *MContext) Abort() {
	c.index = abortIndex
}

func (c *MContext) Write(msg any) {
	c.Agent.Write(msg)
}
