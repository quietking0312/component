package mnet

type Context struct {
	Msg     *Msg
	Agent   AgentIface
	index   int8
	handler []HandlerFunc
}

func (c *Context) reset() {
	c.Agent = nil
	c.index = -1
	c.Msg = nil
}

func (c *Context) Next() {
	c.index++
	for c.index < int8(len(c.handler)) {
		c.handler[c.index](c)
		c.index++
	}
}

func (c *Context) Abort() {
	c.index = abortIndex
}

func (c *Context) Write(msg any) {
	c.Agent.Write(msg)
}
