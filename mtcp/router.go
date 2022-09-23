package mtcp

type Router struct {
	r map[string]func(Msg)
}

func (router *Router) Call(head Head, msg []byte) {
	method := head.GetMethod()
	fn, ok := router.r[method]
	if ok {
		go fn(msg)
	}
}
